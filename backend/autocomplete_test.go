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

// Test Levenshtein distance calculation
func TestLevenshteinDistance_EmptyStrings(t *testing.T) {
	assert.Equal(t, 0, levenshteinDistance("", ""))
	assert.Equal(t, 3, levenshteinDistance("", "abc"))
	assert.Equal(t, 3, levenshteinDistance("abc", ""))
}

func TestLevenshteinDistance_EqualStrings(t *testing.T) {
	assert.Equal(t, 0, levenshteinDistance("hello", "hello"))
	assert.Equal(t, 0, levenshteinDistance("Milk", "milk")) // case insensitive
}

func TestLevenshteinDistance_SingleCharDifference(t *testing.T) {
	assert.Equal(t, 1, levenshteinDistance("cat", "bat"))  // substitution
	assert.Equal(t, 1, levenshteinDistance("cat", "cats")) // insertion
	assert.Equal(t, 1, levenshteinDistance("cats", "cat")) // deletion
}

func TestLevenshteinDistance_MultipleDifferences(t *testing.T) {
	assert.Equal(t, 3, levenshteinDistance("kitten", "sitting"))
	assert.Equal(t, 1, levenshteinDistance("milk", "silk"))   // 1 sub: m -> s
	assert.Equal(t, 3, levenshteinDistance("milk", "meal"))   // 3 subs: i->e, l->a, k->l
	assert.Equal(t, 2, levenshteinDistance("book", "back"))   // 2 subs: o->a, o->c
}

func TestLevenshteinDistance_CaseInsensitive(t *testing.T) {
	assert.Equal(t, 0, levenshteinDistance("MILK", "milk"))
	assert.Equal(t, 0, levenshteinDistance("Bread", "BREAD"))
}

// Test autocomplete with frequency tracking
func TestState_TrackNameFrequency(t *testing.T) {
	state := NewState()

	// Add same item multiple times
	state.Apply(TodoCreated{Type: "TodoCreated", ID: "1", Name: "Milk", CreatedAt: time.Now(), SortOrder: 1000})
	state.Apply(TodoCreated{Type: "TodoCreated", ID: "2", Name: "milk", CreatedAt: time.Now(), SortOrder: 2000})
	state.Apply(TodoCreated{Type: "TodoCreated", ID: "3", Name: "MILK", CreatedAt: time.Now(), SortOrder: 3000})

	freq := state.GetNameFrequency()

	// Should track frequency (case-insensitive)
	// The key should be the most recent canonical form
	var milkFreq int
	for name, count := range freq {
		if strings.ToLower(name) == "milk" {
			milkFreq = count
		}
	}
	assert.Equal(t, 3, milkFreq)
}

func TestState_GetActiveTodoNames(t *testing.T) {
	state := NewState()

	now := time.Now()
	state.Apply(TodoCreated{Type: "TodoCreated", ID: "1", Name: "Active Item", CreatedAt: now, SortOrder: 1000})
	state.Apply(TodoCreated{Type: "TodoCreated", ID: "2", Name: "Completed Item", CreatedAt: now, SortOrder: 2000})
	state.Apply(TodoCompleted{Type: "TodoCompleted", ID: "2", CompletedAt: now})

	names := state.GetActiveTodoNames()

	assert.Len(t, names, 1)
	assert.Contains(t, names, "Active Item")
}

func setupTestServerWithTodos(t *testing.T) (*Server, *httptest.Server, string) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")

	store, err := NewEventStore(filePath)
	require.NoError(t, err)

	server := NewServer(store)

	// Add some historical todos for autocomplete
	now := time.Now().UTC()

	// Milk appears 3 times (high frequency)
	store.Append(TodoCreated{Type: "TodoCreated", ID: "1", Name: "Milk", CreatedAt: now, SortOrder: 1000})
	store.Append(TodoCompleted{Type: "TodoCompleted", ID: "1", CompletedAt: now})
	store.Append(TodoCreated{Type: "TodoCreated", ID: "2", Name: "Milk", CreatedAt: now, SortOrder: 2000})
	store.Append(TodoCompleted{Type: "TodoCompleted", ID: "2", CompletedAt: now})
	store.Append(TodoCreated{Type: "TodoCreated", ID: "3", Name: "Milk", CreatedAt: now, SortOrder: 3000})
	store.Append(TodoCompleted{Type: "TodoCompleted", ID: "3", CompletedAt: now})

	// Bread appears 2 times
	store.Append(TodoCreated{Type: "TodoCreated", ID: "4", Name: "Bread", CreatedAt: now, SortOrder: 4000})
	store.Append(TodoCompleted{Type: "TodoCompleted", ID: "4", CompletedAt: now})
	store.Append(TodoCreated{Type: "TodoCreated", ID: "5", Name: "Bread", CreatedAt: now, SortOrder: 5000})
	store.Append(TodoCompleted{Type: "TodoCompleted", ID: "5", CompletedAt: now})

	// Eggs appears 1 time
	store.Append(TodoCreated{Type: "TodoCreated", ID: "6", Name: "Eggs", CreatedAt: now, SortOrder: 6000})
	store.Append(TodoCompleted{Type: "TodoCompleted", ID: "6", CompletedAt: now})

	// Butter appears 1 time (active - not completed)
	store.Append(TodoCreated{Type: "TodoCreated", ID: "7", Name: "Butter", CreatedAt: now, SortOrder: 7000})

	// Load events into state
	err = server.LoadEvents()
	require.NoError(t, err)

	go server.Run()

	ts := httptest.NewServer(http.HandlerFunc(server.HandleWebSocket))
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	return server, ts, wsURL
}

