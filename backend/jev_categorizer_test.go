package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	decision, err := jev.DecideCategory(context.Background(), "bambuskott", categories)
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
	decision, err := jev.DecideCategory(context.Background(), "kapris", []Category{{ID: "dry", Name: "Torrvaror"}})
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
	decision, err := jev.DecideCategory(context.Background(), "mystery item", []Category{{ID: "dry", Name: "Torrvaror"}})
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
	decision, err := jev.DecideCategory(context.Background(), "kapris", []Category{{ID: "dry", Name: "Torrvaror"}})
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
	decision, err := jev.DecideCategory(context.Background(), "kapris", []Category{{ID: "dry", Name: "Torrvaror"}})
	require.Error(t, err)
	require.Nil(t, decision)
	require.Contains(t, err.Error(), "HTTP 429")
	require.NotContains(t, err.Error(), "rate limited")
}
