package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestServer(t *testing.T) (*Server, *httptest.Server, string) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")

	store, err := NewEventStore(filePath)
	require.NoError(t, err)

	server := NewServer(store)

	// Start the server's event loop in background
	go server.Run()

	ts := httptest.NewServer(http.HandlerFunc(server.HandleWebSocket))

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	return server, ts, wsURL
}

func connectWS(t *testing.T, wsURL string) *websocket.Conn {
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	return conn
}

func TestServer_AcceptConnection(t *testing.T) {
	_, ts, wsURL := setupTestServer(t)
	defer ts.Close()

	conn := connectWS(t, wsURL)
	defer conn.Close()

	// Connection should be established
	assert.NotNil(t, conn)
}

func TestServer_SendStateRollupOnConnect(t *testing.T) {
	server, ts, wsURL := setupTestServer(t)
	defer ts.Close()

	// Add some events to the store first
	now := time.Now().UTC()
	server.store.Append(TodoCreated{
		Type:      "TodoCreated",
		ID:        "todo-1",
		Name:      "Test task",
		CreatedAt: now,
		SortOrder: 1000,
	})
	server.store.Append(CategoryCreated{
		Type:      "CategoryCreated",
		ID:        "cat-1",
		Name:      "Work",
		CreatedAt: now,
		SortOrder: 1000,
	})

	// Rebuild state from store
	events, _ := server.store.ReadAll()
	server.state.ApplyEvents(events)

	conn := connectWS(t, wsURL)
	defer conn.Close()

	// Should receive state rollup
	_, msg, err := conn.ReadMessage()
	require.NoError(t, err)

	var rollup StateRollup
	err = json.Unmarshal(msg, &rollup)
	require.NoError(t, err)

	assert.Equal(t, "StateRollup", rollup.Type)
	require.Len(t, rollup.Todos, 1)
	assert.Equal(t, "todo-1", rollup.Todos[0].ID)
	require.Len(t, rollup.Categories, 1)
	assert.Equal(t, "Work", rollup.Categories[0].Name)
}

func TestServer_BroadcastEventToAllClients(t *testing.T) {
	_, ts, wsURL := setupTestServer(t)
	defer ts.Close()

	// Connect two clients
	conn1 := connectWS(t, wsURL)
	defer conn1.Close()
	conn2 := connectWS(t, wsURL)
	defer conn2.Close()

	// Read initial rollups
	conn1.ReadMessage()
	conn2.ReadMessage()

	// Send event from client1
	event := TodoCreated{
		Type:      "TodoCreated",
		ID:        "todo-new",
		Name:      "New task",
		CreatedAt: time.Now().UTC(),
		SortOrder: 1000,
	}
	eventData, _ := json.Marshal(event)
	err := conn1.WriteMessage(websocket.TextMessage, eventData)
	require.NoError(t, err)

	// Both clients should receive the event
	_, msg1, err := conn1.ReadMessage()
	require.NoError(t, err)
	_, msg2, err := conn2.ReadMessage()
	require.NoError(t, err)

	var received1, received2 TodoCreated
	json.Unmarshal(msg1, &received1)
	json.Unmarshal(msg2, &received2)

	assert.Equal(t, "todo-new", received1.ID)
	assert.Equal(t, "todo-new", received2.ID)
}

func TestServer_PersistEventToStore(t *testing.T) {
	server, ts, wsURL := setupTestServer(t)
	defer ts.Close()

	conn := connectWS(t, wsURL)
	defer conn.Close()

	// Read initial rollup
	conn.ReadMessage()

	// Send event
	event := TodoCreated{
		Type:      "TodoCreated",
		ID:        "persist-test",
		Name:      "Persisted task",
		CreatedAt: time.Now().UTC(),
		SortOrder: 1000,
	}
	eventData, _ := json.Marshal(event)
	conn.WriteMessage(websocket.TextMessage, eventData)

	// Wait for broadcast
	conn.ReadMessage()

	// Give a moment for persistence
	time.Sleep(50 * time.Millisecond)

	// Read events from store
	events, err := server.store.ReadAll()
	require.NoError(t, err)
	require.Len(t, events, 1)

	created := events[0].(TodoCreated)
	assert.Equal(t, "persist-test", created.ID)
}