func TestAutocomplete_EmptyQueryReturnsFrequentItems(t *testing.T) {
	server, ts, _ := setupTestServerWithTodos(t)
	defer ts.Close()

	// Get suggestions with empty query
	suggestions := server.getAutocompleteSuggestions("")

	// Should return items sorted by frequency
	// Butter should be filtered out (active)
	require.NotEmpty(t, suggestions)

	// Milk (freq 3) should be first
	assert.Equal(t, "Milk", suggestions[0])

	// Butter should NOT be in suggestions (it's active)
	for _, s := range suggestions {
		assert.NotEqual(t, "Butter", s)
	}
}

func TestAutocomplete_PartialMatchPrefixPriority(t *testing.T) {
	server, ts, _ := setupTestServerWithTodos(t)
	defer ts.Close()

	// Query "Mi" should match "Milk" with prefix priority
	suggestions := server.getAutocompleteSuggestions("Mi")

	require.NotEmpty(t, suggestions)
	assert.Contains(t, suggestions, "Milk")
}

func TestAutocomplete_FilterOutActiveTodos(t *testing.T) {
	server, ts, _ := setupTestServerWithTodos(t)
	defer ts.Close()

	// "Butter" is an active todo, should not appear in suggestions
	suggestions := server.getAutocompleteSuggestions("But")

	for _, s := range suggestions {
		assert.NotEqual(t, "Butter", s)
	}
}

func TestAutocomplete_FuzzyMatch(t *testing.T) {
	server, ts, _ := setupTestServerWithTodos(t)
	defer ts.Close()

	// "Mlk" should match "Milk" with distance 1
	suggestions := server.getAutocompleteSuggestions("Mlk")

	require.NotEmpty(t, suggestions)
	assert.Contains(t, suggestions, "Milk")
}

func TestAutocomplete_MaxFourSuggestions(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")
	store, _ := NewEventStore(filePath)
	server := NewServer(store)

	now := time.Now()
	// Add 10 different items
	for i := 0; i < 10; i++ {
		name := string(rune('A' + i))
		id := string(rune('0' + i))
		store.Append(TodoCreated{Type: "TodoCreated", ID: id, Name: name, CreatedAt: now, SortOrder: i * 1000})
		store.Append(TodoCompleted{Type: "TodoCompleted", ID: id, CompletedAt: now})
	}
	server.LoadEvents()

	suggestions := server.getAutocompleteSuggestions("")

	assert.LessOrEqual(t, len(suggestions), 4)
}

func TestAutocomplete_CaseInsensitiveMatching(t *testing.T) {
	server, ts, _ := setupTestServerWithTodos(t)
	defer ts.Close()

	// "milk" lowercase should match "Milk"
	suggestions1 := server.getAutocompleteSuggestions("milk")
	assert.Contains(t, suggestions1, "Milk")

	// "BREAD" uppercase should match "Bread"
	suggestions2 := server.getAutocompleteSuggestions("BREAD")
	assert.Contains(t, suggestions2, "Bread")
}

func TestAutocomplete_FrequencyRanking(t *testing.T) {
	server, ts, _ := setupTestServerWithTodos(t)
	defer ts.Close()

	// With empty query, Milk (3x) should rank higher than Bread (2x) which should rank higher than Eggs (1x)
	suggestions := server.getAutocompleteSuggestions("")

	// Find indices
	milkIdx := -1
	breadIdx := -1
	eggsIdx := -1
	for i, s := range suggestions {
		switch s {
		case "Milk":
			milkIdx = i
		case "Bread":
			breadIdx = i
		case "Eggs":
			eggsIdx = i
		}
	}

	// Milk should come before Bread
	if milkIdx != -1 && breadIdx != -1 {
		assert.Less(t, milkIdx, breadIdx)
	}

	// Bread should come before Eggs
	if breadIdx != -1 && eggsIdx != -1 {
		assert.Less(t, breadIdx, eggsIdx)
	}
}

func TestAutocomplete_DistanceLimit(t *testing.T) {
	server, ts, _ := setupTestServerWithTodos(t)
	defer ts.Close()

	// "xyz" should not match anything (distance > 3)
	suggestions := server.getAutocompleteSuggestions("xyz")

	// Should be empty or not contain our items
	for _, s := range suggestions {
		assert.NotEqual(t, "Milk", s)
		assert.NotEqual(t, "Bread", s)
		assert.NotEqual(t, "Eggs", s)
	}
}

