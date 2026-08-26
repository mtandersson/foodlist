package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const validRecipeOutput = `{"title":"Soppa","description":"4 portioner","sections":[{"name":"","ingredients":[{"amount":1,"unit":"l","name":"vatten"}],"instructions":["Koka."]}]}`

func TestOpenAIResponsesRecipeClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		require.Equal(t, "application/json", r.Header.Get("Accept"))
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "gpt-test", body["model"])
		require.Equal(t, false, body["stream"])
		require.Equal(t, false, body["store"])
		require.Equal(t, float64(responsesMaxOutputTokens), body["max_output_tokens"])
		text := body["text"].(map[string]any)
		format := text["format"].(map[string]any)
		require.Equal(t, "json_schema", format["type"])
		response := responsesEnvelope{Status: "completed", Output: []responsesOutputItem{{
			Type: "message", Content: []responsesOutputContent{{Type: "output_text", Text: validRecipeOutput}},
		}}}
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	defer server.Close()

	parser, err := NewOpenAIResponsesRecipeClient("test-key", "gpt-test")
	require.NoError(t, err)
	client := parser.(*responsesRecipeClient)
	client.endpoint = server.URL
	client.http = server.Client()

	recipe, err := client.ParseImage(context.Background(), []byte("image"), "image/jpeg")
	require.NoError(t, err)
	require.Equal(t, "Soppa", recipe.Title)
}

func TestCodexResponsesRecipeClientStreamsAndSetsHeaders(t *testing.T) {
	dataDir := t.TempDir()
	secretsDir := filepath.Join(dataDir, "secrets")
	require.NoError(t, os.Mkdir(secretsDir, 0o700))
	authPath := filepath.Join(secretsDir, "auth.json")
	accessToken := jwtForTest("acct-1", time.Now().Add(time.Hour))
	writeCodexTestAuth(t, authPath, accessToken, "refresh-1", "acct-1")

	manager, err := NewCodexOAuthManager(authPath, dataDir)
	require.NoError(t, err)
	defer manager.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer "+accessToken, r.Header.Get("Authorization"))
		require.Equal(t, "acct-1", r.Header.Get("ChatGPT-Account-ID"))
		require.Equal(t, "codex_cli_rs", r.Header.Get("originator"))
		require.Contains(t, r.Header.Get("User-Agent"), "FoodList/test-version")
		require.Equal(t, "text/event-stream", r.Header.Get("Accept"))
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.NotContains(t, body, "text")
		require.NotContains(t, body, "max_output_tokens")
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprintf(w, "data: {\"type\":\"response.output_text.delta\",\"delta\":%q}\n\n", validRecipeOutput[:40])
		_, _ = fmt.Fprintf(w, "data: {\"type\":\"response.output_text.delta\",\"delta\":%q}\n\n", validRecipeOutput[40:])
		_, _ = fmt.Fprint(w, "data: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\"}}\n\ndata: [DONE]\n\n")
	}))
	defer server.Close()

	parser, err := NewCodexOAuthRecipeClient(manager, "gpt-5.6-sol", "test-version")
	require.NoError(t, err)
	client := parser.(*responsesRecipeClient)
	client.endpoint = server.URL
	client.http = server.Client()

	recipe, err := client.ParseImage(context.Background(), []byte("image"), "image/jpeg")
	require.NoError(t, err)
	require.Equal(t, "Soppa", recipe.Title)
}

func TestParseResponsesSSEFailures(t *testing.T) {
	_, err := parseResponsesSSE(strings.NewReader("data: {\"type\":\"response.output_text.delta\",\"delta\":\"x\"}\n\n"))
	require.ErrorContains(t, err, "before completion")

	_, err = parseResponsesSSE(strings.NewReader("data: {\"type\":\"response.failed\"}\n\n"))
	require.Error(t, err)

	_, err = parseResponsesSSE(strings.NewReader("data: not-json\n\n"))
	require.Error(t, err)

	_, err = parseResponsesSSE(strings.NewReader("data: {\"type\":\"response.output_text.delta\",\"delta\":\"{}\"}\n\ndata: [DONE]\n\n"))
	require.ErrorContains(t, err, "before completion")

	_, err = parseResponsesSSE(strings.NewReader("data: {\"type\":\"error\",\"code\":\"usage_limit_reached\"}\n\n"))
	var providerErr *responsesProviderError
	require.ErrorAs(t, err, &providerErr)
	require.Equal(t, "usage_limit_reached", providerErr.code)
	client := &responsesRecipeClient{}
	llmErr, ok := asRecipeLLMError(client.classifyError(err))
	require.True(t, ok)
	require.Equal(t, LLMErrorQuota, llmErr.Kind)
}

func TestCodexResponsesRetriesOnceAfterUnauthorized(t *testing.T) {
	dataDir := t.TempDir()
	secretsDir := filepath.Join(dataDir, "secrets")
	require.NoError(t, os.Mkdir(secretsDir, 0o700))
	authPath := filepath.Join(secretsDir, "auth.json")
	oldAccess := jwtForTest("acct", time.Now().Add(time.Hour))
	newAccess := jwtForTest("acct", time.Now().Add(2*time.Hour))
	writeCodexTestAuth(t, authPath, oldAccess, "refresh", "acct")

	refreshCalls := 0
	refreshServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		refreshCalls++
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"access_token": newAccess, "refresh_token": "rotated"}))
	}))
	defer refreshServer.Close()
	manager, err := newCodexOAuthManager(authPath, dataDir, refreshServer.Client())
	require.NoError(t, err)
	defer manager.Close()
	manager.tokenURL = refreshServer.URL

	responseCalls := 0
	responseServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		responseCalls++
		if responseCalls == 1 {
			require.Equal(t, "Bearer "+oldAccess, r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		require.Equal(t, "Bearer "+newAccess, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprintf(w, "data: {\"type\":\"response.output_text.delta\",\"delta\":%q}\n\n", validRecipeOutput)
		_, _ = fmt.Fprint(w, "data: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\"}}\n\n")
	}))
	defer responseServer.Close()
	parser, err := NewCodexOAuthRecipeClient(manager, "gpt-5.6-sol", "test")
	require.NoError(t, err)
	client := parser.(*responsesRecipeClient)
	client.endpoint = responseServer.URL
	client.http = responseServer.Client()
	_, err = client.ParseImage(context.Background(), []byte("image"), "image/jpeg")
	require.NoError(t, err)
	require.Equal(t, 2, responseCalls)
	require.Equal(t, 1, refreshCalls)
}
