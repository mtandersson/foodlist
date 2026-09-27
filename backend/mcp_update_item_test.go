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
	"github.com/stretchr/testify/require"
)

func TestMCP_UpdateItemState(t *testing.T) {
	store, err := NewEventStore(filepath.Join(t.TempDir(), "events.jsonl"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	srv := NewServer(store)
	require.NoError(t, srv.ExecuteCommand(CreateTodoCommand{
		BaseCommand: BaseCommand{Type: "CreateTodo", CommandID: "seed"},
		ID:          "milk", Name: "Milk",
	}))
	<-srv.broadcast
	base := initRecipeMCP(t, srv)

	call := func(id int, args map[string]any) map[string]any {
		t.Helper()
		return toolCall(t, base, id, "foodlist_update_item", args)
	}
	check := func(result map[string]any, done, starred bool) {
		t.Helper()
		require.NotEqual(t, true, result["isError"])
		item := result["structuredContent"].(map[string]any)
		require.Equal(t, "milk", item["id"])
		require.Equal(t, "Milk", item["name"])
		require.Equal(t, starred, item["starred"])
		require.Equal(t, done, item["completedAt"] != nil)
		require.Contains(t, firstTextContent(t, result), "Milk")
	}

	start := store.SyncCount()
	check(call(2, map[string]any{"todo_id": "milk", "done": true, "starred": true}), true, true)
	require.Equal(t, start+1, store.SyncCount(), "both events use one durable write")
	for _, want := range []string{"TodoCompleted", "TodoStarred"} {
		var event map[string]any
		require.NoError(t, json.Unmarshal(<-srv.broadcast, &event))
		require.Equal(t, want, event["type"])
	}
	require.Empty(t, srv.broadcast)

	check(call(3, map[string]any{"todo_id": "milk", "done": true, "starred": true}), true, true)
	require.Equal(t, start+1, store.SyncCount(), "identical update is a no-op")
	require.Empty(t, srv.broadcast)
	check(call(4, map[string]any{"todo_id": "milk", "starred": false}), true, false)
	check(call(5, map[string]any{"todo_id": "milk", "done": false}), false, false)
	check(call(6, map[string]any{"todo_id": "milk", "done": false, "starred": true}), false, true)
	check(call(7, map[string]any{"todo_id": "milk", "done": true, "starred": false}), true, false)

	beforeInvalid := store.SyncCount()
	for id, args := range []map[string]any{
		{"todo_id": "milk"},
		{"todo_id": "missing", "done": true},
	} {
		result := call(10+id, args)
		require.Equal(t, true, result["isError"])
	}
	require.Equal(t, beforeInvalid, store.SyncCount())
	item, ok := srv.state.GetTodo("milk")
	require.True(t, ok)
	require.NotNil(t, item.CompletedAt)
	require.False(t, item.Starred)
}

func TestMCP_UpdateItemBroadcastsInOrder(t *testing.T) {
	srv, cleanup := newServerWithTempStore(t)
	defer cleanup()
	require.NoError(t, srv.ExecuteCommand(CreateTodoCommand{
		BaseCommand: BaseCommand{Type: "CreateTodo", CommandID: "seed"},
		ID:          "milk", Name: "Milk",
	}))
	go srv.Run()
	wsServer := httptest.NewServer(http.HandlerFunc(srv.HandleWebSocket))
	defer wsServer.Close()
	ws, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(wsServer.URL, "http"), nil)
	require.NoError(t, err)
	defer ws.Close()
	readInitialMessages(t, ws)

	base := initRecipeMCP(t, srv)
	result := toolCall(t, base, 2, "foodlist_update_item", map[string]any{
		"todo_id": "milk", "done": true, "starred": true,
	})
	require.NotEqual(t, true, result["isError"])
	require.NoError(t, ws.SetReadDeadline(time.Now().Add(3*time.Second)))
	for _, want := range []string{"TodoCompleted", "TodoStarred"} {
		var event map[string]any
		_, wire, err := ws.ReadMessage()
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(wire, &event))
		require.Equal(t, want, event["type"])
		require.Equal(t, "milk", event["id"])
	}
}

func TestExecuteUpdateItem_PersistenceFailureLeavesStateUnchanged(t *testing.T) {
	store, err := NewEventStore(filepath.Join(t.TempDir(), "events.jsonl"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	srv := NewServer(store)
	require.NoError(t, srv.ExecuteCommand(CreateTodoCommand{
		BaseCommand: BaseCommand{Type: "CreateTodo", CommandID: "seed"},
		ID:          "milk", Name: "Milk",
	}))
	<-srv.broadcast
	require.NoError(t, store.file.Close())
	done, starred := true, true
	_, err = srv.ExecuteUpdateItem("milk", &done, &starred)
	require.ErrorContains(t, err, "failed to persist item update")
	item, ok := srv.state.GetTodo("milk")
	require.True(t, ok)
	require.Nil(t, item.CompletedAt)
	require.False(t, item.Starred)
	require.Empty(t, srv.broadcast)
}
