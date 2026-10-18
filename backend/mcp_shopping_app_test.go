package main

import (
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCP_ShoppingListStructuredViewAndMutation(t *testing.T) {
	store, err := NewEventStore(filepath.Join(t.TempDir(), "events.jsonl"))
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	srv := NewServer(store)
	categoryID := uuid.NewString()
	require.NoError(t, srv.ExecuteCommand(CreateCategoryCommand{BaseCommand: BaseCommand{Type: "CreateCategory", CommandID: uuid.NewString()}, ID: categoryID, Name: "Dairy <img onerror=alert(1)>"}))
	count, unit := 2.5, "litres"
	itemID := uuid.NewString()
	require.NoError(t, srv.ExecuteCommand(CreateTodoCommand{BaseCommand: BaseCommand{Type: "CreateTodo", CommandID: uuid.NewString()}, ID: itemID, Name: "Milk <script>alert(1)</script>", CategoryID: &categoryID, Count: &count, Unit: &unit}))
	bareID := uuid.NewString()
	require.NoError(t, srv.ExecuteCommand(CreateTodoCommand{BaseCommand: BaseCommand{Type: "CreateTodo", CommandID: uuid.NewString()}, ID: bareID, Name: "Bread"}))
	ts := httptest.NewServer(foodlistMCPHandler(srv))
	t.Cleanup(ts.Close)
	base := ts.URL
	require.NoError(t, mcpOK(t, base, 1, "initialize", map[string]any{"protocolVersion": "2025-06-18", "capabilities": map[string]any{}, "clientInfo": map[string]any{"name": "shopping-test", "version": "1"}}))

	tools := mcpResult(t, base, 2, "tools/list", map[string]any{})["tools"].([]any)
	var listTool map[string]any
	for _, entry := range tools {
		tool := entry.(map[string]any)
		if tool["name"] == "foodlist_list" {
			listTool = tool
			break
		}
	}
	require.NotNil(t, listTool)
	require.Equal(t, mcpShoppingAppURI, listTool["_meta"].(map[string]any)["ui"].(map[string]any)["resourceUri"])

	resources := mcpResult(t, base, 3, "resources/list", map[string]any{})["resources"].([]any)
	var uiResource map[string]any
	for _, entry := range resources {
		resource := entry.(map[string]any)
		if resource["uri"] == mcpShoppingAppURI {
			uiResource = resource
			break
		}
	}
	require.NotNil(t, uiResource)
	require.Equal(t, "text/html;profile=mcp-app", uiResource["mimeType"])
	contents := mcpResult(t, base, 4, "resources/read", map[string]any{"uri": mcpShoppingAppURI})["contents"].([]any)
	view := contents[0].(map[string]any)
	require.Equal(t, "text/html;profile=mcp-app", view["mimeType"])
	html := view["text"].(string)
	require.True(t, strings.HasPrefix(html, "<!doctype html>"))
	require.Contains(t, html, "foodlist_update_item")
	require.NotContains(t, html, "Milk <script>")
	ui := view["_meta"].(map[string]any)["ui"].(map[string]any)
	require.Empty(t, ui["csp"].(map[string]any)["connectDomains"])
	require.NotContains(t, resourceText(t, base, 9, mcpResourceTodos), "\n")

	list := toolCall(t, base, 5, "foodlist_list", map[string]any{})
	text := firstTextContent(t, list)
	require.Contains(t, text, "### Dairy <img onerror=alert(1)>")
	require.Contains(t, text, "2.5 litres")
	require.Contains(t, text, "### Uncategorized")
	structured := list["structuredContent"].(map[string]any)
	require.Equal(t, false, structured["includeCompleted"])
	require.Equal(t, 2, len(structured["items"].([]any)))
	require.Equal(t, 1, len(structured["categories"].([]any)))
	for _, entry := range structured["items"].([]any) {
		item := entry.(map[string]any)
		require.Contains(t, item, "count")
		require.Contains(t, item, "unit")
		require.Contains(t, item, "categoryId")
		require.Equal(t, false, item["completed"])
		require.Equal(t, false, item["starred"])
		require.Contains(t, item, "sortOrder")
		if item["id"] == bareID {
			require.Nil(t, item["count"])
			require.Nil(t, item["unit"])
			require.Nil(t, item["categoryId"])
		}
	}

	updated := toolCall(t, base, 6, "foodlist_update_item", map[string]any{"todo_id": itemID, "done": true, "starred": true})
	require.NotEqual(t, true, updated["isError"])
	returnedItem := updated["structuredContent"].(map[string]any)
	require.Equal(t, itemID, returnedItem["id"])
	require.Equal(t, true, returnedItem["starred"])
	require.NotNil(t, returnedItem["completedAt"])
	open := toolCall(t, base, 7, "foodlist_list", map[string]any{"include_completed": false})
	require.Equal(t, false, open["structuredContent"].(map[string]any)["includeCompleted"])
	require.Len(t, open["structuredContent"].(map[string]any)["items"].([]any), 1)
	require.NotContains(t, firstTextContent(t, open), itemID)
	all := toolCall(t, base, 8, "foodlist_list", map[string]any{})
	require.Equal(t, false, all["structuredContent"].(map[string]any)["includeCompleted"])
	require.Len(t, all["structuredContent"].(map[string]any)["items"].([]any), 1)
	require.NotContains(t, firstTextContent(t, all), itemID)

	explicitHistory := toolCall(t, base, 10, "foodlist_list", map[string]any{"include_completed": true})
	require.Equal(t, false, explicitHistory["structuredContent"].(map[string]any)["includeCompleted"])
	require.Len(t, explicitHistory["structuredContent"].(map[string]any)["items"].([]any), 1)
	require.NotContains(t, firstTextContent(t, explicitHistory), itemID)

	// MCP resources expose the current list too, not completed history.
	todosResource := resourceText(t, base, 11, mcpResourceTodos)
	require.Contains(t, todosResource, bareID)
	require.NotContains(t, todosResource, itemID)
	stateResource := resourceText(t, base, 12, mcpResourceState)
	require.Contains(t, stateResource, bareID)
	require.NotContains(t, stateResource, itemID)
}
