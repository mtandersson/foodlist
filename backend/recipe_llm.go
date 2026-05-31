package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrLLMConfigInvalid is returned when env-supplied LLM config is unusable.
var ErrLLMConfigInvalid = errors.New("recipe llm config invalid")

// ErrLLMResponse signals the upstream LLM produced an unusable response.
// Callers may map this to HTTP 422 with a generic message.
var ErrLLMResponse = errors.New("recipe llm response invalid")

const (
	recipeLLMTimeout       = 60 * time.Second
	recipeLLMMaxRespBytes  = 256 * 1024
	recipeLLMRequestSchema = `{
  "title": "string (required, <= 200 chars)",
  "description": "string (optional, light markdown: **fet**, *kursiv*, listor, citat, länkar, ### och mindre rubriker, inline-kod; INTE råa HTML-taggar, bilder, eller # / ##)",
  "sections": [
    {
      "name": "string (\"\" för enkla recept; annars rubriker som 'Sås', 'Sallad')",
      "ingredients": [
        {"amount": "number or null", "unit": "short string", "name": "string (required)"}
      ],
      "instructions": ["string", "..."]
    }
  ]
}`
)

// RecipeLLMClient calls an OpenAI-compatible chat completions endpoint with
// vision input to extract structured recipe data from an image.
//
// The base URL is configured at startup from RECIPE_LLM_BASE_URL and is
// never user-supplied (SSRF rule: outbound HTTP destinations are not
// influenced by request data). Only http(s) schemes are accepted.
type RecipeLLMClient struct {
	baseURL string
	apiKey  string
	model   string
	http    *http.Client
}

// NewRecipeLLMClient validates the configured base URL and returns a
// client ready to call the LLM. An empty baseURL/apiKey/model returns
// ErrLLMConfigInvalid so the caller can disable the feature flag.
func NewRecipeLLMClient(baseURL, apiKey, model string) (*RecipeLLMClient, error) {
	if baseURL == "" || apiKey == "" || model == "" {
		return nil, ErrLLMConfigInvalid
	}
	u, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("%w: parse url: %v", ErrLLMConfigInvalid, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("%w: only http(s) schemes allowed", ErrLLMConfigInvalid)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("%w: empty host", ErrLLMConfigInvalid)
	}
	return &RecipeLLMClient{
		baseURL: u.String(),
		apiKey:  apiKey,
		model:   model,
		http:    &http.Client{Timeout: recipeLLMTimeout},
	}, nil
}

// Model returns the configured model name (used for logging).
func (c *RecipeLLMClient) Model() string { return c.model }

// llmChatRequest mirrors the OpenAI chat-completions schema with vision
// input. Only the fields the server actually populates are included.
type llmChatRequest struct {
	Model          string         `json:"model"`
	Messages       []llmMessage   `json:"messages"`
	Temperature    float64        `json:"temperature"`
	MaxTokens      int            `json:"max_tokens"`
	ResponseFormat *llmRespFormat `json:"response_format,omitempty"`
}

type llmRespFormat struct {
	Type string `json:"type"`
}

type llmMessage struct {
	Role    string              `json:"role"`
	Content []llmMessageContent `json:"content"`
}

type llmMessageContent struct {
	Type     string       `json:"type"`
	Text     string       `json:"text,omitempty"`
	ImageURL *llmImageURL `json:"image_url,omitempty"`
}

type llmImageURL struct {
	URL string `json:"url"`
}

type llmChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// rawRecipeResponse is the JSON shape the model is asked to emit.
// Only the sectioned shape is accepted; rawToRecipe maps it straight
// to a Recipe that ValidateAndNormalize will then trim and bounds-check.
type rawRecipeResponse struct {
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Sections    []rawRecipeSection `json:"sections"`
}

type rawRecipeSection struct {
	Name         string          `json:"name"`
	Ingredients  []rawIngredient `json:"ingredients"`
	Instructions []string        `json:"instructions"`
}

type rawIngredient struct {
	Amount *float64 `json:"amount"`
	Unit   string   `json:"unit"`
	Name   string   `json:"name"`
}

func rawToRecipe(raw rawRecipeResponse) Recipe {
	sections := make([]RecipeSection, 0, len(raw.Sections))
	for _, s := range raw.Sections {
		ings := make([]Ingredient, 0, len(s.Ingredients))
		for _, ing := range s.Ingredients {
			ings = append(ings, Ingredient(ing))
		}
		sections = append(sections, RecipeSection{
			Name:         s.Name,
			Ingredients:  ings,
			Instructions: s.Instructions,
		})
	}
	return Recipe{
		Title:       raw.Title,
		Description: raw.Description,
		Sections:    sections,
	}
}

