package main

import (
	_ "embed"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed mcp_app_dist/shopping.html
var shoppingAppHTML string

func shoppingAppMeta() mcp.Meta {
	return mcp.Meta{"ui": map[string]any{
		"csp": map[string]any{
			"connectDomains":  []string{},
			"resourceDomains": []string{},
			"frameDomains":    []string{},
		},
	}}
}

func shoppingAppResource() *mcp.ReadResourceResult {
	return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{
		URI: mcpShoppingAppURI, MIMEType: "text/html;profile=mcp-app",
		Text: shoppingAppHTML, Meta: shoppingAppMeta(),
	}}}
}
