package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func jwtForTest(account string, expires time.Time) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	claims, _ := json.Marshal(map[string]any{
		"exp":                         expires.Unix(),
		"https://api.openai.com/auth": map[string]any{"chatgpt_account_id": account},
	})
	payload := base64.RawURLEncoding.EncodeToString(claims)
	return header + "." + payload + ".signature"
}

func writeCodexTestAuth(t *testing.T, path, access, refresh, account string) {
	t.Helper()
	doc := map[string]any{
		"auth_mode":    "chatgpt",
		"last_refresh": "2026-01-01T00:00:00Z",
		"future_field": map[string]any{"preserve": true},
		"tokens": map[string]any{
			"access_token":       access,
			"refresh_token":      refresh,
			"account_id":         account,
			"id_token":           "keep-me",
			"future_token_field": "keep-too",
		},
	}
	data, err := json.Marshal(doc)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o600))
}

func TestCodexOAuthManagerRefreshesAndPersistsRotation(t *testing.T) {
	dataDir := t.TempDir()
	secretsDir := filepath.Join(dataDir, "secrets")
	require.NoError(t, os.Mkdir(secretsDir, 0o700))
	authPath := filepath.Join(secretsDir, "auth.json")
	writeCodexTestAuth(t, authPath, jwtForTest("acct", time.Now().Add(-time.Minute)), "old-refresh", "acct")

	newAccess := jwtForTest("acct", time.Now().Add(time.Hour))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		require.Equal(t, "refresh_token", r.Form.Get("grant_type"))
		require.Equal(t, "old-refresh", r.Form.Get("refresh_token"))
		require.Equal(t, codexOAuthClientID, r.Form.Get("client_id"))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  newAccess,
			"refresh_token": "rotated-refresh",
		})
	}))
	defer server.Close()

	manager, err := newCodexOAuthManager(authPath, dataDir, server.Client())
	require.NoError(t, err)
	defer manager.Close()
	manager.tokenURL = server.URL

	creds, err := manager.Credentials(context.Background(), false)
	require.NoError(t, err)
	require.Equal(t, newAccess, creds.AccessToken)
	require.Equal(t, "acct", creds.AccountID)

	var saved map[string]any
	data, err := os.ReadFile(authPath)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &saved))
	tokens := saved["tokens"].(map[string]any)
	require.Equal(t, "rotated-refresh", tokens["refresh_token"])
	require.Equal(t, "keep-me", tokens["id_token"])
	require.Equal(t, "keep-too", tokens["future_token_field"])
	require.Equal(t, true, saved["future_field"].(map[string]any)["preserve"])
	info, err := os.Stat(authPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestCodexOAuthManagerRejectsAccountChange(t *testing.T) {
	dataDir := t.TempDir()
	secretsDir := filepath.Join(dataDir, "secrets")
	require.NoError(t, os.Mkdir(secretsDir, 0o700))
	authPath := filepath.Join(secretsDir, "auth.json")
	writeCodexTestAuth(t, authPath, jwtForTest("acct-a", time.Now().Add(-time.Minute)), "refresh", "acct-a")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  jwtForTest("acct-b", time.Now().Add(time.Hour)),
			"refresh_token": "rotated",
		})
	}))
	defer server.Close()
	manager, err := newCodexOAuthManager(authPath, dataDir, server.Client())
	require.NoError(t, err)
	defer manager.Close()
	manager.tokenURL = server.URL

	_, err = manager.Credentials(context.Background(), false)
	llmErr, ok := asRecipeLLMError(err)
	require.True(t, ok)
	require.Equal(t, LLMErrorAuth, llmErr.Kind)
}

func TestCodexOAuthManagerExclusiveOwnership(t *testing.T) {
	dataDir := t.TempDir()
	secretsDir := filepath.Join(dataDir, "secrets")
	require.NoError(t, os.Mkdir(secretsDir, 0o700))
	authPath := filepath.Join(secretsDir, "auth.json")
	writeCodexTestAuth(t, authPath, jwtForTest("acct", time.Now().Add(time.Hour)), "refresh", "acct")

	first, err := NewCodexOAuthManager(authPath, dataDir)
	require.NoError(t, err)
	defer first.Close()
	second, err := NewCodexOAuthManager(authPath, dataDir)
	require.Error(t, err)
	require.Nil(t, second)
}

func TestValidateCodexAuthPathSecurity(t *testing.T) {
	dataDir := t.TempDir()
	secretsDir := filepath.Join(dataDir, "secrets")
	require.NoError(t, os.Mkdir(secretsDir, 0o700))
	authPath := filepath.Join(secretsDir, "auth.json")
	writeCodexTestAuth(t, authPath, jwtForTest("acct", time.Now().Add(time.Hour)), "refresh", "acct")

	_, err := validateCodexAuthPath(authPath, dataDir)
	require.NoError(t, err)

	require.NoError(t, os.Chmod(authPath, 0o644))
	_, err = validateCodexAuthPath(authPath, dataDir)
	require.Error(t, err)
	require.NoError(t, os.Chmod(authPath, 0o600))

	outside := filepath.Join(t.TempDir(), "auth.json")
	writeCodexTestAuth(t, outside, jwtForTest("acct", time.Now().Add(time.Hour)), "refresh", "acct")
	_, err = validateCodexAuthPath(outside, dataDir)
	require.Error(t, err)

	link := filepath.Join(secretsDir, "link.json")
	require.NoError(t, os.Symlink(authPath, link))
	_, err = validateCodexAuthPath(link, dataDir)
	require.Error(t, err)

	outsideDir := t.TempDir()
	require.NoError(t, os.Chmod(outsideDir, 0o700))
	outsideAuth := filepath.Join(outsideDir, "auth.json")
	writeCodexTestAuth(t, outsideAuth, jwtForTest("acct", time.Now().Add(time.Hour)), "refresh", "acct")
	linkedParent := filepath.Join(dataDir, "linked-secrets")
	require.NoError(t, os.Symlink(outsideDir, linkedParent))
	_, err = validateCodexAuthPath(filepath.Join(linkedParent, "auth.json"), dataDir)
	require.Error(t, err)
}

func TestNoRedirectHTTPClient(t *testing.T) {
	client := noRedirectHTTPClient(time.Second)
	require.NotNil(t, client.CheckRedirect)
	require.ErrorIs(t, client.CheckRedirect(nil, nil), http.ErrUseLastResponse)
}

func TestCodexAccessTokenMetadataMalformed(t *testing.T) {
	_, _, err := codexAccessTokenMetadata("not-a-jwt")
	require.Error(t, err)
	_, _, err = codexAccessTokenMetadata(fmt.Sprintf("x.%s.y", base64.RawURLEncoding.EncodeToString([]byte(`{}`))))
	require.Error(t, err)
}
