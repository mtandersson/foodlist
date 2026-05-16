package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	mcpHTTPPath = "/mcp"

	mcpResourceState       = "foodlist://state"
	mcpResourceCategories  = "foodlist://categories"
	mcpResourceSuggestions = "foodlist://suggestions"
	// Legacy URI kept for compatibility with existing MCP clients.
	mcpResourceTodos = "foodlist://todos"
)

type foodlistListIn struct {
	IncludeCompleted *bool `json:"include_completed,omitempty"`
}

type foodlistAddIn struct {
	Name       string  `json:"name"`
	CategoryID *string `json:"category_id,omitempty"`
}

type foodlistCategorizeIn struct {
	TodoID     string  `json:"todo_id"`
	CategoryID *string `json:"category_id,omitempty"`
}

type foodlistMarkDoneIn struct {
	TodoID string `json:"todo_id"`
	Done   bool   `json:"done"`
}

type foodlistMarkStarredIn struct {
	TodoID  string `json:"todo_id"`
	Starred bool   `json:"starred"`
}

func newFoodlistMCPServer(app *Server) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "foodlist", Version: version}, nil)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "foodlist_list",
		Description: "List grocery items as markdown, grouped by category (only categories that appear on at least one shown item). For every defined category ID, use foodlist_categories or the foodlist://categories resource.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in foodlistListIn) (*mcp.CallToolResult, any, error) {
		_ = ctx
		_ = req
		includeCompleted := true
		if in.IncludeCompleted != nil {
			includeCompleted = *in.IncludeCompleted
		}
		cats := app.state.GetCategories()
		catName := make(map[string]string, len(cats))
		for _, c := range cats {
			catName[c.ID] = c.Name
		}
		var b strings.Builder
		_, _ = fmt.Fprintf(&b, "**%s**\n\n", app.state.GetListTitle())
		n := 0
		for _, t := range app.state.GetTodos() {
			if !includeCompleted && t.CompletedAt != nil {
				continue
			}
			n++
			catLabel := "(uncategorized)"
			if t.CategoryID != nil {
				if name, ok := catName[*t.CategoryID]; ok {
					catLabel = name
				} else {
					catLabel = *t.CategoryID
				}
			}
			status := "open"
			if t.CompletedAt != nil {
				status = "done"
			}
			star := ""
			if t.Starred {
				star = " ★"
			}
			_, _ = fmt.Fprintf(&b, "- **%s** `%s` — %s%s (%s)\n", t.Name, t.ID, catLabel, star, status)
		}
		if n == 0 {
			b.WriteString("_No matching grocery items._\n")
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: b.String()}},
		}, nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "foodlist_categories",
		Description: "Return all defined categories as a JSON array (id, name, sortOrder, etc.), including categories not used by any grocery item. Same data as the foodlist://categories resource.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct{}) (*mcp.CallToolResult, any, error) {
		_ = ctx
		_ = req
		_ = in
		b, err := json.MarshalIndent(app.state.GetCategories(), "", "  ")
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil, nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
		}, nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "foodlist_add",
		Description: "Create a new grocery item. IDs are generated server-side.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in foodlistAddIn) (*mcp.CallToolResult, any, error) {
		_ = ctx
		_ = req
		cmd := CreateTodoCommand{
			BaseCommand: BaseCommand{Type: "CreateTodo", CommandID: uuid.NewString()},
			ID:          uuid.NewString(),
			Name:        in.Name,
			CategoryID:  in.CategoryID,
		}
		if err := app.ExecuteCommand(cmd); err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil, nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Created grocery item %s (%s).", cmd.ID, strings.TrimSpace(in.Name))}},
		}, nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "foodlist_categorize",
		Description: "Assign a grocery item to a category, or clear its category.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in foodlistCategorizeIn) (*mcp.CallToolResult, any, error) {
		_ = ctx
		_ = req
		cmd := CategorizeTodoCommand{
			BaseCommand: BaseCommand{Type: "CategorizeTodo", CommandID: uuid.NewString()},
			ID:          in.TodoID,
			CategoryID:  in.CategoryID,
		}
		if err := app.ExecuteCommand(cmd); err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil, nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Updated category for grocery item %s.", in.TodoID)}},
		}, nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "foodlist_mark_done",
		Description: "Mark a grocery item completed or reopen it.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in foodlistMarkDoneIn) (*mcp.CallToolResult, any, error) {
		_ = ctx
		_ = req
		var cmd Command
		if in.Done {
			cmd = CompleteTodoCommand{
				BaseCommand: BaseCommand{Type: "CompleteTodo", CommandID: uuid.NewString()},
				ID:          in.TodoID,
			}
		} else {
			cmd = UncompleteTodoCommand{
				BaseCommand: BaseCommand{Type: "UncompleteTodo", CommandID: uuid.NewString()},
				ID:          in.TodoID,
			}
		}
		if err := app.ExecuteCommand(cmd); err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil, nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Grocery item %s done=%v.", in.TodoID, in.Done)}},
		}, nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "foodlist_mark_starred",
		Description: "Star or unstar a grocery item.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in foodlistMarkStarredIn) (*mcp.CallToolResult, any, error) {
		_ = ctx
		_ = req
		var cmd Command
		if in.Starred {
			cmd = StarTodoCommand{
				BaseCommand: BaseCommand{Type: "StarTodo", CommandID: uuid.NewString()},
				ID:          in.TodoID,
			}
		} else {
			cmd = UnstarTodoCommand{
				BaseCommand: BaseCommand{Type: "UnstarTodo", CommandID: uuid.NewString()},
				ID:          in.TodoID,
			}
		}
		if err := app.ExecuteCommand(cmd); err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
				IsError: true,
			}, nil, nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Grocery item %s starred=%v.", in.TodoID, in.Starred)}},
		}, nil, nil
	})

	writeResourceJSON := func(uri string, v any) (*mcp.ReadResourceResult, error) {
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return nil, err
		}
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{{
				URI:      uri,
				MIMEType: "application/json",
				Text:     string(b),
			}},
		}, nil
	}

	s.AddResource(&mcp.Resource{
		URI:         mcpResourceState,
		Name:        "state",
		Description: "Full projected state (StateRollup JSON): grocery items, categories, list title.",
		MIMEType:    "application/json",
	}, func(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if req.Params.URI != mcpResourceState {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}
		rollup := StateRollup{
			Type:       "StateRollup",
			Todos:      app.state.GetTodos(),
			Categories: app.state.GetCategories(),
			ListTitle:  app.state.GetListTitle(),
			Version:    version,
		}
		return writeResourceJSON(mcpResourceState, rollup)
	})

	s.AddResource(&mcp.Resource{
		URI:         mcpResourceCategories,
		Name:        "categories",
		Description: "All categories as JSON array.",
		MIMEType:    "application/json",
	}, func(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if req.Params.URI != mcpResourceCategories {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}
		return writeResourceJSON(mcpResourceCategories, app.state.GetCategories())
	})

	s.AddResource(&mcp.Resource{
		URI:         mcpResourceTodos,
		Name:        "grocery_items",
		Description: "All grocery items sorted by sort order (JSON array). Legacy URI remains foodlist://todos for compatibility.",
		MIMEType:    "application/json",
	}, func(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if req.Params.URI != mcpResourceTodos {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}
		return writeResourceJSON(mcpResourceTodos, app.state.GetTodos())
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "foodlist_suggestions",
		Description: "List grocery items the user probably wants to buy soon (frequently purchased, currently not in the shopping list, and due based on the typical interval). Empty when the suggestion engine is disabled.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in struct{}) (*mcp.CallToolResult, any, error) {
		_ = ctx
		_ = req
		_ = in
		var b strings.Builder
		_, _ = fmt.Fprintf(&b, "**Suggestions for %s**\n\n", app.state.GetListTitle())
		if !app.SuggestionsEnabled() {
			b.WriteString("_Suggestion engine is disabled (requires embeddings)._\n")
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: b.String()}},
			}, nil, nil
		}
		sugs := app.suggestions.Snapshot()
		if len(sugs) == 0 {
			b.WriteString("_No suggestions right now._\n")
		}
		now := time.Now().UTC()
		for _, sg := range sugs {
			catLabel := "(uncategorized)"
			if sg.CategoryName != nil && *sg.CategoryName != "" {
				catLabel = *sg.CategoryName
			} else if sg.CategoryID != nil {
				catLabel = *sg.CategoryID
			}
			sinceLast := now.Sub(sg.LastPurchasedAt).Round(time.Hour)
			intervalDays := sg.AvgIntervalSeconds / 86400
			_, _ = fmt.Fprintf(
				&b,
				"- **%s** `%s` — %s (bought %d times, last %s ago, typical interval ~%.1f days)\n",
				sg.Name, sg.ID, catLabel, sg.PurchaseCount, sinceLast, intervalDays,
			)
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: b.String()}},
		}, nil, nil
	})

	s.AddResource(&mcp.Resource{
		URI:         mcpResourceSuggestions,
		Name:        "suggestions",
		Description: "Current grocery suggestions as a JSON array. Empty when the suggestion engine is disabled.",
		MIMEType:    "application/json",
	}, func(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if req.Params.URI != mcpResourceSuggestions {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}
		var sugs []Suggestion
		if app.SuggestionsEnabled() {
			sugs = app.suggestions.Snapshot()
		} else {
			sugs = []Suggestion{}
		}
		return writeResourceJSON(mcpResourceSuggestions, sugs)
	})

	return s
}

// foodlistMCPHandler serves MCP over streamable HTTP at /mcp (no application-level auth; protect at network/reverse-proxy if needed).
func foodlistMCPHandler(app *Server) http.Handler {
	mcpSrv := newFoodlistMCPServer(app)
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return mcpSrv
	}, &mcp.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
	})
}
