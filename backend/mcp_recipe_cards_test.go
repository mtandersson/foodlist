package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMCP_RecipeCardsMetadataResourceAndThumbnail(t *testing.T) {
	srv := newServerWithRecipes(t)
	store := srv.RecipeStore()
	img := makeTestPNG(t, 400, 300)
	withImage, err := store.Save(Recipe{ID: uuid.NewString(), Title: "Cake <img src=x>", Sections: []RecipeSection{{Ingredients: []Ingredient{{Name: "Flour"}}}}}, img, "image/png")
	require.NoError(t, err)
	withoutImage, err := store.Save(Recipe{ID: uuid.NewString(), Title: "Soup", Sections: []RecipeSection{{Ingredients: []Ingredient{{Name: "Water"}}}}}, nil, "")
	require.NoError(t, err)
	base := initRecipeMCP(t, srv)

	tools := mcpResult(t, base, 2, "tools/list", map[string]any{})["tools"].([]any)
	var listTool map[string]any
	for _, entry := range tools {
		tool := entry.(map[string]any)
		if tool["name"] == "foodlist_recipes_list" {
			listTool = tool
			break
		}
	}
	require.NotNil(t, listTool)
	require.Equal(t, mcpRecipesAppURI, listTool["_meta"].(map[string]any)["ui"].(map[string]any)["resourceUri"])

	res := toolCall(t, base, 3, "foodlist_recipes_list", map[string]any{})
	require.True(t, strings.HasPrefix(firstTextContent(t, res), untrustedRecipeBanner))
	data := res["structuredContent"].(map[string]any)
	require.Equal(t, true, data["enabled"])
	metas := data["recipes"].([]any)
	require.Len(t, metas, 2)
	byID := map[string]map[string]any{}
	for _, entry := range metas {
		meta := entry.(map[string]any)
		byID[meta["id"].(string)] = meta
		require.NotEmpty(t, meta["createdAt"])
		require.NotEmpty(t, meta["updatedAt"])
	}
	require.Equal(t, withImage.Title, byID[withImage.ID]["title"])
	require.NotEmpty(t, byID[withImage.ID]["imageFilename"])
	require.Equal(t, "", byID[withoutImage.ID]["imageFilename"])
	require.NotContains(t, firstTextContent(t, res), "data:image")

	resources := mcpResult(t, base, 4, "resources/list", map[string]any{})["resources"].([]any)
	var view map[string]any
	for _, entry := range resources {
		resource := entry.(map[string]any)
		if resource["uri"] == mcpRecipesAppURI {
			view = resource
			break
		}
	}
	require.NotNil(t, view)
	require.Equal(t, "text/html;profile=mcp-app", view["mimeType"])
	contents := mcpResult(t, base, 5, "resources/read", map[string]any{"uri": mcpRecipesAppURI})["contents"].([]any)
	viewContent := contents[0].(map[string]any)
	require.Equal(t, "text/html;profile=mcp-app", viewContent["mimeType"])
	html := viewContent["text"].(string)
	require.True(t, strings.HasPrefix(html, "<!doctype html>"))
	require.Contains(t, html, "foodlist_recipe_get")
	require.Contains(t, html, "foodlist_recipe_add_ingredients")
	require.NotContains(t, html, withImage.Title)
	ui := viewContent["_meta"].(map[string]any)["ui"].(map[string]any)
	require.Empty(t, ui["csp"].(map[string]any)["connectDomains"])

	templates := mcpResult(t, base, 6, "resources/templates/list", map[string]any{})["resourceTemplates"].([]any)
	found := false
	for _, entry := range templates {
		if entry.(map[string]any)["uriTemplate"] == mcpRecipeThumbTemplate {
			found = true
		}
	}
	require.True(t, found)
	thumbURI := mcpRecipeThumbPrefix + withImage.ID
	thumbContents := mcpResult(t, base, 7, "resources/read", map[string]any{"uri": thumbURI})["contents"].([]any)
	thumb := thumbContents[0].(map[string]any)
	require.Equal(t, "image/jpeg", thumb["mimeType"])
	encoded := thumb["blob"].(string)
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	require.NoError(t, err)
	config, format, err := image.DecodeConfig(bytes.NewReader(decoded))
	require.NoError(t, err)
	require.Equal(t, "jpeg", format)
	require.LessOrEqual(t, config.Width, 256)
	require.LessOrEqual(t, config.Height, 256)
	missingImage, err := mcpEnvelope(t, base, 8, "resources/read", map[string]any{"uri": mcpRecipeThumbPrefix + withoutImage.ID})
	require.NoError(t, err)
	require.NotNil(t, missingImage["error"])
}

func TestMCP_RecipeCardsDisabled(t *testing.T) {
	srv, cleanup := newServerWithTempStore(t)
	defer cleanup()
	base := initRecipeMCP(t, srv)
	res := toolCall(t, base, 2, "foodlist_recipes_list", map[string]any{})
	require.Contains(t, firstTextContent(t, res), "disabled")
	data := res["structuredContent"].(map[string]any)
	require.Equal(t, false, data["enabled"])
	require.Empty(t, data["recipes"])
	contents := mcpResult(t, base, 3, "resources/read", map[string]any{"uri": mcpRecipesAppURI})["contents"].([]any)
	require.Equal(t, "text/html;profile=mcp-app", contents[0].(map[string]any)["mimeType"])
	image, err := mcpEnvelope(t, base, 4, "resources/read", map[string]any{"uri": mcpRecipeThumbPrefix + uuid.NewString()})
	require.NoError(t, err)
	require.NotNil(t, image["error"])
}
