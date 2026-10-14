package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
)

// MCP's cache hints are distinct from HTTP Cache-Control. Verify both layers.
func TestMCP_HTTPPrivateNoStore(t *testing.T) {
	srv, cleanup := newServerWithTempStore(t)
	defer cleanup()
	ts := httptest.NewServer(foodlistMCPHandler(srv))
	defer ts.Close()

	cases := []struct {
		name, method, body string
		status             int
	}{
		{
			name:   "successful_initialize",
			method: http.MethodPost,
			body:   `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"cache-test","version":"1"}}}`,
			status: http.StatusOK,
		},
		{
			name:   "error_response",
			method: http.MethodGet,
			status: http.StatusMethodNotAllowed,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, ts.URL, strings.NewReader(tc.body))
			require.NoError(t, err)
			req.Header.Set("Accept", "application/json, text/event-stream")
			if tc.method == http.MethodPost {
				req.Header.Set("Content-Type", "application/json")
			}
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, tc.status, resp.StatusCode)
			require.Equal(t, "private, no-store, no-transform", resp.Header.Get("Cache-Control"))
		})
	}
}

func TestMCP_20260728PrivateImmediatelyStale(t *testing.T) {
	srv, cleanup := newServerWithTempStore(t)
	defer cleanup()
	ts := httptest.NewServer(foodlistMCPHandler(srv))
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "cache-test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint: ts.URL, DisableStandaloneSSE: true,
	}, nil)
	require.NoError(t, err)
	defer session.Close()

	// The modern (2026-07-28) SDK client negotiates per-request cache hints.
	// These must be private even for SDK-produced results, which otherwise
	// default to cacheScope=public, and never considered fresh (ttlMs=0).
	tools, err := session.ListTools(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 0, tools.TTLMs)
	require.Equal(t, "private", tools.CacheScope)
	require.NotEmpty(t, tools.Tools)

	resources, err := session.ListResources(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 0, resources.TTLMs)
	require.Equal(t, "private", resources.CacheScope)

	templates, err := session.ListResourceTemplates(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, 0, templates.TTLMs)
	require.Equal(t, "private", templates.CacheScope)

	for _, uri := range []string{mcpShoppingAppURI, mcpRecipesAppURI} {
		t.Run(uri, func(t *testing.T) {
			result, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri})
			require.NoError(t, err)
			require.Equal(t, 0, result.TTLMs)
			require.Equal(t, "private", result.CacheScope)
			require.NotEmpty(t, result.Contents)
		})
	}
}