func TestServer_UpdateStateOnEvent(t *testing.T) {
	_, ts, wsURL := setupTestServer(t)
	defer ts.Close()

	conn := connectWS(t, wsURL)
	defer conn.Close()

	// Read initial rollup
	conn.ReadMessage()

	// Send event
	event := TodoCreated{
		Type:      "TodoCreated",
		ID:        "state-test",
		Name:      "State task",
		CreatedAt: time.Now().UTC(),
		SortOrder: 1000,
	}
	eventData, _ := json.Marshal(event)
	conn.WriteMessage(websocket.TextMessage, eventData)

	// Wait for processing
	conn.ReadMessage()
	time.Sleep(50 * time.Millisecond)

	// Connect new client and verify state includes new todo
	conn2 := connectWS(t, wsURL)
	defer conn2.Close()

	_, msg, _ := conn2.ReadMessage()
	var rollup StateRollup
	json.Unmarshal(msg, &rollup)

	require.Len(t, rollup.Todos, 1)
	assert.Equal(t, "state-test", rollup.Todos[0].ID)
}

func TestServer_HandleClientDisconnect(t *testing.T) {
	_, ts, wsURL := setupTestServer(t)
	defer ts.Close()

	conn1 := connectWS(t, wsURL)
	conn2 := connectWS(t, wsURL)
	defer conn2.Close()

	// Read rollups
	conn1.ReadMessage()
	conn2.ReadMessage()

	// Disconnect client 1
	conn1.Close()

	// Give time for disconnect handling
	time.Sleep(50 * time.Millisecond)

	// Client 2 should still be able to receive events
	event := TodoCreated{
		Type:      "TodoCreated",
		ID:        "after-disconnect",
		Name:      "Task after disconnect",
		CreatedAt: time.Now().UTC(),
		SortOrder: 1000,
	}
	eventData, _ := json.Marshal(event)
	conn2.WriteMessage(websocket.TextMessage, eventData)

	_, msg, err := conn2.ReadMessage()
	require.NoError(t, err)

	var received TodoCreated
	json.Unmarshal(msg, &received)
	assert.Equal(t, "after-disconnect", received.ID)
}

func TestServer_RejectInvalidEvent(t *testing.T) {
	_, ts, wsURL := setupTestServer(t)
	defer ts.Close()

	conn := connectWS(t, wsURL)
	defer conn.Close()

	// Read initial rollup
	conn.ReadMessage()

	// Send invalid JSON
	err := conn.WriteMessage(websocket.TextMessage, []byte("not valid json"))
	require.NoError(t, err) // Write should succeed

	// Server should not crash, connection should remain usable
	// Send a valid event
	event := TodoCreated{
		Type:      "TodoCreated",
		ID:        "after-invalid",
		Name:      "Valid task",
		CreatedAt: time.Now().UTC(),
		SortOrder: 1000,
	}
	eventData, _ := json.Marshal(event)
	conn.WriteMessage(websocket.TextMessage, eventData)

	// Should receive the broadcast
	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, msg, err := conn.ReadMessage()
	require.NoError(t, err)

	var received TodoCreated
	json.Unmarshal(msg, &received)
	assert.Equal(t, "after-invalid", received.ID)
}

func TestServer_LoadExistingEventsOnStart(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")

	// Create store and add events
	store1, _ := NewEventStore(filePath)
	now := time.Now().UTC()
	store1.Append(TodoCreated{
		Type:      "TodoCreated",
		ID:        "preexisting",
		Name:      "Existing task",
		CreatedAt: now,
		SortOrder: 1000,
	})
	store1.Close()

	// Create new store and server
	store2, _ := NewEventStore(filePath)
	srv := NewServer(store2)

	// Load existing events
	events, _ := store2.ReadAll()
	srv.state.ApplyEvents(events)

	ts := httptest.NewServer(http.HandlerFunc(srv.HandleWebSocket))
	defer ts.Close()

	go srv.Run()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	conn := connectWS(t, wsURL)
	defer conn.Close()

	// Should receive rollup with existing todo
	_, msg, _ := conn.ReadMessage()
	var rollup StateRollup
	json.Unmarshal(msg, &rollup)

	require.Len(t, rollup.Todos, 1)
	assert.Equal(t, "preexisting", rollup.Todos[0].ID)
}

