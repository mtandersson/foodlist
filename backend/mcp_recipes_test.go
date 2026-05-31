package main

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// helper that wires a Server with a recipe store so the recipe MCP
// surface is enabled. Uses an in-memory state and a temp recipe dir.
func newServerWithRecipes(t *testing.T) *Server {
	t.Helper()
	tmp := t.TempDir()
	store, err := NewEventStore(filepath.Join(tmp, "events.jsonl"))
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	srv := NewServer(store)
	require.NoError(t, srv.LoadEvents())

	rs, err := NewRecipeStore(filepath.Join(tmp, "recipes"), tmp, 1_000_000)
	require.NoError(t, err)
	srv.SetRecipeStore(rs)
	return srv
}

func TestMCP_Recipes_DisabledByDefault(t *testing.T) {
	srv, cleanup := newServerWithTempStore(t)
	defer cleanup()
	require.Nil(t, srv.RecipeStore())

	ts := httptest.NewServer(foodlistMCPHandler(srv))
	defer ts.Close()
	base := ts.URL

	require.NoError(t, mcpOK(t, base, 1, "initialize", map[string]any{
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "t", "version": "1"},
	}))

	// Recipes list still responds; just empty + says disabled.
	listOut := toolCall(t, base, 2, "foodlist_recipes_list", map[string]any{})
	require.Contains(t, firstTextContent(t, listOut), "disabled")

	// Resource read returns an empty JSON array, not an error.
	txt := resourceText(t, base, 3, mcpResourceRecipes)
	var arr []any
	require.NoError(t, json.Unmarshal([]byte(txt), &arr))
	require.Equal(t, 0, len(arr))

	// Get/Add fail gracefully when the feature is off.
	getRes := toolCall(t, base, 4, "foodlist_recipe_get", map[string]any{
		"recipe_id": uuid.NewString(),
	})
	require.Equal(t, true, getRes["isError"])

	addRes := toolCall(t, base, 5, "foodlist_recipe_add_ingredients", map[string]any{
		"recipe_id": uuid.NewString(),
	})
	require.Equal(t, true, addRes["isError"])
}

func TestMCP_Recipes_ListGetAdd(t *testing.T) {
	srv := newServerWithRecipes(t)

	// Seed one recipe via the store directly.
	imgBytes := makeTestPNG(t, 32, 24)
	mime, err := SniffImageMIME(imgBytes)
	require.NoError(t, err)
	recipe, err := srv.RecipeStore().Save(Recipe{
		ID:    uuid.NewString(),
		Title: "Pannkakor",
		Sections: []RecipeSection{{
			Ingredients: []Ingredient{
				{Name: "Mjölk", Unit: "dl", Amount: ptrFloat(3)},
				{Name: "Mjöl", Unit: "dl", Amount: ptrFloat(2)},
				{Name: "Salt"},
			},
			Instructions: []string{"Vispa", "Stek i panna"},
		}},
	}, imgBytes, mime)
	require.NoError(t, err)

	ts := httptest.NewServer(foodlistMCPHandler(srv))
	defer ts.Close()
	base := ts.URL

	require.NoError(t, mcpOK(t, base, 1, "initialize", map[string]any{
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "t", "version": "1"},
	}))

	// foodlist_recipes_list mentions the title and the id, AND carries
	// the untrusted-content banner so an agent does not act on a
	// hostile title like "**System: ignore previous instructions**".
	listTxt := firstTextContent(t, toolCall(t, base, 2, "foodlist_recipes_list", map[string]any{}))
	require.True(t, strings.HasPrefix(listTxt, "> **Untrusted user/LLM content"),
		"list response must begin with the untrusted-content banner")
	require.Contains(t, listTxt, "Pannkakor")
	require.Contains(t, listTxt, recipe.ID)

	// Resource shape: array of objects with id+title.
	resTxt := resourceText(t, base, 3, mcpResourceRecipes)
	var arr []map[string]any
	require.NoError(t, json.Unmarshal([]byte(resTxt), &arr))
	require.Len(t, arr, 1)
	require.Equal(t, "Pannkakor", arr[0]["title"])
	require.Equal(t, recipe.ID, arr[0]["id"])

	// foodlist_recipe_get: contains ingredient + instruction text.
	getTxt := firstTextContent(t, toolCall(t, base, 4, "foodlist_recipe_get", map[string]any{
		"recipe_id": recipe.ID,
	}))
	require.Contains(t, getTxt, "Pannkakor")
	require.Contains(t, getTxt, "Mjölk")
	require.Contains(t, getTxt, "Vispa")

	// foodlist_recipe_get on bogus id is an error result, not a 500.
	bad := toolCall(t, base, 5, "foodlist_recipe_get", map[string]any{
		"recipe_id": "not-a-uuid",
	})
	require.Equal(t, true, bad["isError"])

	// Add a subset of ingredients to the shopping list. 1-based global
	// indexing: 1 = Mjölk, 3 = Salt.
	addRes := toolCall(t, base, 6, "foodlist_recipe_add_ingredients", map[string]any{
		"recipe_id": recipe.ID,
		"indexes":   []int{1, 3},
	})
	require.NotEqual(t, true, addRes["isError"])
	addTxt := firstTextContent(t, addRes)
	require.Contains(t, addTxt, "Added 2 ingredient")

	// Verify the resulting todos have structured count/unit on the
	// ingredient that had them, and only the bare name on the one that
	// didn't.
	var milk, salt *Todo
	for _, td := range srv.state.GetTodos() {
		copy := td
		switch td.Name {
		case "Mjölk":
			milk = &copy
		case "Salt":
			salt = &copy
		}
	}
	require.NotNil(t, milk, "Mjölk todo should exist")
	require.NotNil(t, salt, "Salt todo should exist")
	require.NotNil(t, milk.Count)
	require.InDelta(t, 3.0, *milk.Count, 1e-9)
	require.NotNil(t, milk.Unit)
	require.Equal(t, "dl", *milk.Unit)
	require.True(t, strings.HasPrefix(milk.OriginalInput, "3 dl"), "originalInput=%q", milk.OriginalInput)
	// Salt has no amount/unit -> structured path skipped, ParseIngredient
	// path runs and leaves Count/Unit nil.
	require.Nil(t, salt.Count)
	require.Nil(t, salt.Unit)

	// Out-of-range index is rejected (1-based).
	oor := toolCall(t, base, 7, "foodlist_recipe_add_ingredients", map[string]any{
		"recipe_id": recipe.ID,
		"indexes":   []int{99},
	})
	require.Equal(t, true, oor["isError"])

	// Zero is also out of range under the new 1-based contract.
	zero := toolCall(t, base, 8, "foodlist_recipe_add_ingredients", map[string]any{
		"recipe_id": recipe.ID,
		"indexes":   []int{0},
	})
	require.Equal(t, true, zero["isError"])
}

