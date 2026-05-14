package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"time"
)

// geminiAPIBaseURL is the base URL for the Gemini Generative Language API.
// Hosted at a fixed Google domain; not user-controlled. SSRF-safe because the
// only user-controllable inputs are the model name (validated to a strict
// allowlist of characters) and the API key (carried as a query parameter).
const geminiAPIBaseURL = "https://generativelanguage.googleapis.com/v1beta"

// EmbeddingClient is a thin HTTP client over the Gemini batchEmbedContents
// endpoint with built-in batching, RPM rate limiting and retries.
type EmbeddingClient struct {
	apiKey     string
	model      string
	httpClient *http.Client

	// batchSize is the maximum number of inputs sent in one API call.
	batchSize int

	// rate limiter — a buffered channel acting as a token bucket. One token
	// is consumed per outbound request. Tokens are refilled at rpm/min.
	tokens chan struct{}
	stopCh chan struct{}
}

// NewEmbeddingClient constructs a client. rpm must be > 0; batchSize is
// clamped to the API's documented maximum of 100.
func NewEmbeddingClient(apiKey, model string, batchSize, rpm int) *EmbeddingClient {
	if batchSize <= 0 {
		batchSize = 100
	}
	if batchSize > 100 {
		batchSize = 100
	}
	if rpm <= 0 {
		rpm = 60
	}

	c := &EmbeddingClient{
		apiKey:    apiKey,
		model:     model,
		batchSize: batchSize,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		tokens: make(chan struct{}, rpm),
		stopCh: make(chan struct{}),
	}

	// Pre-fill the bucket so the very first request goes through immediately.
	for i := 0; i < rpm; i++ {
		c.tokens <- struct{}{}
	}

	// Refill one token every (60s / rpm).
	interval := time.Duration(int64(time.Minute) / int64(rpm))
	go c.refill(interval, rpm)

	return c
}

func (c *EmbeddingClient) refill(interval time.Duration, capacity int) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-c.stopCh:
			return
		case <-t.C:
			select {
			case c.tokens <- struct{}{}:
			default:
				_ = capacity // bucket full
			}
		}
	}
}

// Close stops the background refill goroutine.
func (c *EmbeddingClient) Close() {
	select {
	case <-c.stopCh:
	default:
		close(c.stopCh)
	}
}

// BatchSize returns the configured batch size.
func (c *EmbeddingClient) BatchSize() int { return c.batchSize }

// EmbedBatch embeds a single batch of texts (must be <= BatchSize). Returns
// vectors in the same order as the input. Retries transient errors (429/5xx)
// up to maxAttempts with exponential backoff.
func (c *EmbeddingClient) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	if len(texts) > c.batchSize {
		return nil, fmt.Errorf("embedding batch exceeds batch size: %d > %d", len(texts), c.batchSize)
	}

	const maxAttempts = 3
	var lastErr error
	backoff := time.Second
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := c.acquireToken(ctx); err != nil {
			return nil, err
		}

		vectors, retryable, err := c.doRequest(ctx, texts)
		if err == nil {
			return vectors, nil
		}
		lastErr = err
		if !retryable || attempt == maxAttempts {
			break
		}
		// Generic log only — never include API key or upstream body.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
		backoff *= 2
	}
	// Return a generic error; callers should surface a sanitized message and
	// avoid logging the underlying upstream body or the API key. The wrapped
	// error here is for server-side diagnostics only.
	return nil, fmt.Errorf("gemini embedding request failed: %w", lastErr)
}

func (c *EmbeddingClient) acquireToken(ctx context.Context) error {
	select {
	case <-c.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// geminiBatchEmbedRequest matches the JSON shape of batchEmbedContents.
type geminiBatchEmbedRequest struct {
	Requests []geminiEmbedItemRequest `json:"requests"`
}

type geminiEmbedItemRequest struct {
	Model   string        `json:"model"`
	Content geminiContent `json:"content"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiBatchEmbedResponse struct {
	Embeddings []geminiEmbedding `json:"embeddings"`
}

type geminiEmbedding struct {
	Values []float32 `json:"values"`
}

// doRequest performs a single HTTP call. Returns (vectors, retryable, error).
func (c *EmbeddingClient) doRequest(ctx context.Context, texts []string) ([][]float32, bool, error) {
	reqBody := geminiBatchEmbedRequest{
		Requests: make([]geminiEmbedItemRequest, len(texts)),
	}
	modelPath := "models/" + c.model
	for i, t := range texts {
		reqBody.Requests[i] = geminiEmbedItemRequest{
			Model:   modelPath,
			Content: geminiContent{Parts: []geminiPart{{Text: t}}},
		}
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, false, fmt.Errorf("marshal request: %w", err)
	}

	endpoint, err := buildGeminiEndpoint(c.model, c.apiKey)
	if err != nil {
		return nil, false, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, false, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Network-level errors are retryable.
		return nil, true, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		// Drain a bounded amount of the body for diagnostics but don't return
		// it to callers (could leak upstream details).
		_, _ = io.CopyN(io.Discard, resp.Body, 4096)
		return nil, true, fmt.Errorf("upstream status %d", resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.CopyN(io.Discard, resp.Body, 4096)
		return nil, false, fmt.Errorf("upstream status %d", resp.StatusCode)
	}

	var parsed geminiBatchEmbedResponse
	dec := json.NewDecoder(io.LimitReader(resp.Body, 64*1024*1024)) // 64MiB safety cap
	if err := dec.Decode(&parsed); err != nil {
		return nil, false, fmt.Errorf("decode response: %w", err)
	}
	if len(parsed.Embeddings) != len(texts) {
		return nil, false, fmt.Errorf("response length mismatch: got %d want %d", len(parsed.Embeddings), len(texts))
	}

	vectors := make([][]float32, len(texts))
	for i, e := range parsed.Embeddings {
		if len(e.Values) == 0 {
			return nil, false, errors.New("empty embedding in response")
		}
		vectors[i] = e.Values
	}
	return vectors, false, nil
}

// buildGeminiEndpoint constructs and validates the outbound URL. The model
// name is restricted to a safe character set to prevent URL injection. The
// host is hard-coded so this function is not vulnerable to SSRF via user
// input.
func buildGeminiEndpoint(model, apiKey string) (string, error) {
	if model == "" {
		return "", errors.New("embedding model is empty")
	}
	for _, r := range model {
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.'
		if !ok {
			return "", fmt.Errorf("invalid embedding model name")
		}
	}
	if apiKey == "" {
		return "", errors.New("gemini api key is empty")
	}

	// url.Values.Encode escapes the API key safely.
	q := url.Values{}
	q.Set("key", apiKey)
	return fmt.Sprintf("%s/models/%s:batchEmbedContents?%s", geminiAPIBaseURL, model, q.Encode()), nil
}

// CosineSimilarity returns the cosine similarity of two equal-length vectors.
// Returns 0 if either is zero-length, has different lengths, or has zero norm.
// Provided here so the next PR can drop straight in.
func CosineSimilarity(a, b []float32) float32 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		ai := float64(a[i])
		bi := float64(b[i])
		dot += ai * bi
		na += ai * ai
		nb += bi * bi
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return float32(dot / (math.Sqrt(na) * math.Sqrt(nb)))
}
