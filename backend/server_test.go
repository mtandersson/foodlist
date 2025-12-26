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

	// Should receive state rollup and client count (order may vary due to concurrency)
	var rollup StateRollup
	var clientCount ClientCountMessage
	rollupReceived := false
	clientCountReceived := false

	// Read up to 2 messages
	for i := 0; i < 2; i++ {
		_, msg, err := conn.ReadMessage()
		require.NoError(t, err)

		// Try to unmarshal as StateRollup
		var tempRollup StateRollup
		if err := json.Unmarshal(msg, &tempRollup); err == nil && tempRollup.Type == "StateRollup" {
			rollup = tempRollup
			rollupReceived = true
			continue
		}

		// Try to unmarshal as ClientCount
		var tempCount ClientCountMessage
		if err := json.Unmarshal(msg, &tempCount); err == nil && tempCount.Type == "ClientCount" {
			clientCount = tempCount
			clientCountReceived = true
			continue
		}
	}

	// Verify we received both messages
	require.True(t, rollupReceived, "Should receive StateRollup message")
	require.True(t, clientCountReceived, "Should receive ClientCount message")

	// Verify StateRollup content
	require.Len(t, rollup.Todos, 1)
	assert.Equal(t, "todo-1", rollup.Todos[0].ID)
	require.Len(t, rollup.Categories, 1)
	assert.Equal(t, "Work", rollup.Categories[0].Name)

	// Verify ClientCount
	assert.Equal(t, 1, clientCount.Count)
}