func TestServer_ServeStaticFiles(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")

	// Create a test static file
	staticDir := filepath.Join(tmpDir, "static")
	os.MkdirAll(staticDir, 0755)
	os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<html>test</html>"), 0644)

	store, _ := NewEventStore(filePath)
	server := NewServer(store)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", server.HandleWebSocket)
	mux.Handle("/", http.FileServer(http.Dir(staticDir)))

	ts := httptest.NewServer(mux)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/index.html")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestServer_LoadEvents(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")

	// Create store with some events
	store1, _ := NewEventStore(filePath)
	now := time.Now().UTC()
	store1.Append(TodoCreated{
		Type:      "TodoCreated",
		ID:        "load-test",
		Name:      "Test load",
		CreatedAt: now,
		SortOrder: 1000,
	})
	store1.Close()

	// Create new server with existing events
	store2, _ := NewEventStore(filePath)
	defer store2.Close()

	server := NewServer(store2)

	// Initially empty
	assert.Equal(t, 0, server.state.TodoCount())

	// Load events
	err := server.LoadEvents()
	require.NoError(t, err)

	// Should have loaded event
	assert.Equal(t, 1, server.state.TodoCount())

	todos := server.state.GetTodos()
	assert.Equal(t, "load-test", todos[0].ID)
}

func TestServer_LoadEvents_InvalidStore(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")

	// Create store with invalid event
	os.WriteFile(filePath, []byte(`{"type":"UnknownEvent","id":"1"}`), 0644)

	store, _ := NewEventStore(filePath)
	defer store.Close()

	server := NewServer(store)

	// Should fail to load
	err := server.LoadEvents()
	assert.Error(t, err)
}

func TestServer_Run_BroadcastToMultipleClients(t *testing.T) {
	_, ts, wsURL := setupTestServer(t)
	defer ts.Close()

	// Connect three clients
	conn1 := connectWS(t, wsURL)
	defer conn1.Close()
	conn2 := connectWS(t, wsURL)
	defer conn2.Close()
	conn3 := connectWS(t, wsURL)
	defer conn3.Close()

	// Read initial rollups
	conn1.ReadMessage()
	conn2.ReadMessage()
	conn3.ReadMessage()

	// Send event from client1
	event := TodoCreated{
		Type:      "TodoCreated",
		ID:        "broadcast-test",
		Name:      "Broadcast task",
		CreatedAt: time.Now().UTC(),
		SortOrder: 1000,
	}
	eventData, _ := json.Marshal(event)
	conn1.WriteMessage(websocket.TextMessage, eventData)

	// All three clients should receive
	_, msg1, err1 := conn1.ReadMessage()
	_, msg2, err2 := conn2.ReadMessage()
	_, msg3, err3 := conn3.ReadMessage()

	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NoError(t, err3)

	var received1, received2, received3 TodoCreated
	json.Unmarshal(msg1, &received1)
	json.Unmarshal(msg2, &received2)
	json.Unmarshal(msg3, &received3)

	assert.Equal(t, "broadcast-test", received1.ID)
	assert.Equal(t, "broadcast-test", received2.ID)
	assert.Equal(t, "broadcast-test", received3.ID)
}

func TestServer_WritePump_HandleClosedConnection(t *testing.T) {
	_, ts, wsURL := setupTestServer(t)
	defer ts.Close()

	conn := connectWS(t, wsURL)

	// Read initial rollup
	conn.ReadMessage()

	// Close connection immediately
	conn.Close()

	// Wait a bit for cleanup
	time.Sleep(100 * time.Millisecond)

	// Server should handle it gracefully (no panic)
	// We can verify by connecting another client
	conn2 := connectWS(t, wsURL)
	defer conn2.Close()

	// Should still work
	_, msg, err := conn2.ReadMessage()
	require.NoError(t, err)

	var rollup StateRollup
	json.Unmarshal(msg, &rollup)
	assert.Equal(t, "StateRollup", rollup.Type)
}

