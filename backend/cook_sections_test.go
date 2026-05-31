package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

// TestCookCommand_StepIndexValidatedAgainstTotalSteps boots a real
// Server with an attached RecipeStore, seeds a 2+2 multi-section
// recipe, and asserts that the WebSocket cook handler accepts flat
// step indices across all sections (the bug fix from recipes.go's
// new recipeTotalSteps helper). Before the fix, only indices 0 and 1
// would be accepted because the handler used len(recipe.Instructions)
// on a Recipe struct that no longer carries that field.
func TestCookCommand_StepIndexValidatedAgainstTotalSteps(t *testing.T) {
	tmp := t.TempDir()
	es, err := NewEventStore(filepath.Join(tmp, "events.jsonl"))
	require.NoError(t, err)
	server := NewServer(es)

	store, err := NewRecipeStore(filepath.Join(tmp, "recipes"), tmp, 1_000_000)
	require.NoError(t, err)
	server.SetRecipeStore(store)

	imgBytes := makeTestPNG(t, 16, 16)
	id := uuid.NewString()
	_, err = store.Save(Recipe{
		ID:    id,
		Title: "x",
		Sections: []RecipeSection{
			{Ingredients: []Ingredient{{Name: "A"}}, Instructions: []string{"a1", "a2"}},
			{Ingredients: []Ingredient{{Name: "B"}}, Instructions: []string{"b1", "b2"}},
		},
	}, imgBytes, "image/png")
	require.NoError(t, err)

	go server.Run()
	ts := httptest.NewServer(http.HandlerFunc(server.HandleWebSocket))
	defer ts.Close()
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	send := func(stepIndex int, commandID string) map[string]any {
		cmd := map[string]any{
			"type":      "CookCheckStep",
			"commandId": commandID,
			"recipeId":  id,
			"stepIndex": stepIndex,
		}
		raw, _ := json.Marshal(cmd)
		require.NoError(t, conn.WriteMessage(websocket.TextMessage, raw))
		// Read until we see OUR CommandResponse. Initial rollups and
		// other clients' messages are ignored without a fixed-count
		// drain that could otherwise consume our response.
		for {
			conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			var msg map[string]any
			require.NoError(t, conn.ReadJSON(&msg))
			if msg["type"] == "CommandResponse" && msg["commandId"] == commandID {
				return msg
			}
		}
	}

	// Index 3 is the last step in section 2 (flat indexing across
	// sections). Must be accepted.
	resp := send(3, "ok-3")
	require.Equal(t, true, resp["success"], "stepIndex=3 must be accepted with 2+2 sections")

	// Index 4 is one past the end. Must be rejected with the
	// existing out-of-range error message.
	resp = send(4, "bad-4")
	require.Equal(t, false, resp["success"])
	require.Equal(t, "step index out of range", resp["error"])

	// Negative index also rejected.
	resp = send(-1, "bad-neg")
	require.Equal(t, false, resp["success"])
}
