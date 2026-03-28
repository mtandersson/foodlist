package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCP_InitializeAndListTools(t *testing.T) {
	srv, cleanup := newServerWithTempStore(t)
	defer cleanup()

	ts := httptest.NewServer(foodlistMCPHandler(srv))
	defer ts.Close()

	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	req, err := http.NewRequest(http.MethodPost, ts.URL, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var env struct {
		Result struct {
			ServerInfo struct {
				Name string `json:"name"`
			} `json:"serverInfo"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal(b, &env))
	assert.Equal(t, "foodlist", env.Result.ServerInfo.Name)
}

func TestExecuteCommand_CreateTodo(t *testing.T) {
	srv, cleanup := newServerWithTempStore(t)
	defer cleanup()

	err := srv.ExecuteCommand(CreateTodoCommand{
		BaseCommand: BaseCommand{Type: "CreateTodo", CommandID: "c1"},
		ID:          "todo-1",
		Name:        "  Milk  ",
	})
	require.NoError(t, err)
	td, ok := srv.state.GetTodo("todo-1")
	require.True(t, ok)
	assert.Equal(t, "Milk", td.Name)
}

func newServerWithTempStore(t *testing.T) (*Server, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "events.jsonl")
	store, err := NewEventStore(path)
	require.NoError(t, err)
	srv := NewServer(store)
	require.NoError(t, srv.LoadEvents())
	return srv, func() { store.Close() }
}
