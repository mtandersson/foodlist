package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJevCategorizerDecideCategory(t *testing.T) {
	t.Parallel()

	categories := []Category{
		{ID: "dry", Name: "Torrvaror"},
		{ID: "asian", Name: "Asiatiskt & Taco🌮"},
	}

	type observedRequest struct {
		method string
		auth   string
		body   jevSystemOneRequest
		err    error
	}
	observed := make(chan observedRequest, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body jevSystemOneRequest
		decodeErr := json.NewDecoder(r.Body).Decode(&body)
		observed <- observedRequest{
			method: r.Method,
			auth:   r.Header.Get("Authorization"),
			body:   body,
			err:    decodeErr,
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model":"jev-1.13.0",
			"answers":{
				"category":{
					"type":"choice",
					"choice":"category_002",
					"confidence":0.91,
					"probabilities":{"category_001":0.05,"category_002":0.93,"none_of_above":0.02}
				}
			},
			"usage":{"input_tokens":123,"output_tokens":12},
			"request_id":"req-test"
		}`))
	}))
	defer upstream.Close()

	jev := NewJevCategorizer("test-key", upstream.URL, "jev-test", 0.70)
	decision, err := jev.DecideCategory(context.Background(), "bambuskott", categories, nil)
	require.NoError(t, err)
	require.NotNil(t, decision)
	require.Equal(t, "asian", decision.CategoryID)
	require.Equal(t, 0.91, decision.Confidence)
	require.Equal(t, 0.93, decision.Probability)
	require.Equal(t, "jev-1.13.0", decision.Model)

	got := <-observed
	require.NoError(t, got.err)
	require.Equal(t, http.MethodPost, got.method)
	require.Equal(t, "Bearer test-key", got.auth)
	require.Equal(t, "jev-test", got.body.Model)
	require.Equal(t, "bambuskott", got.body.State["item"])
	require.Equal(t, "sv-SE", got.body.State["locale"])
	q := got.body.Questions["category"]
	require.Equal(t, "choice", q.Type)
	require.Equal(t, "Torrvaror", q.Criteria["category_001"])
	require.Equal(t, "Asiatiskt & Taco🌮", q.Criteria["category_002"])
	require.Contains(t, q.Criteria, "none_of_above")
}

func TestJevCategorizerLowConfidenceReturnsNoDecision(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model":"jev-1.13.0",
			"answers":{"category":{
				"type":"choice",
				"choice":"category_001",
				"confidence":0.69,
				"probabilities":{"category_001":0.60,"none_of_above":0.40}
			}}
		}`))
	}))
	defer upstream.Close()

	jev := NewJevCategorizer("test-key", upstream.URL, "jev-test", 0.70)
	decision, err := jev.DecideCategory(context.Background(), "kapris", []Category{{ID: "dry", Name: "Torrvaror"}}, nil)
	require.NoError(t, err)
	require.Nil(t, decision)
}

func TestJevCategorizerNoneOfAboveReturnsNoDecision(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model":"jev-1.13.0",
			"answers":{"category":{
				"type":"choice",
				"choice":"none_of_above",
				"confidence":0.95,
				"probabilities":{"category_001":0.05,"none_of_above":0.95}
			}}
		}`))
	}))
	defer upstream.Close()

	jev := NewJevCategorizer("test-key", upstream.URL, "jev-test", 0.70)
	decision, err := jev.DecideCategory(context.Background(), "mystery item", []Category{{ID: "dry", Name: "Torrvaror"}}, nil)
	require.NoError(t, err)
	require.Nil(t, decision)
}

func TestJevCategorizerRejectsUnknownChoice(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model":"jev-1.13.0",
			"answers":{"category":{
				"type":"choice",
				"choice":"invented_category",
				"confidence":0.99,
				"probabilities":{"invented_category":0.99}
			}}
		}`))
	}))
	defer upstream.Close()

	jev := NewJevCategorizer("test-key", upstream.URL, "jev-test", 0.70)
	decision, err := jev.DecideCategory(context.Background(), "kapris", []Category{{ID: "dry", Name: "Torrvaror"}}, nil)
	require.Error(t, err)
	require.Nil(t, decision)
	require.Contains(t, err.Error(), "unknown category choice")
}

