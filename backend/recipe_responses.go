package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

const (
	openAIResponsesEndpoint  = "https://api.openai.com/v1/responses"
	codexResponsesEndpoint   = "https://chatgpt.com/backend-api/codex/responses"
	responsesMaxOutputTokens = 8192
)

// This schema mirrors rawRecipeResponse and the storage-side limits. Local
// ValidateAndNormalize remains authoritative even when an upstream claims to
// enforce structured output.
const recipeResponsesJSONSchema = `{
  "type":"object",
  "additionalProperties":false,
  "required":["title","description","sections"],
  "properties":{
    "title":{"type":"string","maxLength":200},
    "description":{"type":"string","maxLength":4000},
    "sections":{
      "type":"array","maxItems":10,
      "items":{
        "type":"object","additionalProperties":false,
        "required":["name","ingredients","instructions"],
        "properties":{
          "name":{"type":"string","maxLength":2000},
          "ingredients":{
            "type":"array","maxItems":50,
            "items":{
              "type":"object","additionalProperties":false,
              "required":["amount","unit","name"],
              "properties":{
                "amount":{"anyOf":[{"type":"number"},{"type":"null"}]},
                "unit":{"type":"string","maxLength":2000},
                "name":{"type":"string","maxLength":2000}
              }
            }
          },
          "instructions":{"type":"array","maxItems":50,"items":{"type":"string","maxLength":2000}}
        }
      }
    }
  }
}`

type responsesRecipeClient struct {
	endpoint string
	apiKey   string
	model    string
	stream   bool
	provider string
	version  string
	oauth    *CodexOAuthManager
	http     *http.Client
}

func NewOpenAIResponsesRecipeClient(apiKey, model string) (RecipeImageParser, error) {
	if strings.TrimSpace(apiKey) == "" || strings.TrimSpace(model) == "" {
		return nil, ErrLLMConfigInvalid
	}
	return &responsesRecipeClient{
		endpoint: openAIResponsesEndpoint,
		apiKey:   apiKey,
		model:    model,
		provider: "openai_api",
		http:     noRedirectHTTPClient(recipeLLMTimeout),
	}, nil
}

func NewCodexOAuthRecipeClient(auth *CodexOAuthManager, model, appVersion string) (RecipeImageParser, error) {
	if auth == nil || strings.TrimSpace(model) == "" {
		return nil, ErrLLMConfigInvalid
	}
	return &responsesRecipeClient{
		endpoint: codexResponsesEndpoint,
		model:    model,
		stream:   true,
		provider: "experimental_codex_oauth",
		version:  appVersion,
		oauth:    auth,
		http:     noRedirectHTTPClient(recipeLLMTimeout),
	}, nil
}

type responsesRequest struct {
	Model           string                  `json:"model"`
	Instructions    string                  `json:"instructions"`
	Input           []responsesInputMessage `json:"input"`
	Text            *responsesText          `json:"text,omitempty"`
	Reasoning       responsesReasoning      `json:"reasoning"`
	MaxOutputTokens *int                    `json:"max_output_tokens,omitempty"`
	Store           bool                    `json:"store"`
	Stream          bool                    `json:"stream"`
}

type responsesInputMessage struct {
	Role    string                  `json:"role"`
	Content []responsesInputContent `json:"content"`
}

type responsesInputContent struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

type responsesText struct {
	Format responsesTextFormat `json:"format"`
}

type responsesTextFormat struct {
	Type   string          `json:"type"`
	Name   string          `json:"name"`
	Strict bool            `json:"strict"`
	Schema json.RawMessage `json:"schema"`
}

type responsesReasoning struct {
	Effort string `json:"effort"`
}

type responsesEnvelope struct {
	Status string                `json:"status"`
	Output []responsesOutputItem `json:"output"`
	Error  *struct {
		Code string `json:"code"`
	} `json:"error,omitempty"`
}

type responsesOutputItem struct {
	Type    string                   `json:"type"`
	Content []responsesOutputContent `json:"content"`
}

type responsesOutputContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type responsesStreamEvent struct {
	Type     string             `json:"type"`
	Code     string             `json:"code,omitempty"`
	Delta    string             `json:"delta"`
	Response *responsesEnvelope `json:"response,omitempty"`
	Error    *struct {
		Code string `json:"code"`
	} `json:"error,omitempty"`
}

type responsesHTTPError struct {
	status     int
	retryAfter string
}

type responsesProviderError struct{ code string }

func (e *responsesProviderError) Error() string { return "responses provider error" }

func (e *responsesHTTPError) Error() string { return fmt.Sprintf("responses status %d", e.status) }

func (c *responsesRecipeClient) ParseImage(ctx context.Context, imageBytes []byte, mime string) (Recipe, error) {
	if _, ok := allowedImageMimes[mime]; !ok {
		return Recipe{}, ErrUnsupportedImage
	}

	dataURL := "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(imageBytes)
	maxOutputTokens := responsesMaxOutputTokens
	body := responsesRequest{
		Model:        c.model,
		Instructions: recipeSystemPrompt(),
		Input: []responsesInputMessage{{
			Role: "user",
			Content: []responsesInputContent{
				{Type: "input_text", Text: recipeUserPrompt},
				{Type: "input_image", ImageURL: dataURL, Detail: "high"},
			},
		}},
		Text: &responsesText{Format: responsesTextFormat{
			Type: "json_schema", Name: "recipe", Strict: true,
			Schema: json.RawMessage(recipeResponsesJSONSchema),
		}},
		Reasoning:       responsesReasoning{Effort: "low"},
		MaxOutputTokens: &maxOutputTokens,
		Store:           false,
		Stream:          c.stream,
	}
	// The native ChatGPT/Codex endpoint rejects structured text.format and
	// max_output_tokens. Keep strict schema enforcement on the official API;
	// OAuth mode relies on the JSON-only prompt plus local validation.
	if c.oauth != nil {
		body.Text = nil
		body.MaxOutputTokens = nil
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return Recipe{}, newRecipeLLMError(LLMErrorInvalid, err)
	}

	output, err := c.do(ctx, bodyBytes, false)
	if err != nil && c.oauth != nil {
		var httpErr *responsesHTTPError
		if errors.As(err, &httpErr) && (httpErr.status == http.StatusUnauthorized || httpErr.status == http.StatusForbidden) {
			output, err = c.do(ctx, bodyBytes, true)
		}
	}
	if err != nil {
		return Recipe{}, c.classifyError(err)
	}
	return decodeRecipeOutput(output)
}

func (c *responsesRecipeClient) do(ctx context.Context, body []byte, forceRefresh bool) (string, error) {
	reqCtx, cancel := context.WithTimeout(ctx, recipeLLMTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.stream {
		req.Header.Set("Accept", "text/event-stream")
	} else {
		req.Header.Set("Accept", "application/json")
	}

	if c.oauth != nil {
		creds, err := c.oauth.Credentials(reqCtx, forceRefresh)
		if err != nil {
			return "", err
		}
		req.Header.Set("Authorization", "Bearer "+creds.AccessToken)
		req.Header.Set("ChatGPT-Account-ID", creds.AccountID)
		req.Header.Set("originator", "codex_cli_rs")
		version := strings.TrimSpace(c.version)
		if version == "" {
			version = "dev"
		}
		req.Header.Set("User-Agent", "codex_cli_rs/0.0.0 (FoodList/"+version+")")
	} else {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	slog.Info("recipe responses result",
		"provider", c.provider,
		"model", c.model,
		"status", resp.StatusCode,
		"response_content_type", resp.Header.Get("Content-Type"),
	)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Drain only a bounded amount for connection reuse. Never log or return it.
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 8*1024))
		return "", &responsesHTTPError{status: resp.StatusCode, retryAfter: resp.Header.Get("Retry-After")}
	}

	limited := io.LimitReader(resp.Body, recipeLLMMaxRespBytes+1)
	if c.stream {
		return parseResponsesSSE(limited)
	}
	data, err := io.ReadAll(limited)
	if err != nil {
		return "", err
	}
	if len(data) > recipeLLMMaxRespBytes {
		return "", fmt.Errorf("responses body too large")
	}
	var envelope responsesEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return "", err
	}
	return outputTextFromEnvelope(envelope)
}