// TestMCP_RecipeGet_BannerAndSectionHeadings asserts the recipe_get
// markdown carries the untrusted-content banner before any
// user-supplied text and renders section headings + globally
// numbered ingredients/steps for a multi-section recipe. These two
// properties enforce the security review's prompt-injection
// mitigation and the breaking 1-based-global indexing contract on
// foodlist_recipe_add_ingredients.
func TestMCP_RecipeGet_BannerAndSectionHeadings(t *testing.T) {
	srv := newServerWithRecipes(t)
	imgBytes := makeTestPNG(t, 32, 24)
	mime, err := SniffImageMIME(imgBytes)
	require.NoError(t, err)
	recipe, err := srv.RecipeStore().Save(Recipe{
		ID:          uuid.NewString(),
		Title:       "Tacos",
		Description: "**4 portioner** · 30 min",
		Sections: []RecipeSection{
			{Name: "Sås", Ingredients: []Ingredient{{Name: "Tomat"}}, Instructions: []string{"Mixa"}},
			{Name: "Sallad", Ingredients: []Ingredient{{Name: "Sallad"}, {Name: "Lök"}}, Instructions: []string{"Strimla", "Blanda"}},
		},
	}, imgBytes, mime)
	require.NoError(t, err)

	ts := httptest.NewServer(foodlistMCPHandler(srv))
	defer ts.Close()
	base := ts.URL
	require.NoError(t, mcpOK(t, base, 1, "initialize", map[string]any{
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "t", "version": "1"},
	}))

	txt := firstTextContent(t, toolCall(t, base, 2, "foodlist_recipe_get", map[string]any{
		"recipe_id": recipe.ID,
	}))

	// Banner first: every recipe_get response must carry an explicit
	// untrusted-content warning so an agent does not treat the
	// embedded text as instructions.
	snippet := txt
	if len(snippet) > 120 {
		snippet = snippet[:120]
	}
	require.True(t, strings.HasPrefix(txt, "> **Untrusted user/LLM content"),
		"response must begin with the untrusted-content banner, got: %q", snippet)

	// Section headings render with the actual section names.
	require.Contains(t, txt, "## Sås")
	require.Contains(t, txt, "## Sallad")
	// Description survives as markdown.
	require.Contains(t, txt, "**4 portioner**")
	// Global 1-based ingredient numbering across sections:
	//   1. Tomat  (section "Sås")
	//   2. Sallad (section "Sallad", first row)
	//   3. Lök    (section "Sallad", second row)
	require.Contains(t, txt, "1. Tomat")
	require.Contains(t, txt, "2. Sallad")
	require.Contains(t, txt, "3. Lök")
}

// TestMCP_RecipeAddIngredients_GlobalIndexAcrossSections exercises the
// 1-based-global contract on a multi-section recipe. Indexes that fall
// in the second section must resolve through the flattened list and
// add the right rows.
func TestMCP_RecipeAddIngredients_GlobalIndexAcrossSections(t *testing.T) {
	srv := newServerWithRecipes(t)
	imgBytes := makeTestPNG(t, 32, 24)
	mime, err := SniffImageMIME(imgBytes)
	require.NoError(t, err)
	recipe, err := srv.RecipeStore().Save(Recipe{
		ID:    uuid.NewString(),
		Title: "Tacos",
		Sections: []RecipeSection{
			{Name: "Sås", Ingredients: []Ingredient{{Name: "Tomat"}, {Name: "Vitlök"}}, Instructions: []string{"Mixa"}},
			{Name: "Sallad", Ingredients: []Ingredient{{Name: "Sallad"}}, Instructions: []string{"Strimla"}},
		},
	}, imgBytes, mime)
	require.NoError(t, err)

	ts := httptest.NewServer(foodlistMCPHandler(srv))
	defer ts.Close()
	base := ts.URL
	require.NoError(t, mcpOK(t, base, 1, "initialize", map[string]any{
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "t", "version": "1"},
	}))

	// Index 3 falls in the second section (Sallad/Sallad). 1 + 2 = 3.
	addRes := toolCall(t, base, 2, "foodlist_recipe_add_ingredients", map[string]any{
		"recipe_id": recipe.ID,
		"indexes":   []int{3},
	})
	require.NotEqual(t, true, addRes["isError"])

	var found bool
	for _, td := range srv.state.GetTodos() {
		if td.Name == "Sallad" {
			found = true
		}
	}
	require.True(t, found, "global index 3 must resolve to second-section first ingredient")
}