func TestServer_ReadPump_IgnoreMalformedMessages(t *testing.T) {
	_, ts, wsURL := setupTestServer(t)
	defer ts.Close()

	conn := connectWS(t, wsURL)
	defer conn.Close()

	// Read initial rollup
	conn.ReadMessage()

	// Send malformed messages
	conn.WriteMessage(websocket.TextMessage, []byte("not json"))
	conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"InvalidType"}`))

	// Connection should still be alive
	// Send a valid event
	event := TodoCreated{
		Type:      "TodoCreated",
		ID:        "after-malformed",
		Name:      "Valid",
		CreatedAt: time.Now().UTC(),
		SortOrder: 1000,
	}
	eventData, _ := json.Marshal(event)
	conn.WriteMessage(websocket.TextMessage, eventData)

	// Should receive broadcast
	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, msg, err := conn.ReadMessage()
	require.NoError(t, err)

	var received TodoCreated
	json.Unmarshal(msg, &received)
	assert.Equal(t, "after-malformed", received.ID)
}

func TestServer_SetListTitle(t *testing.T) {
	_, ts, wsURL := setupTestServer(t)
	defer ts.Close()

	conn := connectWS(t, wsURL)
	defer conn.Close()

	// Read initial rollup and client count
	_, msg, err := conn.ReadMessage()
	require.NoError(t, err)
	conn.ReadMessage() // skip client count

	var rollup StateRollup
	json.Unmarshal(msg, &rollup)
	assert.Equal(t, "My Todo List", rollup.ListTitle)

	// Send SetListTitle command
	cmdJSON := `{"type":"SetListTitle","title":"Shopping List 2025"}`
	conn.WriteMessage(websocket.TextMessage, []byte(cmdJSON))

	// Read ListTitleChanged event
	_, msg2, err := conn.ReadMessage()
	require.NoError(t, err)

	var event ListTitleChanged
	json.Unmarshal(msg2, &event)
	assert.Equal(t, "ListTitleChanged", event.Type)
	assert.Equal(t, "Shopping List 2025", event.Title)
}

func TestServer_SetListTitle_BroadcastsToAllClients(t *testing.T) {
	_, ts, wsURL := setupTestServer(t)
	defer ts.Close()

	// Connect two clients
	conn1 := connectWS(t, wsURL)
	defer conn1.Close()
	conn2 := connectWS(t, wsURL)
	defer conn2.Close()

	// Read initial rollups and client counts
	conn1.ReadMessage() // rollup
	conn1.ReadMessage() // client count (1)
	conn2.ReadMessage() // rollup
	conn2.ReadMessage() // client count (2)
	conn1.ReadMessage() // client count (2) to first client

	// Client 1 changes the title
	cmdJSON := `{"type":"SetListTitle","title":"Shared List"}`
	conn1.WriteMessage(websocket.TextMessage, []byte(cmdJSON))

	// Both clients should receive the ListTitleChanged event
	_, msg1, err1 := conn1.ReadMessage()
	_, msg2, err2 := conn2.ReadMessage()

	require.NoError(t, err1)
	require.NoError(t, err2)

	var event1, event2 ListTitleChanged
	json.Unmarshal(msg1, &event1)
	json.Unmarshal(msg2, &event2)

	assert.Equal(t, "ListTitleChanged", event1.Type)
	assert.Equal(t, "Shared List", event1.Title)
	assert.Equal(t, "ListTitleChanged", event2.Type)
	assert.Equal(t, "Shared List", event2.Title)
}

func TestCommandToEvent_DeleteCategoryRejectedWhenNotEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewEventStore(filepath.Join(tmpDir, "events.jsonl"))
	require.NoError(t, err)
	server := NewServer(store)

	now := time.Now().UTC()
	catID := "cat-1"
	server.state.Apply(CategoryCreated{
		Type:      "CategoryCreated",
		ID:        catID,
		Name:      "Work",
		CreatedAt: now,
		SortOrder: 1000,
	})
	server.state.Apply(TodoCreated{
		Type:       "TodoCreated",
		ID:         "todo-1",
		Name:       "Task",
		CreatedAt:  now,
		SortOrder:  1000,
		CategoryID: &catID,
	})

	_, err = server.commandToEvent(DeleteCategoryCommand{
		BaseCommand: BaseCommand{Type: "DeleteCategory"},
		ID:          catID,
	})

	assert.Error(t, err)
}

func TestCommandToEvent_CategorizeRequiresExistingCategory(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewEventStore(filepath.Join(tmpDir, "events.jsonl"))
	require.NoError(t, err)
	server := NewServer(store)

	now := time.Now().UTC()
	server.state.Apply(TodoCreated{
		Type:      "TodoCreated",
		ID:        "todo-1",
		Name:      "Task",
		CreatedAt: now,
		SortOrder: 1000,
	})

	invalidCat := "missing"
	_, err = server.commandToEvent(CategorizeTodoCommand{
		BaseCommand: BaseCommand{Type: "CategorizeTodo"},
		ID:          "todo-1",
		CategoryID:  &invalidCat,
	})
	assert.Error(t, err)
}