func (c *responsesRecipeClient) classifyError(err error) error {
	if llmErr, ok := asRecipeLLMError(err); ok {
		return llmErr
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return newRecipeLLMError(LLMErrorTimeout, err)
	}
	var httpErr *responsesHTTPError
	if errors.As(err, &httpErr) {
		switch httpErr.status {
		case http.StatusUnauthorized, http.StatusForbidden:
			return newRecipeLLMError(LLMErrorAuth, err)
		case http.StatusTooManyRequests:
			return newRecipeLLMQuotaError(httpErr.retryAfter)
		default:
			return newRecipeLLMError(LLMErrorUpstream, err)
		}
	}
	var providerErr *responsesProviderError
	if errors.As(err, &providerErr) {
		code := strings.ToLower(providerErr.code)
		switch {
		case strings.Contains(code, "quota"), strings.Contains(code, "rate_limit"), strings.Contains(code, "usage_limit"):
			return newRecipeLLMQuotaError("")
		case strings.Contains(code, "auth"), strings.Contains(code, "token"), strings.Contains(code, "api_key"), strings.Contains(code, "account"):
			return newRecipeLLMError(LLMErrorAuth, err)
		default:
			return newRecipeLLMError(LLMErrorUpstream, err)
		}
	}
	return newRecipeLLMError(LLMErrorUpstream, err)
}

func decodeRecipeOutput(output string) (Recipe, error) {
	if strings.TrimSpace(output) == "" {
		return Recipe{}, newRecipeLLMError(LLMErrorInvalid, errors.New("empty model output"))
	}
	var raw rawRecipeResponse
	if err := json.Unmarshal([]byte(output), &raw); err != nil {
		return Recipe{}, newRecipeLLMError(LLMErrorInvalid, err)
	}
	cleaned, err := ValidateAndNormalize(rawToRecipe(raw))
	if err != nil {
		return Recipe{}, newRecipeLLMError(LLMErrorInvalid, err)
	}
	return cleaned, nil
}

func outputTextFromEnvelope(envelope responsesEnvelope) (string, error) {
	if envelope.Error != nil {
		return "", &responsesProviderError{code: envelope.Error.Code}
	}
	if envelope.Status == "failed" || envelope.Status == "incomplete" {
		return "", errors.New("responses request did not complete")
	}
	var out strings.Builder
	for _, item := range envelope.Output {
		for _, content := range item.Content {
			if content.Type == "output_text" && content.Text != "" {
				out.WriteString(content.Text)
			}
		}
	}
	if out.Len() == 0 {
		return "", errors.New("responses output text missing")
	}
	return out.String(), nil
}

func parseResponsesSSE(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), recipeLLMMaxRespBytes+1)
	var dataLines []string
	var deltas strings.Builder
	var completed *responsesEnvelope
	seenTerminal := false

	process := func() error {
		if len(dataLines) == 0 {
			return nil
		}
		data := strings.Join(dataLines, "\n")
		dataLines = dataLines[:0]
		if data == "[DONE]" {
			return nil
		}
		var event responsesStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return fmt.Errorf("decode responses event: %w", err)
		}
		switch event.Type {
		case "response.output_text.delta":
			deltas.WriteString(event.Delta)
		case "response.completed":
			seenTerminal = true
			completed = event.Response
		case "response.failed", "response.incomplete", "error":
			code := event.Code
			if event.Error != nil {
				code = event.Error.Code
			} else if event.Response != nil && event.Response.Error != nil {
				code = event.Response.Error.Code
			}
			return &responsesProviderError{code: code}
		}
		return nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if err := process(); err != nil {
				return "", err
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	if err := process(); err != nil {
		return "", err
	}
	if !seenTerminal {
		return "", errors.New("responses stream ended before completion")
	}
	if deltas.Len() > 0 {
		return deltas.String(), nil
	}
	if completed != nil {
		return outputTextFromEnvelope(*completed)
	}
	return "", errors.New("responses stream output missing")
}

var _ RecipeImageParser = (*responsesRecipeClient)(nil)
