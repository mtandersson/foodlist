package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func suggestionNames(list []AutocompleteSuggestion) []string {
	names := make([]string, 0, len(list))
	for _, s := range list {
		names = append(names, s.Name)
	}
	return names
}

func TestAutocompletePriority(t *testing.T) {
	s := &Server{}
	s.state = NewState()

	// Setup categories
	mejeriID := "mejeri"
	konserverID := "konserver"
	bageriID := "bageri"
	s.state.ApplyEvents([]Event{CategoryCreated{ID: mejeriID, Name: "Mejeri"}})
	s.state.ApplyEvents([]Event{CategoryCreated{ID: konserverID, Name: "Konserver"}})
	s.state.ApplyEvents([]Event{CategoryCreated{ID: bageriID, Name: "Bageri"}})

	// --- Populate state with items based on user's history ---
	// Simulating the history provided in the scenario
	s.state.ApplyEvents([]Event{TodoCreated{ID: "1", Name: "Bröd"}})
	s.state.ApplyEvents([]Event{TodoCreated{ID: "2", Name: "Bröd 🍞", CategoryID: &bageriID}})
	s.state.ApplyEvents([]Event{TodoCreated{ID: "3", Name: "Bröd 🍞", CategoryID: &bageriID}}) // This is the highest priority variant
	s.state.ApplyEvents([]Event{TodoCreated{ID: "4", Name: "Ost"}})
	s.state.ApplyEvents([]Event{TodoCreated{ID: "5", Name: "Oliver 🫒", CategoryID: &konserverID}})
	s.state.ApplyEvents([]Event{TodoCreated{ID: "6", Name: "Mjölk", CategoryID: &mejeriID}})
	s.state.ApplyEvents([]Event{TodoCreated{ID: "7", Name: "Mjöl"}})
	s.state.ApplyEvents([]Event{TodoCreated{ID: "8", Name: "Pasta"}})
	s.state.ApplyEvents([]Event{TodoCreated{ID: "9", Name: "Ost 🧀", CategoryID: &mejeriID}}) // This is the highest priority variant

	// Complete all todos so they are available for autocomplete suggestions.
	s.state.ApplyEvents([]Event{TodoCompleted{ID: "1", CompletedAt: time.Now()}})
	s.state.ApplyEvents([]Event{TodoCompleted{ID: "2", CompletedAt: time.Now()}})
	s.state.ApplyEvents([]Event{TodoCompleted{ID: "3", CompletedAt: time.Now()}})
	s.state.ApplyEvents([]Event{TodoCompleted{ID: "4", CompletedAt: time.Now()}})
	s.state.ApplyEvents([]Event{TodoCompleted{ID: "5", CompletedAt: time.Now()}})
	s.state.ApplyEvents([]Event{TodoCompleted{ID: "6", CompletedAt: time.Now()}})
	s.state.ApplyEvents([]Event{TodoCompleted{ID: "7", CompletedAt: time.Now()}})
	s.state.ApplyEvents([]Event{TodoCompleted{ID: "8", CompletedAt: time.Now()}})
	s.state.ApplyEvents([]Event{TodoCompleted{ID: "9", CompletedAt: time.Now()}})

	// --- Run Test ---
	t.Run("Prioritizes items with categories, emojis, and frequency correctly", func(t *testing.T) {
		// Get suggestions for "br", should match "Bröd"
		suggestions := s.getAutocompleteSuggestions("br")

		assert.NotEmpty(t, suggestions, "Should get some suggestions for 'br'")

		// The top suggestion should be the one with the category.
		assert.Equal(t, "Bröd 🍞", suggestions[0].Name, "Highest priority for 'Bröd' should have a category")
		require.NotNil(t, suggestions[0].CategoryID, "Category should not be nil")
		assert.Equal(t, bageriID, *suggestions[0].CategoryID, "Highest priority for 'Bröd' should have the 'Bageri' category")

		// Get suggestions for "ost", should match "Ost"
		suggestions = s.getAutocompleteSuggestions("ost")
		assert.NotEmpty(t, suggestions, "Should get some suggestions for 'ost'")
		assert.Equal(t, "Ost 🧀", suggestions[0].Name, "Highest priority for 'Ost' should have a category")
		require.NotNil(t, suggestions[0].CategoryID, "Category should not be nil")
		assert.Equal(t, mejeriID, *suggestions[0].CategoryID, "Highest priority for 'Ost' should have the 'Mejeri' category")

		// Get suggestions for an empty query, which should show prioritized history.
		suggestions = s.getAutocompleteSuggestions("")
		assert.NotEmpty(t, suggestions, "Should get some suggestions for empty query")

		// Expected order based on priority:
		// 1. "Bröd 🍞" (from "Bröd" entry, has category & emoji)
		// 2. "Ost 🧀" (from "Ost" entry, has category & emoji)
		// 3. "Oliver 🫒" (from "Oliver" entry, has category & emoji)
		// 4. "Mjölk" (from "Mjölk" entry, has category)
		// We only take the top 4 for this assertion.
		expectedTop4 := []string{"Bröd 🍞", "Ost 🧀", "Oliver 🫒", "Mjölk"}
		actualTop4 := suggestionNames(suggestions[:4])

		assert.Equal(t, expectedTop4, actualTop4, "The suggestions for empty query are not in the correct priority order.")
	})
}

func setupTestServerForWebsocket(t *testing.T) (*Server, *httptest.Server, string) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")
	store, err := NewEventStore(filePath)
	require.NoError(t, err)

	server := NewServer(store)

	// Add some data for autocomplete
	mejeriID := "mejeri"
	server.state.ApplyEvents([]Event{CategoryCreated{ID: mejeriID, Name: "Mejeri"}})
	server.state.ApplyEvents([]Event{TodoCreated{ID: "1", Name: "Mjölk", CategoryID: &mejeriID}})
	server.state.ApplyEvents([]Event{TodoCreated{ID: "2", Name: "Ost 🧀", CategoryID: &mejeriID}})
	server.state.ApplyEvents([]Event{TodoCompleted{ID: "1", CompletedAt: time.Now()}})
	server.state.ApplyEvents([]Event{TodoCompleted{ID: "2", CompletedAt: time.Now()}})
	go server.Run()

	ts := httptest.NewServer(http.HandlerFunc(server.HandleWebSocket))
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	return server, ts, wsURL
}

func TestAutocomplete_WebSocketIntegration(t *testing.T) {
	_, ts, wsURL := setupTestServerForWebsocket(t)
	defer ts.Close()

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Read initial rollup and client count
	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, _, err = conn.ReadMessage() // rollup
	require.NoError(t, err)
	_, _, err = conn.ReadMessage() // client count
	require.NoError(t, err)

	// Send autocomplete request
	request := AutocompleteRequest{
		Type:      "AutocompleteRequest",
		Query:     "Mj",
		RequestID: "test-ws-123",
	}
	requestData, _ := json.Marshal(request)
	err = conn.WriteMessage(websocket.TextMessage, requestData)
	require.NoError(t, err)

	// Read autocomplete response
	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, msg, err := conn.ReadMessage()
	require.NoError(t, err)

	var response AutocompleteResponse
	err = json.Unmarshal(msg, &response)
	require.NoError(t, err)

	assert.Equal(t, "AutocompleteResponse", response.Type)
	assert.Equal(t, "test-ws-123", response.RequestID)
	require.Len(t, response.Suggestions, 1)
	assert.Equal(t, "Mjölk", response.Suggestions[0].Name)
}