// ParseImage uploads imageBytes to the LLM and returns the validated
// recipe (without ID/timestamps; caller fills those). The function never
// logs API key, request body, or response body; only model, image size,
// and HTTP status are emitted.
func (c *RecipeLLMClient) ParseImage(ctx context.Context, imageBytes []byte, mime string) (Recipe, error) {
	if _, ok := allowedImageMimes[mime]; !ok {
		return Recipe{}, ErrUnsupportedImage
	}

	dataURL := "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(imageBytes)

	systemPrompt := strings.Join([]string{
		"Du är en assistent som tolkar recept från en bild.",
		"Svara på svenska. Använd metriska enheter (dl, ml, l, g, kg, st, msk, tsk, krm).",
		"Returnera ENDAST giltig JSON i exakt detta format (utan kommentarer):",
		recipeLLMRequestSchema,
		"Om bilden inte är ett recept, returnera ett tomt title och en tom sections-array.",
		fmt.Sprintf("Maximalt %d ingredienser och %d instruktioner TOTALT över alla sektioner. Varje sträng <= %d tecken.", maxRecipeIngredients, maxRecipeInstructions, maxRecipeStringLen),
		"Beskrivning (`description`): extrahera intro-text, portioner, tillagningstid och källa/byline. Behåll radbrytningar och stycken. Om receptet saknar dessa, lämna `description` tom (`\"\"`). Använd markdown bara när källan har formatering att bevara. Tillåtna markdown-element: **fet**, *kursiv*, listor (`- ` / `1. `), blockcitat (`> `), länkar `[text](url)`, rubriker `###` eller mindre, inline-kod med backticks. INTE råa HTML-taggar, bilder, eller `#`/`##`.",
		"Flytta aldrig sektionsrubriker (t.ex. \"Sås\") till `description` — använd `sections[].name`.",
		"Dela upp i sektioner när källan har rubriker som \"Sås\", \"Sallad\", \"Topping\". Enkla recept ska ha EXAKT en sektion med `name: \"\"` — uppfinn inte namn som \"Övrigt\" eller \"Huvudrätt\".",
		"Exempel (enkelt): {\"description\":\"\",\"sections\":[{\"name\":\"\",\"ingredients\":[…],\"instructions\":[…]}]}",
		"Exempel (rikt intro): {\"description\":\"**4 portioner** · ca 30 min\\n\\n> Ett enkelt vardagsrecept.\",\"sections\":[…]}",
	}, "\n")

	userPrompt := "Extrahera receptet från bilden."

	body := llmChatRequest{
		Model:          c.model,
		Temperature:    0.1,
		MaxTokens:      2048,
		ResponseFormat: &llmRespFormat{Type: "json_object"},
		Messages: []llmMessage{
			{
				Role: "system",
				Content: []llmMessageContent{
					{Type: "text", Text: systemPrompt},
				},
			},
			{
				Role: "user",
				Content: []llmMessageContent{
					{Type: "text", Text: userPrompt},
					{Type: "image_url", ImageURL: &llmImageURL{URL: dataURL}},
				},
			},
		},
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return Recipe{}, fmt.Errorf("marshal llm request: %w", err)
	}

	endpoint := c.baseURL + "/chat/completions"
	reqCtx, cancel := context.WithTimeout(ctx, recipeLLMTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return Recipe{}, fmt.Errorf("build llm request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// Bearer token is server-config (env), not request-derived. We do NOT
	// log the value in any branch, including error paths.
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	logAttrs := []any{
		"model", c.model,
		"image_size_bytes", len(imageBytes),
	}

	resp, err := c.http.Do(req)
	if err != nil {
		slog.Warn("recipe llm request failed", append(logAttrs, "error_class", classifyHTTPError(err))...)
		return Recipe{}, fmt.Errorf("%w: upstream unreachable", ErrLLMResponse)
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, recipeLLMMaxRespBytes+1)
	respBytes, err := io.ReadAll(limited)
	if err != nil {
		slog.Warn("recipe llm read failed",
			append(logAttrs, "status", resp.StatusCode, "error_class", "read")...,
		)
		return Recipe{}, fmt.Errorf("%w: read upstream", ErrLLMResponse)
	}
	if int64(len(respBytes)) > recipeLLMMaxRespBytes {
		slog.Warn("recipe llm response too large", append(logAttrs, "status", resp.StatusCode)...)
		return Recipe{}, fmt.Errorf("%w: response too large", ErrLLMResponse)
	}
	logAttrs = append(logAttrs, "status", resp.StatusCode, "response_size_bytes", len(respBytes))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Do NOT include respBytes in the log - it can contain attacker-
		// controlled payloads from prompt injection.
		slog.Warn("recipe llm non-2xx", logAttrs...)
		return Recipe{}, fmt.Errorf("%w: upstream status %d", ErrLLMResponse, resp.StatusCode)
	}

	var chat llmChatResponse
	if err := json.Unmarshal(respBytes, &chat); err != nil {
		slog.Warn("recipe llm decode failed", append(logAttrs, "error_class", "decode_envelope")...)
		return Recipe{}, fmt.Errorf("%w: cannot decode envelope", ErrLLMResponse)
	}
	if len(chat.Choices) == 0 || strings.TrimSpace(chat.Choices[0].Message.Content) == "" {
		slog.Warn("recipe llm empty content", logAttrs...)
		return Recipe{}, fmt.Errorf("%w: empty model output", ErrLLMResponse)
	}

	var raw rawRecipeResponse
	if err := json.Unmarshal([]byte(chat.Choices[0].Message.Content), &raw); err != nil {
		slog.Warn("recipe llm payload decode failed", append(logAttrs, "error_class", "decode_payload")...)
		return Recipe{}, fmt.Errorf("%w: cannot decode payload", ErrLLMResponse)
	}

	recipe := rawToRecipe(raw)
	cleaned, err := ValidateAndNormalize(recipe)
	if err != nil {
		slog.Warn("recipe llm payload invalid", append(logAttrs, "error_class", "validate")...)
		return Recipe{}, fmt.Errorf("%w: %v", ErrLLMResponse, err)
	}
	slog.Info("recipe llm parse ok", logAttrs...)
	return cleaned, nil
}

// classifyHTTPError reduces a transport-level error to a coarse class so
// it can be safely logged. The original error message can include host
// names, certificate paths, or other deployment details.
func classifyHTTPError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	return "transport"
}
