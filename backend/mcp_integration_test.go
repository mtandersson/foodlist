package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// Exercises every MCP tool and resource and checks JSON shapes against
// schema/events.schema.json (StateRollup, Todo, Category). WebSocket commands
// use camelCase (categoryId); MCP tool arguments use snake_case (category_id).

func TestMCP_FullIntegration_MatchesEventsSchemaShape(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "events.jsonl")
	store, err := NewEventStore(path)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	srv := NewServer(store)
	require.NoError(t, srv.LoadEvents())

	catID := uuid.MustParse("11111111-1111-4111-8111-111111111111").String()
	require.NoError(t, srv.ExecuteCommand(CreateCategoryCommand{
		BaseCommand: BaseCommand{Type: "CreateCategory", CommandID: "seed-cat"},
		ID:          catID,
		Name:        "Dairy",
	}))

	ts := httptest.NewServer(foodlistMCPHandler(srv))
	t.Cleanup(ts.Close)
	base := ts.URL

	require.NoError(t, mcpOK(t, base, 1, "initialize", map[string]any{
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "integration", "version": "1"},
	}))

	// --- resources/list ---
	resList := mcpResult(t, base, 2, "resources/list", map[string]any{})
	resources, ok := resList["resources"].([]any)
	require.True(t, ok, "resources/list: resources array")
	uris := make([]string, 0, len(resources))
	for _, r := range resources {
		m, _ := r.(map[string]any)
		uris = append(uris, m["uri"].(string))
	}
	for _, want := range []string{mcpResourceState, mcpResourceCategories, mcpResourceTodos} {
		require.Contains(t, uris, want, "resources/list should include %s", want)
	}

	// --- resources/read + schema-shaped validation ---
	stateText := resourceText(t, base, 10, mcpResourceState)
	var rollup map[string]any
	require.NoError(t, json.Unmarshal([]byte(stateText), &rollup))
	assertStateRollupLikeSchema(t, rollup)

	catText := resourceText(t, base, 11, mcpResourceCategories)
	var cats []any
	require.NoError(t, json.Unmarshal([]byte(catText), &cats))
	require.NotEmpty(t, cats)
	for _, c := range cats {
		assertCategoryLikeSchema(t, c.(map[string]any))
	}

	todoText := resourceText(t, base, 12, mcpResourceTodos)
	var todos []any
	require.NoError(t, json.Unmarshal([]byte(todoText), &todos))
	_ = todos // may be empty until add

	// --- tools/list ---
	toolsRes := mcpResult(t, base, 3, "tools/list", map[string]any{})
	tools, _ := toolsRes["tools"].([]any)
	names := make([]string, 0, len(tools))
	for _, x := range tools {
		names = append(names, x.(map[string]any)["name"].(string))
	}
	slices.Sort(names)
	require.Equal(t, []string{
		"foodlist_add",
		"foodlist_categories",
		"foodlist_categorize",
		"foodlist_list",
		"foodlist_mark_done",
		"foodlist_mark_starred",
	}, names)

	// --- foodlist_categories (JSON matches resource + schema shape) ---
	catToolOut := toolCall(t, base, 4, "foodlist_categories", map[string]any{})
	catToolTxt := firstTextContent(t, catToolOut)
	var catsFromTool []any
	require.NoError(t, json.Unmarshal([]byte(catToolTxt), &catsFromTool))
	require.NotEmpty(t, catsFromTool)
	for _, c := range catsFromTool {
		assertCategoryLikeSchema(t, c.(map[string]any))
	}

	// --- foodlist_add (empty name -> error tool result) ---
	addErr := toolCall(t, base, 20, "foodlist_add", map[string]any{"name": "   "})
	require.True(t, addErr["isError"].(bool), "empty name should be isError")

	// --- foodlist_add OK ---
	addOK := toolCall(t, base, 21, "foodlist_add", map[string]any{"name": "  Milk  "})
	if isErr, ok := addOK["isError"].(bool); ok {
		require.False(t, isErr)
	}
	txt := firstTextContent(t, addOK)
	todoID := parseCreatedTodoID(t, txt)
	require.NotEmpty(t, todoID)

	// --- foodlist_categorize invalid category ---
	badCat := toolCall(t, base, 22, "foodlist_categorize", map[string]any{
		"todo_id":     todoID,
		"category_id": uuid.NewString(),
	})
	require.Equal(t, true, badCat["isError"])

	// --- foodlist_categorize OK ---
	okCat := toolCall(t, base, 23, "foodlist_categorize", map[string]any{
		"todo_id":     todoID,
		"category_id": catID,
	})
	require.NotEqual(t, true, okCat["isError"])

	// Re-read todos resource; todo should have categoryId matching schema (camelCase JSON)
	todoText2 := resourceText(t, base, 24, mcpResourceTodos)
	require.NoError(t, json.Unmarshal([]byte(todoText2), &todos))
	var milk map[string]any
	for _, x := range todos {
		m := x.(map[string]any)
		if m["name"] == "Milk" {
			milk = m
			break
		}
	}
	require.NotNil(t, milk)
	assertTodoLikeSchema(t, milk)
	require.Equal(t, catID, milk["categoryId"])

	// --- foodlist_list (markdown contains name + id) ---
	listOut := toolCall(t, base, 25, "foodlist_list", map[string]any{})
	listTxt := firstTextContent(t, listOut)
	require.Contains(t, listTxt, "Milk")
	require.Contains(t, listTxt, todoID)
	require.Contains(t, listTxt, "Dairy")

	// --- foodlist_mark_starred ---
	star := toolCall(t, base, 26, "foodlist_mark_starred", map[string]any{"todo_id": todoID, "starred": true})
	require.NotEqual(t, true, star["isError"])
	listStar := firstTextContent(t, toolCall(t, base, 27, "foodlist_list", map[string]any{}))
	require.Contains(t, listStar, "★")

	unstar := toolCall(t, base, 28, "foodlist_mark_starred", map[string]any{"todo_id": todoID, "starred": false})
	require.NotEqual(t, true, unstar["isError"])

	// --- foodlist_mark_done + foodlist_list include_completed ---
	done := toolCall(t, base, 29, "foodlist_mark_done", map[string]any{"todo_id": todoID, "done": true})
	require.NotEqual(t, true, done["isError"])

	openOnly := firstTextContent(t, toolCall(t, base, 30, "foodlist_list", map[string]any{"include_completed": false}))
	require.Contains(t, openOnly, "No matching todos")

	undo := toolCall(t, base, 31, "foodlist_mark_done", map[string]any{"todo_id": todoID, "done": false})
	require.NotEqual(t, true, undo["isError"])

	// --- foodlist_categorize clear (null category_id) ---
	clearCat := toolCall(t, base, 32, "foodlist_categorize", map[string]any{
		"todo_id":     todoID,
		"category_id": nil,
	})
	require.NotEqual(t, true, clearCat["isError"])
	stateAfter := resourceText(t, base, 33, mcpResourceState)
	require.NoError(t, json.Unmarshal([]byte(stateAfter), &rollup))
	todosArr, _ := rollup["todos"].([]any)
	var milk2 map[string]any
	for _, x := range todosArr {
		m := x.(map[string]any)
		if m["id"] == todoID {
			milk2 = m
			break
		}
	}
	require.NotNil(t, milk2)
	cid, hasCID := milk2["categoryId"]
	require.True(t, !hasCID || cid == nil, "expected category cleared (missing or null categoryId), got %v", cid)
}