func TestJevCategorizerHTTPFailure(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer upstream.Close()

	jev := NewJevCategorizer("test-key", upstream.URL, "jev-test", 0.70)
	decision, err := jev.DecideCategory(context.Background(), "kapris", []Category{{ID: "dry", Name: "Torrvaror"}}, nil)
	require.Error(t, err)
	require.Nil(t, decision)
	require.Contains(t, err.Error(), "HTTP 429")
	require.NotContains(t, err.Error(), "rate limited")
}

func TestJevCategorizerIncludesHistoryExamplesInCriteria(t *testing.T) {
	t.Parallel()

	type observedExamplesRequest struct {
		body jevSystemOneRequest
		err  error
	}
	observed := make(chan observedExamplesRequest, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body jevSystemOneRequest
		decodeErr := json.NewDecoder(r.Body).Decode(&body)
		observed <- observedExamplesRequest{body: body, err: decodeErr}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model":"jev-1.13.0",
			"answers":{"category":{
				"type":"choice",
				"choice":"category_001",
				"confidence":0.90,
				"probabilities":{"category_001":0.95,"none_of_above":0.05}
			}}
		}`))
	}))
	defer upstream.Close()

	categories := []Category{{ID: "dry", Name: "Torrvaror"}}
	examples := map[string][]string{"dry": {"havregryn", "makaroner", "maizena"}}
	jev := NewJevCategorizer("test-key", upstream.URL, "jev-test", 0.70)

	decision, err := jev.DecideCategory(context.Background(), "strösocker", categories, examples)
	require.NoError(t, err)
	require.NotNil(t, decision)

	got := <-observed
	require.NoError(t, got.err)
	require.Equal(t,
		"Torrvaror. Examples from this Foodlist history: havregryn; makaroner; maizena",
		got.body.Questions["category"].Criteria["category_001"],
	)
}

func TestJevCategorizerBoundsHistoryExamplesInRequest(t *testing.T) {
	t.Parallel()

	observed := make(chan jevSystemOneRequest, 2)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body jevSystemOneRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		observed <- body
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"jev-test","answers":{"category":{"type":"choice","choice":"none_of_above","confidence":0.99}}}`))
	}))
	defer upstream.Close()

	jev := NewJevCategorizer("test-key", upstream.URL, "jev-test", 0.70)
	categories := []Category{{ID: "dry", Name: "Torrvaror"}, {ID: "produce", Name: "Frukt & Grönt"}}
	examples := map[string][]string{"dry": {strings.Repeat("x", maxJevExampleNameBytes+1), "havregryn", "makaroner"}}
	decision, err := jev.DecideCategory(context.Background(), "strösocker", categories, examples)
	require.NoError(t, err)
	require.Nil(t, decision)
	criteria := (<-observed).Questions["category"].Criteria
	require.Equal(t, "Torrvaror"+jevHistoryPrefix+"havregryn; makaroner", criteria["category_001"])
	require.Equal(t, "Frukt & Grönt", criteria["category_002"])

	categories = make([]Category, maxJevCategories)
	examples = make(map[string][]string, maxJevCategories)
	for i := range categories {
		id := fmt.Sprintf("category-%03d", i)
		categories[i] = Category{ID: id, Name: id}
		examples[id] = []string{
			strings.Repeat("<", maxJevExampleNameBytes),
			strings.Repeat("&", maxJevExampleNameBytes),
			strings.Repeat("c", maxJevExampleNameBytes),
			strings.Repeat("d", maxJevExampleNameBytes),
		}
	}
	decision, err = jev.DecideCategory(context.Background(), "strösocker", categories, examples)
	require.NoError(t, err)
	require.Nil(t, decision)
	criteria = (<-observed).Questions["category"].Criteria
	addedBytes := 0
	for i, category := range categories {
		description := criteria[fmt.Sprintf("category_%03d", i+1)]
		require.True(t, strings.HasPrefix(description, category.Name))
		encodedDescription, err := json.Marshal(description)
		require.NoError(t, err)
		encodedName, err := json.Marshal(category.Name)
		require.NoError(t, err)
		addedBytes += len(encodedDescription) - len(encodedName)
	}
	require.Positive(t, addedBytes)
	require.LessOrEqual(t, addedBytes, maxJevHistoryBytes)
	require.Equal(t, categories[len(categories)-1].Name, criteria[fmt.Sprintf("category_%03d", len(categories))])
}