func TestServer_BroadcastEventToAllClients(t *testing.T) {
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

	// Send CreateTodo command from client1
	cmd := CreateTodoCommand{
		BaseCommand: BaseCommand{Type: "CreateTodo", CommandID: "cmd-1"},
		ID:          "todo-new",
		Name:        "New task",
		SortOrder:   1000,
	}
	cmdData, _ := json.Marshal(cmd)
	err := conn1.WriteMessage(websocket.TextMessage, cmdData)
	require.NoError(t, err)

	// Client1 should receive CommandResponse first, then both receive the event
	_, resp1, err := conn1.ReadMessage()
	require.NoError(t, err)
	var cmdResp CommandResponse
	json.Unmarshal(resp1, &cmdResp)
	assert.Equal(t, "CommandResponse", cmdResp.Type)
	assert.True(t, cmdResp.Success)

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

	// Read initial rollup and client count
	conn.ReadMessage()
	conn.ReadMessage()

	// Send command
	cmd := CreateTodoCommand{
		BaseCommand: BaseCommand{Type: "CreateTodo", CommandID: "cmd-2"},
		ID:          "persist-test",
		Name:        "Persisted task",
		SortOrder:   1000,
	}
	cmdData, _ := json.Marshal(cmd)
	conn.WriteMessage(websocket.TextMessage, cmdData)

	// Wait for CommandResponse and broadcast
	conn.ReadMessage() // CommandResponse
	conn.ReadMessage() // Event broadcast

	// Read events from store - the event is persisted synchronously before broadcast
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

	// Read initial rollup and client count
	conn.ReadMessage()
	conn.ReadMessage()

	// Send command
	cmd := CreateTodoCommand{
		BaseCommand: BaseCommand{Type: "CreateTodo", CommandID: "cmd-3"},
		ID:          "state-test",
		Name:        "State task",
		SortOrder:   1000,
	}
	cmdData, _ := json.Marshal(cmd)
	conn.WriteMessage(websocket.TextMessage, cmdData)

	// Wait for processing
	conn.ReadMessage() // CommandResponse
	conn.ReadMessage() // Event broadcast

	// Connect new client and verify state includes new todo
	// State is updated synchronously, so it should be immediately available
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

	// Read rollups and client counts
	conn1.ReadMessage() // rollup
	conn1.ReadMessage() // client count (1)
	conn2.ReadMessage() // rollup
	conn2.ReadMessage() // client count (2)
	conn1.ReadMessage() // client count (2) to first client

	// Disconnect client 1
	conn1.Close()

	// Client count update is sent synchronously to remaining clients
	conn2.ReadMessage() // client count (1) after disconnect

	// Client 2 should still be able to receive events
	cmd := CreateTodoCommand{
		BaseCommand: BaseCommand{Type: "CreateTodo", CommandID: "cmd-4"},
		ID:          "after-disconnect",
		Name:        "Task after disconnect",
		SortOrder:   1000,
	}
	cmdData, _ := json.Marshal(cmd)
	conn2.WriteMessage(websocket.TextMessage, cmdData)

	// Read CommandResponse first
	_, respMsg, err := conn2.ReadMessage()
	require.NoError(t, err)
	var cmdResp CommandResponse
	json.Unmarshal(respMsg, &cmdResp)
	assert.True(t, cmdResp.Success)

	// Then read the event broadcast
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

	// Read initial rollup and client count
	conn.ReadMessage()
	conn.ReadMessage()

	// Send invalid JSON
	err := conn.WriteMessage(websocket.TextMessage, []byte("not valid json"))
	require.NoError(t, err) // Write should succeed

	// Server should not crash, connection should remain usable
	// Send a valid command
	cmd := CreateTodoCommand{
		BaseCommand: BaseCommand{Type: "CreateTodo", CommandID: "cmd-5"},
		ID:          "after-invalid",
		Name:        "Valid task",
		SortOrder:   1000,
	}
	cmdData, _ := json.Marshal(cmd)
	conn.WriteMessage(websocket.TextMessage, cmdData)

	// Should receive CommandResponse and then the broadcast
	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, respMsg, err := conn.ReadMessage()
	require.NoError(t, err)
	var cmdResp CommandResponse
	json.Unmarshal(respMsg, &cmdResp)
	assert.True(t, cmdResp.Success)

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
	os.MkdirAll(staticDir, 0o755)
	os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<html>test</html>"), 0o644)

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
	os.WriteFile(filePath, []byte(`{"type":"UnknownEvent","id":"1"}`), 0o644)

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

	// Read initial rollups and client counts
	conn1.ReadMessage() // rollup
	conn1.ReadMessage() // client count (1)
	conn2.ReadMessage() // rollup
	conn2.ReadMessage() // client count (2)
	conn1.ReadMessage() // client count (2) to first client
	conn3.ReadMessage() // rollup
	conn3.ReadMessage() // client count (3)
	conn1.ReadMessage() // client count (3) to first client
	conn2.ReadMessage() // client count (3) to second client

	// Send command from client1
	cmd := CreateTodoCommand{
		BaseCommand: BaseCommand{Type: "CreateTodo", CommandID: "cmd-6"},
		ID:          "broadcast-test",
		Name:        "Broadcast task",
		SortOrder:   1000,
	}
	cmdData, _ := json.Marshal(cmd)
	conn1.WriteMessage(websocket.TextMessage, cmdData)

	// Client 1 gets CommandResponse
	_, respMsg, err1 := conn1.ReadMessage()
	require.NoError(t, err1)
	var cmdResp CommandResponse
	json.Unmarshal(respMsg, &cmdResp)
	assert.True(t, cmdResp.Success)

	// All three clients should receive the event
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

	// Verify server handles it gracefully by connecting another client
	// If cleanup wasn't handled properly, this would fail or panic
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

	// Read initial rollup and client count
	conn.ReadMessage()
	conn.ReadMessage()

	// Send malformed messages
	conn.WriteMessage(websocket.TextMessage, []byte("not json"))
	conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"InvalidType"}`))

	// Connection should still be alive
	// Send a valid command
	cmd := CreateTodoCommand{
		BaseCommand: BaseCommand{Type: "CreateTodo", CommandID: "cmd-7"},
		ID:          "after-malformed",
		Name:        "Valid",
		SortOrder:   1000,
	}
	cmdData, _ := json.Marshal(cmd)
	conn.WriteMessage(websocket.TextMessage, cmdData)

	// Should receive CommandResponse and then broadcast
	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, respMsg, err := conn.ReadMessage()
	require.NoError(t, err)
	var cmdResp CommandResponse
	json.Unmarshal(respMsg, &cmdResp)
	assert.True(t, cmdResp.Success)

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
	cmdJSON := `{"type":"SetListTitle","commandId":"set-title-1","title":"Shopping List 2025"}`
	conn.WriteMessage(websocket.TextMessage, []byte(cmdJSON))

	// Read CommandResponse first
	_, respMsg, err := conn.ReadMessage()
	require.NoError(t, err)
	var cmdResp CommandResponse
	json.Unmarshal(respMsg, &cmdResp)
	assert.Equal(t, "CommandResponse", cmdResp.Type)
	assert.True(t, cmdResp.Success)

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
	cmdJSON := `{"type":"SetListTitle","commandId":"set-title-2","title":"Shared List"}`
	conn1.WriteMessage(websocket.TextMessage, []byte(cmdJSON))

	// Client 1 receives CommandResponse first
	_, respMsg, err1 := conn1.ReadMessage()
	require.NoError(t, err1)
	var cmdResp CommandResponse
	json.Unmarshal(respMsg, &cmdResp)
	assert.True(t, cmdResp.Success)

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

func TestCommandToEvent_CreateCategory_ReuseDeletedCategory(t *testing.T) {
	server, ts, _ := setupTestServer(t)
	defer ts.Close()

	now := time.Now().UTC()

	// Create a category
	server.state.Apply(CategoryCreated{
		Type:      "CategoryCreated",
		ID:        "cat-original",
		Name:      "Shopping",
		CreatedAt: now,
		SortOrder: 1000,
	})

	// Delete the category
	server.state.Apply(CategoryDeleted{
		Type: "CategoryDeleted",
		ID:   "cat-original",
	})

	// Verify category is deleted
	_, ok := server.state.GetCategory("cat-original")
	assert.False(t, ok)

	// Verify we can find it in deleted categories
	deletedID := server.state.FindDeletedCategoryByName("Shopping")
	assert.Equal(t, "cat-original", deletedID)

	// Try to create a new category with the same name
	event, err := server.commandToEvent(CreateCategoryCommand{
		BaseCommand: BaseCommand{Type: "CreateCategory"},
		ID:          "cat-new", // New ID provided
		Name:        "Shopping",
		SortOrder:   2000,
	})
	require.NoError(t, err)

	// Verify the event reuses the old ID instead of the new one
	created := event.(CategoryCreated)
	assert.Equal(t, "cat-original", created.ID, "Should reuse deleted category ID")
	assert.Equal(t, "Shopping", created.Name)
}

func TestCommandToEvent_CreateCategory_ReuseDeletedCategoryCaseInsensitive(t *testing.T) {
	server, ts, _ := setupTestServer(t)
	defer ts.Close()

	now := time.Now().UTC()

	// Create and delete a category
	server.state.Apply(CategoryCreated{
		Type:      "CategoryCreated",
		ID:        "cat-1",
		Name:      "Work",
		CreatedAt: now,
		SortOrder: 1000,
	})

	server.state.Apply(CategoryDeleted{
		Type: "CategoryDeleted",
		ID:   "cat-1",
	})

	// Try to create category with different case - should NOT reuse (case-sensitive)
	event, err := server.commandToEvent(CreateCategoryCommand{
		BaseCommand: BaseCommand{Type: "CreateCategory"},
		ID:          "cat-2",
		Name:        "WORK", // Different case
		SortOrder:   2000,
	})
	require.NoError(t, err)

	// Verify the event uses the NEW ID (not reused because of case mismatch)
	created := event.(CategoryCreated)
	assert.Equal(t, "cat-2", created.ID, "Should NOT reuse deleted category ID (case-sensitive)")
	assert.Equal(t, "WORK", created.Name, "Should preserve the new name casing")
}

func TestCommandToEvent_CreateCategory_ReuseDeletedCategoryExactMatch(t *testing.T) {
	server, ts, _ := setupTestServer(t)
	defer ts.Close()

	now := time.Now().UTC()

	// Create and delete a category
	server.state.Apply(CategoryCreated{
		Type:      "CategoryCreated",
		ID:        "cat-1",
		Name:      "Work",
		CreatedAt: now,
		SortOrder: 1000,
	})

	server.state.Apply(CategoryDeleted{
		Type: "CategoryDeleted",
		ID:   "cat-1",
	})

	// Try to create category with exact same name (case-sensitive match)
	event, err := server.commandToEvent(CreateCategoryCommand{
		BaseCommand: BaseCommand{Type: "CreateCategory"},
		ID:          "cat-2",
		Name:        "Work", // Exact match
		SortOrder:   2000,
	})
	require.NoError(t, err)

	// Verify the event reuses the old ID
	created := event.(CategoryCreated)
	assert.Equal(t, "cat-1", created.ID, "Should reuse deleted category ID (exact case match)")
	assert.Equal(t, "Work", created.Name)
}

func TestCommandToEvent_CreateCategory_NoDeletedCategoryToReuse(t *testing.T) {
	server, ts, _ := setupTestServer(t)
	defer ts.Close()

	// Create a category with no deleted category to reuse
	event, err := server.commandToEvent(CreateCategoryCommand{
		BaseCommand: BaseCommand{Type: "CreateCategory"},
		ID:          "cat-new",
		Name:        "Fresh Category",
		SortOrder:   1000,
	})
	require.NoError(t, err)

	// Verify it uses the provided ID
	created := event.(CategoryCreated)
	assert.Equal(t, "cat-new", created.ID, "Should use provided ID when no deleted category exists")
	assert.Equal(t, "Fresh Category", created.Name)
}

func TestCommandToEvent_CreateCategory_DeletedAndRecreatedMultipleTimes(t *testing.T) {
	server, ts, _ := setupTestServer(t)
	defer ts.Close()

	now := time.Now().UTC()

	// Create, delete, recreate cycle
	server.state.Apply(CategoryCreated{
		Type:      "CategoryCreated",
		ID:        "cat-1",
		Name:      "Temporary",
		CreatedAt: now,
		SortOrder: 1000,
	})

	server.state.Apply(CategoryDeleted{
		Type: "CategoryDeleted",
		ID:   "cat-1",
	})

	// First recreate
	event, err := server.commandToEvent(CreateCategoryCommand{
		BaseCommand: BaseCommand{Type: "CreateCategory"},
		ID:          "cat-2",
		Name:        "Temporary",
		SortOrder:   2000,
	})
	require.NoError(t, err)
	created := event.(CategoryCreated)
	assert.Equal(t, "cat-1", created.ID)

	// Apply the recreate event
	server.state.Apply(created)

	// Delete again
	server.state.Apply(CategoryDeleted{
		Type: "CategoryDeleted",
		ID:   "cat-1",
	})

	// Second recreate - should reuse cat-1 again
	event, err = server.commandToEvent(CreateCategoryCommand{
		BaseCommand: BaseCommand{Type: "CreateCategory"},
		ID:          "cat-3",
		Name:        "Temporary",
		SortOrder:   3000,
	})
	require.NoError(t, err)
	created = event.(CategoryCreated)
	assert.Equal(t, "cat-1", created.ID)
}

func TestCommandToEvent_CreateCategory_RejectDuplicateName(t *testing.T) {
	server, ts, _ := setupTestServer(t)
	defer ts.Close()

	now := time.Now().UTC()

	// Create a category
	server.state.Apply(CategoryCreated{
		Type:      "CategoryCreated",
		ID:        "cat-1",
		Name:      "Work",
		CreatedAt: now,
		SortOrder: 1000,
	})

	// Try to create another category with the same name
	_, err := server.commandToEvent(CreateCategoryCommand{
		BaseCommand: BaseCommand{Type: "CreateCategory"},
		ID:          "cat-2",
		Name:        "Work",
		SortOrder:   2000,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestCommandToEvent_CreateCategory_RejectDuplicateNameCaseSensitive(t *testing.T) {
	server, ts, _ := setupTestServer(t)
	defer ts.Close()

	now := time.Now().UTC()

	// Create a category
	server.state.Apply(CategoryCreated{
		Type:      "CategoryCreated",
		ID:        "cat-1",
		Name:      "Work",
		CreatedAt: now,
		SortOrder: 1000,
	})

	// Try to create category with different case - should be allowed (case-sensitive)
	event, err := server.commandToEvent(CreateCategoryCommand{
		BaseCommand: BaseCommand{Type: "CreateCategory"},
		ID:          "cat-2",
		Name:        "WORK",
		SortOrder:   2000,
	})
	require.NoError(t, err)

	created := event.(CategoryCreated)
	assert.Equal(t, "cat-2", created.ID)
	assert.Equal(t, "WORK", created.Name)
}

func TestCommandToEvent_CreateCategory_AllowAfterDeletion(t *testing.T) {
	server, ts, _ := setupTestServer(t)
	defer ts.Close()

	now := time.Now().UTC()

	// Create a category
	server.state.Apply(CategoryCreated{
		Type:      "CategoryCreated",
		ID:        "cat-1",
		Name:      "Work",
		CreatedAt: now,
		SortOrder: 1000,
	})

	// Try to create duplicate - should fail
	_, err := server.commandToEvent(CreateCategoryCommand{
		BaseCommand: BaseCommand{Type: "CreateCategory"},
		ID:          "cat-2",
		Name:        "Work",
		SortOrder:   2000,
	})
	require.Error(t, err)

	// Delete the category
	server.state.Apply(CategoryDeleted{
		Type: "CategoryDeleted",
		ID:   "cat-1",
	})

	// Now creating with same name should work (and reuse the ID)
	event, err := server.commandToEvent(CreateCategoryCommand{
		BaseCommand: BaseCommand{Type: "CreateCategory"},
		ID:          "cat-2",
		Name:        "Work",
		SortOrder:   3000,
	})
	require.NoError(t, err)

	created := event.(CategoryCreated)
	assert.Equal(t, "cat-1", created.ID, "Should reuse deleted category ID")
	assert.Equal(t, "Work", created.Name)
}

func TestCommandToEvent_RenameCategory_RejectDuplicateName(t *testing.T) {
	server, ts, _ := setupTestServer(t)
	defer ts.Close()

	now := time.Now().UTC()

	// Create two categories
	server.state.Apply(CategoryCreated{
		Type:      "CategoryCreated",
		ID:        "cat-1",
		Name:      "Work",
		CreatedAt: now,
		SortOrder: 1000,
	})

	server.state.Apply(CategoryCreated{
		Type:      "CategoryCreated",
		ID:        "cat-2",
		Name:      "Personal",
		CreatedAt: now,
		SortOrder: 2000,
	})

	// Try to rename cat-2 to "Work" (which already exists)
	_, err := server.commandToEvent(RenameCategoryCommand{
		BaseCommand: BaseCommand{Type: "RenameCategory"},
		ID:          "cat-2",
		Name:        "Work",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestCommandToEvent_RenameCategory_AllowSameName(t *testing.T) {
	server, ts, _ := setupTestServer(t)
	defer ts.Close()

	now := time.Now().UTC()

	// Create a category
	server.state.Apply(CategoryCreated{
		Type:      "CategoryCreated",
		ID:        "cat-1",
		Name:      "Work",
		CreatedAt: now,
		SortOrder: 1000,
	})

	// Rename to the same name (should be allowed - no-op)
	event, err := server.commandToEvent(RenameCategoryCommand{
		BaseCommand: BaseCommand{Type: "RenameCategory"},
		ID:          "cat-1",
		Name:        "Work",
	})
	require.NoError(t, err)

	renamed := event.(CategoryRenamed)
	assert.Equal(t, "cat-1", renamed.ID)
	assert.Equal(t, "Work", renamed.Name)
}

func TestCommandToEvent_RenameCategory_CaseSensitive(t *testing.T) {
	server, ts, _ := setupTestServer(t)
	defer ts.Close()

	now := time.Now().UTC()

	// Create two categories
	server.state.Apply(CategoryCreated{
		Type:      "CategoryCreated",
		ID:        "cat-1",
		Name:      "Work",
		CreatedAt: now,
		SortOrder: 1000,
	})

	server.state.Apply(CategoryCreated{
		Type:      "CategoryCreated",
		ID:        "cat-2",
		Name:      "Personal",
		CreatedAt: now,
		SortOrder: 2000,
	})

	// Rename to "WORK" (different case) - should be allowed
	event, err := server.commandToEvent(RenameCategoryCommand{
		BaseCommand: BaseCommand{Type: "RenameCategory"},
		ID:          "cat-2",
		Name:        "WORK",
	})
	require.NoError(t, err)

	renamed := event.(CategoryRenamed)
	assert.Equal(t, "cat-2", renamed.ID)
	assert.Equal(t, "WORK", renamed.Name)
}

func TestServer_SendErrorMessageOnDuplicateCategory(t *testing.T) {
	server, ts, wsURL := setupTestServer(t)
	defer ts.Close()

	now := time.Now().UTC()

	// Create a category
	server.state.Apply(CategoryCreated{
		Type:      "CategoryCreated",
		ID:        "cat-1",
		Name:      "Work",
		CreatedAt: now,
		SortOrder: 1000,
	})

	// Connect WebSocket client
	conn := connectWS(t, wsURL)
	defer conn.Close()

	// Read initial state rollup
	_, _, err := conn.ReadMessage()
	require.NoError(t, err)

	// Read client count message
	_, _, err = conn.ReadMessage()
	require.NoError(t, err)

	// Try to create duplicate category with commandId
	command := map[string]interface{}{
		"type":      "CreateCategory",
		"commandId": "test-command-1",
		"id":        "cat-2",
		"name":      "Work",
	}
	err = conn.WriteJSON(command)
	require.NoError(t, err)

	// Should receive error response
	_, message, err := conn.ReadMessage()
	require.NoError(t, err)

	var response CommandResponse
	err = json.Unmarshal(message, &response)
	require.NoError(t, err)

	assert.Equal(t, "CommandResponse", response.Type)
	assert.Equal(t, "test-command-1", response.CommandID)
	assert.False(t, response.Success)
	assert.Contains(t, response.Error, "already exists")
}
