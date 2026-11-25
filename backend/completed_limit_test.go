package main

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// completedHistoryEvents creates open open items and done completed items.
// Completed item i finishes at base+i minutes, so "done-<done-1>" is newest.
func completedHistoryEvents(open, done int) []Event {
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	var events []Event
	for i := 0; i < open; i++ {
		events = append(events, TodoCreated{Type: "TodoCreated", ID: fmt.Sprintf("open-%d", i), Name: "Open", CreatedAt: base.AddDate(-1, 0, 0), SortOrder: 100000 + i})
	}
	for i := 0; i < done; i++ {
		id := fmt.Sprintf("done-%d", i)
		// Sort order runs opposite to completion time so the limit must use completedAt.
		events = append(events,
			TodoCreated{Type: "TodoCreated", ID: id, Name: "Done", CreatedAt: base, SortOrder: done - i},
			TodoCompleted{Type: "TodoCompleted", ID: id, CompletedAt: base.Add(time.Duration(i) * time.Minute)},
		)
	}
	return events
}

func idsOf(todos []Todo) map[string]bool {
	ids := make(map[string]bool, len(todos))
	for _, t := range todos {
		ids[t.ID] = true
	}
	return ids
}

func TestState_GetListSnapshotKeepsOpenItemsAndMostRecentlyCompleted(t *testing.T) {
	for _, tc := range []struct {
		name          string
		done, limit   int
		wantCompleted int
	}{
		{"fewer than limit", 3, 50, 3},
		{"exactly limit", 50, 50, 50},
		{"over limit", 60, 50, 50},
		{"no limit", 60, -1, 60},
		{"zero limit", 5, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := NewState()
			state.ApplyEvents(completedHistoryEvents(2, tc.done))

			snap := state.GetListSnapshot(tc.limit)

			assert.Equal(t, tc.done, snap.CompletedTotal)
			require.Len(t, snap.Todos, 2+tc.wantCompleted)
			ids := idsOf(snap.Todos)
			assert.True(t, ids["open-0"] && ids["open-1"], "all open items are kept")
			for i := tc.done - tc.wantCompleted; i < tc.done; i++ {
				assert.True(t, ids[fmt.Sprintf("done-%d", i)], "done-%d is among the most recently completed", i)
			}
			for i := 1; i < len(snap.Todos); i++ {
				assert.GreaterOrEqual(t, snap.Todos[i-1].SortOrder, snap.Todos[i].SortOrder)
			}
			assert.Len(t, state.GetTodos(), 2+tc.done, "history is retained")
		})
	}
}

func readStateRollup(t *testing.T, conn *websocket.Conn) StateRollup {
	t.Helper()
	for i := 0; i < 10; i++ {
		require.NoError(t, conn.SetReadDeadline(time.Now().Add(2*time.Second)))
		_, msg, err := conn.ReadMessage()
		require.NoError(t, err)
		var rollup StateRollup
		if json.Unmarshal(msg, &rollup) == nil && rollup.Type == "StateRollup" {
			return rollup
		}
	}
	t.Fatal("no StateRollup received")
	return StateRollup{}
}

func TestServer_StateRollupLimitsCompletedUntilClientRequestsAll(t *testing.T) {
	server, ts, wsURL := setupTestServer(t)
	defer ts.Close()
	server.state.ApplyEvents(completedHistoryEvents(1, DefaultCompletedLimit+10))

	conn := connectWS(t, wsURL)
	defer conn.Close()

	initial := readStateRollup(t, conn)
	assert.Equal(t, DefaultCompletedLimit+10, initial.CompletedTotal)
	require.Len(t, initial.Todos, 1+DefaultCompletedLimit)
	ids := idsOf(initial.Todos)
	assert.True(t, ids["open-0"])
	assert.True(t, ids[fmt.Sprintf("done-%d", DefaultCompletedLimit+9)], "newest completed item is sent")
	assert.False(t, ids["done-0"], "oldest completed item is withheld")

	require.NoError(t, conn.WriteJSON(RequestStateRollup{Type: "RequestStateRollup", IncludeAllCompleted: true}))
	full := readStateRollup(t, conn)
	assert.Equal(t, DefaultCompletedLimit+10, full.CompletedTotal)
	assert.Len(t, full.Todos, 1+DefaultCompletedLimit+10)

	require.NoError(t, conn.WriteJSON(RequestStateRollup{Type: "RequestStateRollup"}))
	assert.Len(t, readStateRollup(t, conn).Todos, 1+DefaultCompletedLimit)
}

func TestMCP_ListLimitsCompletedUnlessAllRequested(t *testing.T) {
	store, err := NewEventStore(filepath.Join(t.TempDir(), "events.jsonl"))
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	srv := NewServer(store)
	srv.state.ApplyEvents(completedHistoryEvents(1, DefaultCompletedLimit+5))
	ts := httptest.NewServer(foodlistMCPHandler(srv))
	t.Cleanup(ts.Close)
	base := ts.URL
	require.NoError(t, mcpOK(t, base, 1, "initialize", map[string]any{"protocolVersion": "2025-06-18", "capabilities": map[string]any{}, "clientInfo": map[string]any{"name": "limit-test", "version": "1"}}))

	countCompleted := func(res map[string]any) (int, float64) {
		structured := res["structuredContent"].(map[string]any)
		n := 0
		for _, entry := range structured["items"].([]any) {
			if entry.(map[string]any)["completed"] == true {
				n++
			}
		}
		return n, structured["completedTotal"].(float64)
	}

	limited := toolCall(t, base, 2, "foodlist_list", map[string]any{})
	n, total := countCompleted(limited)
	assert.Equal(t, DefaultCompletedLimit, n)
	assert.Equal(t, float64(DefaultCompletedLimit+5), total)
	assert.Contains(t, firstTextContent(t, limited), "include_all_completed")

	all := toolCall(t, base, 3, "foodlist_list", map[string]any{"include_all_completed": true})
	n, _ = countCompleted(all)
	assert.Equal(t, DefaultCompletedLimit+5, n)
	assert.NotContains(t, firstTextContent(t, all), "include_all_completed")

	var rollup StateRollup
	require.NoError(t, json.Unmarshal([]byte(resourceText(t, base, 4, mcpResourceState)), &rollup))
	assert.Equal(t, DefaultCompletedLimit+5, rollup.CompletedTotal)
	assert.Len(t, rollup.Todos, 1+DefaultCompletedLimit)

	var todos []Todo
	require.NoError(t, json.Unmarshal([]byte(resourceText(t, base, 5, mcpResourceTodos)), &todos))
	assert.Len(t, todos, 1+DefaultCompletedLimit)
}