func TestAutocomplete_WebSocketIntegration(t *testing.T) {
	_, ts, wsURL := setupTestServerWithTodos(t)
	defer ts.Close()

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Read initial rollup and client count
	conn.ReadMessage() // rollup
	conn.ReadMessage() // client count

	// Send autocomplete request
	request := AutocompleteRequest{
		Type:      "AutocompleteRequest",
		Query:     "Mi",
		RequestID: "test-123",
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
	assert.Equal(t, "test-123", response.RequestID)
	assert.Contains(t, response.Suggestions, "Milk")
}

func TestAutocomplete_ResponseOnlyToRequestingClient(t *testing.T) {
	_, ts, wsURL := setupTestServerWithTodos(t)
	defer ts.Close()

	// Connect two clients
	conn1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn1.Close()

	conn2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn2.Close()

	// Read initial messages
	conn1.ReadMessage() // rollup
	conn1.ReadMessage() // client count 1
	conn2.ReadMessage() // rollup
	conn2.ReadMessage() // client count 2
	conn1.ReadMessage() // client count 2 update

	// Client 1 sends autocomplete request
	request := AutocompleteRequest{
		Type:      "AutocompleteRequest",
		Query:     "Mi",
		RequestID: "client1-request",
	}
	requestData, _ := json.Marshal(request)
	conn1.WriteMessage(websocket.TextMessage, requestData)

	// Client 1 should receive response
	conn1.SetReadDeadline(time.Now().Add(time.Second))
	_, msg1, err := conn1.ReadMessage()
	require.NoError(t, err)

	var response AutocompleteResponse
	err = json.Unmarshal(msg1, &response)
	require.NoError(t, err)
	assert.Equal(t, "client1-request", response.RequestID)

	// Client 2 should NOT receive the autocomplete response
	// Set a short timeout
	conn2.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	_, _, err = conn2.ReadMessage()
	// Should timeout (no message received)
	assert.Error(t, err)
}

func TestAutocomplete_SubstringMatch(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")
	store, _ := NewEventStore(filePath)
	server := NewServer(store)

	now := time.Now()
	store.Append(TodoCreated{Type: "TodoCreated", ID: "1", Name: "Whole Milk", CreatedAt: now, SortOrder: 1000})
	store.Append(TodoCompleted{Type: "TodoCompleted", ID: "1", CompletedAt: now})
	server.LoadEvents()

	// "Milk" should match "Whole Milk" (substring)
	suggestions := server.getAutocompleteSuggestions("Milk")

	assert.Contains(t, suggestions, "Whole Milk")
}

func TestContainsEmoji(t *testing.T) {
	assert.True(t, containsEmoji("Milk 🥛"))
	assert.True(t, containsEmoji("🍞 Bread"))
	assert.True(t, containsEmoji("Hello 👋 World"))
	assert.True(t, containsEmoji("🧸"))
	assert.False(t, containsEmoji("Milk"))
	assert.False(t, containsEmoji("Plain text"))
	assert.False(t, containsEmoji("Numbers 123"))
}

func TestAutocomplete_PrefersEmojis(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")
	store, _ := NewEventStore(filePath)
	server := NewServer(store)

	now := time.Now()
	// Add both with same frequency (1 each)
	store.Append(TodoCreated{Type: "TodoCreated", ID: "1", Name: "Milk", CreatedAt: now, SortOrder: 1000})
	store.Append(TodoCompleted{Type: "TodoCompleted", ID: "1", CompletedAt: now})
	store.Append(TodoCreated{Type: "TodoCreated", ID: "2", Name: "Milk 🥛", CreatedAt: now, SortOrder: 2000})
	store.Append(TodoCompleted{Type: "TodoCompleted", ID: "2", CompletedAt: now})
	server.LoadEvents()

	// With same frequency, emoji version should come first
	suggestions := server.getAutocompleteSuggestions("Milk")

	require.Len(t, suggestions, 2)
	assert.Equal(t, "Milk 🥛", suggestions[0]) // Emoji version first
	assert.Equal(t, "Milk", suggestions[1])
}

func TestAutocomplete_EmptyQueryPrefersEmojis(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")
	store, _ := NewEventStore(filePath)
	server := NewServer(store)

	now := time.Now()
	// Add items with same frequency
	store.Append(TodoCreated{Type: "TodoCreated", ID: "1", Name: "Apples", CreatedAt: now, SortOrder: 1000})
	store.Append(TodoCompleted{Type: "TodoCompleted", ID: "1", CompletedAt: now})
	store.Append(TodoCreated{Type: "TodoCreated", ID: "2", Name: "Bananas 🍌", CreatedAt: now, SortOrder: 2000})
	store.Append(TodoCompleted{Type: "TodoCompleted", ID: "2", CompletedAt: now})
	server.LoadEvents()

	// With empty query and same frequency, emoji version should come first
	suggestions := server.getAutocompleteSuggestions("")

	require.Len(t, suggestions, 2)
	assert.Equal(t, "Bananas 🍌", suggestions[0]) // Emoji version first
	assert.Equal(t, "Apples", suggestions[1])
}

