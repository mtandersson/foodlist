package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultJevBaseURL             = "https://api.typesafe.ai/v1/systemone"
	DefaultJevModel               = "jev-latest"
	DefaultJevConfidenceThreshold = 0.70
	maxJevCategories              = 254 // one extra Choice option is reserved for none_of_above
)

// CategoryDecision is a provider-independent category choice. CategoryID is
// always one of the live category IDs supplied to DecideCategory.
type CategoryDecision struct {
	CategoryID  string
	Confidence  float64
	Probability float64
	Model       string
}

// CategoryDecider is the minimal surface required by auto-categorize. Jev is
// the production implementation; tests can inject a deterministic stub.
type CategoryDecider interface {
	DecideCategory(ctx context.Context, itemName string, categories []Category) (*CategoryDecision, error)
}

// JevCategorizer classifies a grocery item with TypeSafe Jev's Choice
// primitive. It deliberately does not own retry/fallback policy: the
// auto-categorize pipeline can fall back to the existing embedding scorer.
type JevCategorizer struct {
	apiKey              string
	baseURL             string
	model               string
	confidenceThreshold float64
	client              *http.Client
}

func NewJevCategorizer(apiKey, baseURL, model string, confidenceThreshold float64) *JevCategorizer {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DefaultJevBaseURL
	}
	if strings.TrimSpace(model) == "" {
		model = DefaultJevModel
	}
	if confidenceThreshold <= 0 || confidenceThreshold > 1 {
		confidenceThreshold = DefaultJevConfidenceThreshold
	}
	return &JevCategorizer{
		apiKey:              strings.TrimSpace(apiKey),
		baseURL:             strings.TrimSpace(baseURL),
		model:               strings.TrimSpace(model),
		confidenceThreshold: confidenceThreshold,
		client:              &http.Client{Timeout: 5 * time.Second},
	}
}

type jevChoiceQuestion struct {
	Type         string            `json:"type"`
	Instructions string            `json:"instructions"`
	Criteria     map[string]string `json:"criteria"`
}

type jevSystemOneRequest struct {
	Model     string                       `json:"model"`
	State     map[string]string            `json:"state"`
	Questions map[string]jevChoiceQuestion `json:"questions"`
}

type jevChoiceAnswer struct {
	Type          string             `json:"type"`
	Choice        string             `json:"choice"`
	Confidence    float64            `json:"confidence"`
	Probabilities map[string]float64 `json:"probabilities"`
}

type jevSystemOneResponse struct {
	Model   string                     `json:"model"`
	Answers map[string]jevChoiceAnswer `json:"answers"`
}

// DecideCategory asks Jev to pick one live category. Synthetic choice labels
// keep category IDs out of the model-facing text while still making the
// returned value trivial to validate and map back to an ID.
func (j *JevCategorizer) DecideCategory(ctx context.Context, itemName string, categories []Category) (*CategoryDecision, error) {
	if j == nil || j.apiKey == "" {
		return nil, errors.New("jev api key not configured")
	}
	itemName = strings.TrimSpace(itemName)
	if itemName == "" || len(categories) == 0 {
		return nil, nil
	}
	if len(categories) > maxJevCategories {
		return nil, fmt.Errorf("too many categories for Jev Choice: %d (max %d)", len(categories), maxJevCategories)
	}

	criteria := make(map[string]string, len(categories)+1)
	choiceToID := make(map[string]string, len(categories))
	for i, category := range categories {
		label := fmt.Sprintf("category_%03d", i+1)
		criteria[label] = category.Name
		choiceToID[label] = category.ID
	}
	criteria["none_of_above"] = "No existing Foodlist category is a reasonable fit for this grocery item"

	payload := jevSystemOneRequest{
		Model: j.model,
		State: map[string]string{
			"item":   itemName,
			"locale": "sv-SE",
		},
		Questions: map[string]jevChoiceQuestion{
			"category": {
				Type: "choice",
				Instructions: "Which existing Foodlist category best fits the grocery item in `item`? Treat the criteria descriptions as Swedish grocery-list section names. Choose none_of_above only when no existing category is a reasonable fit.",
				Criteria:     criteria,
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal Jev request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, j.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build Jev request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+j.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "foodlist/"+version)

	resp, err := j.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call Jev: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Do not include the upstream body: it may echo user input. The
		// caller logs only this status-bearing error.
		return nil, fmt.Errorf("Jev returned HTTP %d", resp.StatusCode)
	}

	var decoded jevSystemOneResponse
	dec := json.NewDecoder(resp.Body)
	// Ignore unknown top-level/provider metadata so a backwards-compatible
	// API addition does not break categorization.
	if err := dec.Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode Jev response: %w", err)
	}
	answer, ok := decoded.Answers["category"]
	if !ok {
		return nil, errors.New("Jev response missing category answer")
	}
	if answer.Type != "choice" {
		return nil, fmt.Errorf("Jev category answer has type %q", answer.Type)
	}
	if answer.Choice == "none_of_above" {
		return nil, nil
	}
	categoryID, ok := choiceToID[answer.Choice]
	if !ok {
		return nil, fmt.Errorf("Jev returned unknown category choice %q", answer.Choice)
	}
	if answer.Confidence < j.confidenceThreshold {
		return nil, nil
	}

	return &CategoryDecision{
		CategoryID:  categoryID,
		Confidence:  answer.Confidence,
		Probability: answer.Probabilities[answer.Choice],
		Model:       decoded.Model,
	}, nil
}

var _ CategoryDecider = (*JevCategorizer)(nil)
