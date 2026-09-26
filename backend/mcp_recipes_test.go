package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/jpeg"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func initRecipeMCP(t *testing.T, srv *Server) string {
	t.Helper()
	ts := httptest.NewServer(foodlistMCPHandler(srv))
	t.Cleanup(ts.Close)
	require.NoError(t, mcpOK(t, ts.URL, 1, "initialize", map[string]any{
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "recipe-test", "version": "1"},
	}))
	return ts.URL
}

func recipeArgs(title string) map[string]any {
	return map[string]any{
		"title": title,
		"sections": []any{
			map[string]any{
				"name": "",
				"ingredients": []any{
					map[string]any{"name": " Salt ", "amount": 2, "unit": " tsk "},
					map[string]any{"name": " ", "unit": "g"},
				},
				"instructions": []string{" Cook ", "Serve"},
			},
		},
	}
}

func TestMCP_RecipeCreateUpdate_TextOnly(t *testing.T) {
	srv := newServerWithRecipes(t)
	base := initRecipeMCP(t, srv)
	args := recipeArgs("  Soup  ")
	created := toolCall(t, base, 2, "foodlist_recipe_create", args)
	require.NotEqual(t, true, created["isError"], "%v", created)
	output := created["structuredContent"].(map[string]any)
	id := output["id"].(string)
	require.Equal(t, "Soup", output["recipe"].(map[string]any)["title"])
	require.Equal(t, false, output["has_image"])

	saved, err := srv.RecipeStore().Get(id)
	require.NoError(t, err)
	require.Empty(t, saved.ImageFilename)
	require.Empty(t, saved.ImageMIME)
	require.Len(t, saved.Sections[0].Ingredients, 1)
	require.Equal(t, "Salt", saved.Sections[0].Ingredients[0].Name)
	require.Equal(t, "tsk", saved.Sections[0].Ingredients[0].Unit)
	require.Equal(t, "Cook", saved.Sections[0].Instructions[0])
	require.NotZero(t, saved.CreatedAt)
	_, _, err = srv.RecipeStore().ReadImage(id)
	require.ErrorIs(t, err, ErrRecipeNotFound)

	get := toolCall(t, base, 3, "foodlist_recipe_get", map[string]any{"recipe_id": id})
	require.Contains(t, firstTextContent(t, get), "Soup")
	require.Contains(t, firstTextContent(t, toolCall(t, base, 4, "foodlist_recipes_list", map[string]any{})), id)
	require.Contains(t, resourceText(t, base, 5, mcpResourceRecipes), id)
	added := toolCall(t, base, 6, "foodlist_recipe_add_ingredients", map[string]any{"recipe_id": id})
	require.NotEqual(t, true, added["isError"])

	srv.CookSessions().Check(id, 1)
	updated := toolCall(t, base, 7, "foodlist_recipe_update", map[string]any{
		"recipe_id":   id,
		"description": "  Hot  ",
	})
	require.NotEqual(t, true, updated["isError"], "%v", updated)
	got, err := srv.RecipeStore().Get(id)
	require.NoError(t, err)
	require.Equal(t, "Hot", got.Description)
	require.Equal(t, saved.Title, got.Title)
	require.Equal(t, saved.Sections, got.Sections)
	require.Equal(t, saved.CreatedAt, got.CreatedAt)
	require.True(t, got.UpdatedAt.After(saved.UpdatedAt))

	updated = toolCall(t, base, 8, "foodlist_recipe_update", map[string]any{
		"recipe_id": id,
		"sections": []any{map[string]any{
			"name": "Finish", "ingredients": []any{}, "instructions": []string{" Eat "},
		}},
	})
	require.NotEqual(t, true, updated["isError"])
	require.Empty(t, srv.CookSessions().Snapshot()[id])
	got, err = srv.RecipeStore().Get(id)
	require.NoError(t, err)
	require.Equal(t, "Eat", got.Sections[0].Instructions[0])

	reloaded, err := NewRecipeStore(srv.RecipeStore().baseDir, filepath.Dir(srv.RecipeStore().baseDir), 1_000_000)
	require.NoError(t, err)
	got, err = reloaded.Get(id)
	require.NoError(t, err)
	require.Equal(t, "Hot", got.Description)
}

func TestMCP_RecipeCreate_ImageFormatsAndMIMEHint(t *testing.T) {
	srv := newServerWithRecipes(t)
	srv.RecipeStore().maxPixels = 24_000_000
	base := initRecipeMCP(t, srv)
	var jpg bytes.Buffer
	require.NoError(t, jpeg.Encode(&jpg, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil))
	heic, err := os.ReadFile(filepath.Join("testdata", "sample.heic"))
	require.NoError(t, err)
	webp, err := base64.StdEncoding.DecodeString("UklGRiIAAABXRUJQVlA4IBYAAAAwAQCdASoBAAEADsD+JaQAA3AAAAAA")
	require.NoError(t, err)
	for i, tc := range []struct {
		name, wantMIME string
		bytes          []byte
	}{
		{"PNG", "image/png", makeTestPNG(t, 2, 2)},
		{"JPEG", "image/jpeg", jpg.Bytes()},
		{"WebP", "image/webp", webp},
		{"HEIC", "image/jpeg", heic},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := recipeArgs(tc.name)
			args["image"] = map[string]any{
				"data_base64": base64.StdEncoding.EncodeToString(tc.bytes),
				"mime_type":   "image/gif", // deliberately wrong; bytes win
			}
			out := toolCall(t, base, 10+i, "foodlist_recipe_create", args)
			require.NotEqual(t, true, out["isError"], "%v", out)
			result := out["structuredContent"].(map[string]any)
			require.Equal(t, true, result["has_image"])
			id := result["id"].(string)
			stored, err := srv.RecipeStore().Get(id)
			require.NoError(t, err)
			require.Equal(t, tc.wantMIME, stored.ImageMIME)
			data, mime, err := srv.RecipeStore().ReadImage(id)
			require.NoError(t, err)
			require.Equal(t, tc.wantMIME, mime)
			require.NotEmpty(t, data)
			update := toolCall(t, base, 20+i, "foodlist_recipe_update", map[string]any{"recipe_id": id, "title": "Changed"})
			require.NotEqual(t, true, update["isError"])
			after, afterMIME, err := srv.RecipeStore().ReadImage(id)
			require.NoError(t, err)
			require.Equal(t, data, after)
			require.Equal(t, mime, afterMIME)
		})
	}
}