func mcpOK(t *testing.T, base string, id int, method string, params map[string]any) error {
	t.Helper()
	_, err := mcpEnvelope(t, base, id, method, params)
	return err
}

func mcpResult(t *testing.T, base string, id int, method string, params map[string]any) map[string]any {
	t.Helper()
	env, err := mcpEnvelope(t, base, id, method, params)
	require.NoError(t, err)
	if env["error"] != nil {
		t.Fatalf("jsonrpc error: %v", env["error"])
	}
	res, ok := env["result"].(map[string]any)
	require.True(t, ok, "expected result object")
	return res
}

func mcpEnvelope(t *testing.T, base string, id int, method string, params map[string]any) (map[string]any, error) {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	})
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, base, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "body=%s", b)
	var env map[string]any
	if err := json.Unmarshal(b, &env); err != nil {
		return nil, err
	}
	return env, nil
}

func resourceText(t *testing.T, base string, id int, uri string) string {
	t.Helper()
	res := mcpResult(t, base, id, "resources/read", map[string]any{"uri": uri})
	contents, ok := res["contents"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, contents)
	c0 := contents[0].(map[string]any)
	text, _ := c0["text"].(string)
	require.NotEmpty(t, text)
	return text
}

func toolCall(t *testing.T, base string, id int, name string, arguments map[string]any) map[string]any {
	t.Helper()
	return mcpResult(t, base, id, "tools/call", map[string]any{
		"name":      name,
		"arguments": arguments,
	})
}

func firstTextContent(t *testing.T, result map[string]any) string {
	t.Helper()
	arr, ok := result["content"].([]any)
	require.True(t, ok)
	require.NotEmpty(t, arr)
	c0 := arr[0].(map[string]any)
	return c0["text"].(string)
}

var createdTodoRE = regexp.MustCompile(`Created todo ([0-9a-f-]{36})`)

func parseCreatedTodoID(t *testing.T, s string) string {
	t.Helper()
	m := createdTodoRE.FindStringSubmatch(s)
	require.Len(t, m, 2, "text=%q", s)
	return m[1]
}

// --- Shape checks mirroring schema/events.schema.json (definitions) ---

func assertStateRollupLikeSchema(t *testing.T, m map[string]any) {
	t.Helper()
	require.Equal(t, "StateRollup", m["type"])
	require.NotNil(t, m["todos"])
	require.NotNil(t, m["categories"])
	require.NotNil(t, m["listTitle"])
	_, hasVersion := m["version"]
	_ = hasVersion // optional in JSON if empty, but struct usually omits — OK either way
	_, ok := m["todos"].([]any)
	require.True(t, ok)
	_, ok = m["categories"].([]any)
	require.True(t, ok)
}

func assertTodoLikeSchema(t *testing.T, m map[string]any) {
	t.Helper()
	for _, k := range []string{"id", "name", "createdAt", "sortOrder", "starred"} {
		require.Contains(t, m, k, "todo missing %q", k)
	}
	if ca, ok := m["completedAt"]; ok && ca != nil {
		_, isStr := ca.(string)
		require.True(t, isStr)
	}
}

func assertCategoryLikeSchema(t *testing.T, m map[string]any) {
	t.Helper()
	for _, k := range []string{"id", "name", "createdAt", "sortOrder"} {
		require.Contains(t, m, k, "category missing %q", k)
	}
}