func TestMCP_RecipeCreate_MultiSectionAndIngredientShapes(t *testing.T) {
	srv := newServerWithRecipes(t)
	base := initRecipeMCP(t, srv)
	args := map[string]any{
		"title": "Dinner",
		"sections": []any{
			map[string]any{"name": "Main", "ingredients": []any{
				map[string]any{"name": "Rice", "amount": 2},
				map[string]any{"name": "Water", "unit": "dl"},
			}, "instructions": []string{"Boil"}},
			map[string]any{"name": "Finish", "ingredients": []any{
				map[string]any{"name": "Salt"},
			}, "instructions": []string{"Season"}},
		},
	}
	out := toolCall(t, base, 2, "foodlist_recipe_create", args)
	require.NotEqual(t, true, out["isError"])
	id := out["structuredContent"].(map[string]any)["id"].(string)
	got, err := srv.RecipeStore().Get(id)
	require.NoError(t, err)
	require.Len(t, got.Sections, 2)
	require.Equal(t, "Main", got.Sections[0].Name)
	require.Equal(t, float64(2), *got.Sections[0].Ingredients[0].Amount)
	require.Empty(t, got.Sections[0].Ingredients[0].Unit)
	require.Nil(t, got.Sections[0].Ingredients[1].Amount)
	require.Equal(t, "dl", got.Sections[0].Ingredients[1].Unit)
	require.Nil(t, got.Sections[1].Ingredients[0].Amount)
	require.Empty(t, got.Sections[1].Ingredients[0].Unit)
}

func TestMCP_RecipeWrite_InvalidInputs(t *testing.T) {
	srv := newServerWithRecipes(t)
	base := initRecipeMCP(t, srv)
	for i, args := range []map[string]any{
		recipeArgs(" "),
		{"title": "Empty", "sections": []any{}},
		{"title": "Bad image", "sections": recipeArgs("x")["sections"], "image": map[string]any{"data_base64": base64.StdEncoding.EncodeToString([]byte("not an image"))}},
		{"title": "Bad base64", "sections": recipeArgs("x")["sections"], "image": map[string]any{"data_base64": "!!!"}},
		{"title": "Large", "sections": recipeArgs("x")["sections"], "image": map[string]any{"data_base64": strings.Repeat("A", base64.StdEncoding.EncodedLen(recipeUploadMaxBytes)+1)}},
		{"title": "Decoded large", "sections": recipeArgs("x")["sections"], "image": map[string]any{"data_base64": base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0}, recipeUploadMaxBytes+1))}},
	} {
		out := toolCall(t, base, 30+i, "foodlist_recipe_create", args)
		require.Equal(t, true, out["isError"], "%v", out)
	}
	metas, err := srv.RecipeStore().List()
	require.NoError(t, err)
	require.Empty(t, metas)
	badUpdate := toolCall(t, base, 40, "foodlist_recipe_update", map[string]any{"recipe_id": "not-a-uuid", "title": "x"})
	require.Equal(t, true, badUpdate["isError"])
}

func TestMCP_RecipeCreate_RejectsExcessiveImageDimensions(t *testing.T) {
	srv := newServerWithRecipes(t)
	srv.RecipeStore().maxPixels = 3
	base := initRecipeMCP(t, srv)
	args := recipeArgs("Too many pixels")
	args["image"] = map[string]any{"data_base64": base64.StdEncoding.EncodeToString(makeTestPNG(t, 2, 2))}
	out := toolCall(t, base, 2, "foodlist_recipe_create", args)
	require.Equal(t, true, out["isError"])
	metas, err := srv.RecipeStore().List()
	require.NoError(t, err)
	require.Empty(t, metas)
}

func TestMCP_RecipeWritesBroadcastChanges(t *testing.T) {
	srv := newServerWithRecipes(t)
	base := initRecipeMCP(t, srv)
	created := toolCall(t, base, 2, "foodlist_recipe_create", recipeArgs("Soup"))
	require.NotEqual(t, true, created["isError"])
	id := created["structuredContent"].(map[string]any)["id"].(string)
	assertRecipeChanged := func() {
		t.Helper()
		select {
		case data := <-srv.broadcast:
			var event RecipeChanged
			require.NoError(t, json.Unmarshal(data, &event))
			require.Equal(t, "RecipeChanged", event.Type)
			require.Equal(t, id, event.ID)
		default:
			t.Fatal("missing RecipeChanged broadcast")
		}
	}
	assertRecipeChanged()
	updated := toolCall(t, base, 3, "foodlist_recipe_update", map[string]any{"recipe_id": id, "title": "New soup"})
	require.NotEqual(t, true, updated["isError"])
	assertRecipeChanged()
}

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
	require.Equal(t, true, toolCall(t, base, 6, "foodlist_recipe_create", recipeArgs("Soup"))["isError"])
	require.Equal(t, true, toolCall(t, base, 7, "foodlist_recipe_update", map[string]any{
		"recipe_id": uuid.NewString(), "title": "Soup",
	})["isError"])
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
