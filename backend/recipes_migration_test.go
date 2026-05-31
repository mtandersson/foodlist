package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestValidateAndNormalize_SectionsAndDescription covers every rule
// added by the sections refactor that isn't already exercised in
// TestValidateAndNormalize: section caps, summed totals, description
// length, drop-empty-section behavior including the named-but-empty
// case, and the empty-recipe rejection.
func TestValidateAndNormalize_SectionsAndDescription(t *testing.T) {
	mkSection := func(name string, ings, steps int) RecipeSection {
		s := RecipeSection{Name: name}
		for i := 0; i < ings; i++ {
			s.Ingredients = append(s.Ingredients, Ingredient{Name: "x"})
		}
		for i := 0; i < steps; i++ {
			s.Instructions = append(s.Instructions, "do")
		}
		return s
	}

	t.Run("description whitespace collapses to empty", func(t *testing.T) {
		r := Recipe{Title: "x", Description: "   \n\t ", Sections: []RecipeSection{mkSection("", 1, 1)}}
		out, err := ValidateAndNormalize(r)
		require.NoError(t, err)
		require.Equal(t, "", out.Description)
	})

	t.Run("description over cap is rejected", func(t *testing.T) {
		r := Recipe{Title: "x", Description: strings.Repeat("a", maxRecipeDescriptionLen+1), Sections: []RecipeSection{mkSection("", 1, 1)}}
		_, err := ValidateAndNormalize(r)
		require.ErrorIs(t, err, ErrRecipeInvalid)
	})

	t.Run("too many sections is rejected", func(t *testing.T) {
		secs := make([]RecipeSection, maxRecipeSections+1)
		for i := range secs {
			secs[i] = mkSection("s", 1, 1)
		}
		_, err := ValidateAndNormalize(Recipe{Title: "x", Sections: secs})
		require.ErrorIs(t, err, ErrRecipeInvalid)
	})

	t.Run("summed ingredients across sections respected", func(t *testing.T) {
		// Two sections totalling maxRecipeIngredients+1 must reject
		// even though no single section is over the limit.
		half := maxRecipeIngredients/2 + 1
		secs := []RecipeSection{mkSection("a", half, 1), mkSection("b", half, 1)}
		_, err := ValidateAndNormalize(Recipe{Title: "x", Sections: secs})
		require.ErrorIs(t, err, ErrRecipeInvalid)
	})

	t.Run("named but empty section is dropped, recipe rejected if nothing left", func(t *testing.T) {
		r := Recipe{Title: "x", Sections: []RecipeSection{{Name: "Sås"}}}
		_, err := ValidateAndNormalize(r)
		require.ErrorIs(t, err, ErrRecipeInvalid, "named-but-empty alone leaves zero valid sections")
	})

	t.Run("named empty section is dropped but a non-empty sibling survives", func(t *testing.T) {
		r := Recipe{Title: "x", Sections: []RecipeSection{
			{Name: "Sås"},
			mkSection("Huvudrätt", 1, 1),
		}}
		out, err := ValidateAndNormalize(r)
		require.NoError(t, err)
		require.Len(t, out.Sections, 1)
		require.Equal(t, "Huvudrätt", out.Sections[0].Name)
	})

	t.Run("recipe with no content is rejected", func(t *testing.T) {
		_, err := ValidateAndNormalize(Recipe{Title: "x"})
		require.ErrorIs(t, err, ErrRecipeInvalid)
	})

	t.Run("section name over cap is rejected", func(t *testing.T) {
		r := Recipe{Title: "x", Sections: []RecipeSection{{
			Name:         strings.Repeat("n", maxRecipeStringLen+1),
			Instructions: []string{"do"},
		}}}
		_, err := ValidateAndNormalize(r)
		require.ErrorIs(t, err, ErrRecipeInvalid)
	})
}

// TestRecipeAPI_Update_DescriptionOnly verifies PATCH with only the
// description does not wipe sections (nil vs empty pointer discipline).
// Failure mode would be: client sends {"description":"x"} and the
// server overwrites the recipe with an empty sections array.
func TestRecipeAPI_Update_DescriptionOnly(t *testing.T) {
	_, mux, store := newTestRecipeAPI(t)
	imgBytes := makeTestPNG(t, 16, 16)
	id := uuid.NewString()
	_, err := store.Save(Recipe{
		ID:    id,
		Title: "x",
		Sections: []RecipeSection{{
			Ingredients:  []Ingredient{{Name: "Salt"}},
			Instructions: []string{"A", "B"},
		}},
	}, imgBytes, "image/png")
	require.NoError(t, err)

	body, _ := json.Marshal(map[string]any{"description": "Updated text"})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/recipes/"+id, strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	got, err := store.Get(id)
	require.NoError(t, err)
	require.Equal(t, "Updated text", got.Description)
	require.Len(t, got.Sections, 1, "description-only PATCH must not wipe sections")
	require.Len(t, got.Sections[0].Instructions, 2)
}

// TestRecipeAPI_Update_PrunesCookSteps_MultiSection verifies the cook
// session pruning math uses the summed step count across sections, not
// the length of any single section. After shrinking the total below a
// previously-checked global index, that index must be dropped from the
// session.
func TestRecipeAPI_Update_PrunesCookSteps_MultiSection(t *testing.T) {
	tmp := t.TempDir()
	store, err := NewRecipeStore(filepath.Join(tmp, "recipes"), tmp, 1_000_000)
	require.NoError(t, err)

	// Wire a minimal Server so RecipeAPI's update() finds CookSessions
	// via Server.CookSessions(). NewServer requires an EventStore even
	// though we don't exercise the event log here.
	es, err := NewEventStore(filepath.Join(tmp, "events.jsonl"))
	require.NoError(t, err)
	server := NewServer(es)
	server.SetRecipeStore(store)
	sessions := server.CookSessions()

	api := NewRecipeAPI(store, nil, server, "", 100)
	mux := http.NewServeMux()
	api.Register(mux, func(h http.Handler) http.Handler { return h })

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

	// Total steps = 4. Check indices 0 and 3 (last step in second section).
	sessions.Check(id, 0)
	sessions.Check(id, 3)

	// Shrink: PATCH with two single-step sections (total=2). Index 3
	// should be pruned, 0 should remain.
	body, _ := json.Marshal(map[string]any{
		"sections": []map[string]any{
			{"ingredients": []map[string]any{{"name": "A"}}, "instructions": []string{"a1"}},
			{"ingredients": []map[string]any{{"name": "B"}}, "instructions": []string{"b1"}},
		},
	})
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/recipes/"+id, strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	snap := sessions.Snapshot()
	require.Equal(t, []int{0}, snap[id], "step index 3 must be pruned after total shrunk to 2")
}

// TestRawToRecipe maps the LLM raw payload into a Recipe ready for
// ValidateAndNormalize. Only the sectioned shape is accepted; the
// legacy top-level shape was removed when we ripped out migration code.
func TestRawToRecipe(t *testing.T) {
	t.Run("sections pass through", func(t *testing.T) {
		raw := rawRecipeResponse{
			Title: "X",
			Sections: []rawRecipeSection{
				{Name: "Sauce", Ingredients: []rawIngredient{{Name: "Tomato"}}, Instructions: []string{"Blend"}},
			},
		}
		got := rawToRecipe(raw)
		require.Len(t, got.Sections, 1)
		require.Equal(t, "Sauce", got.Sections[0].Name)
		require.Equal(t, "Tomato", got.Sections[0].Ingredients[0].Name)
	})

	t.Run("description preserved", func(t *testing.T) {
		raw := rawRecipeResponse{Description: "**Hi**", Sections: []rawRecipeSection{{Instructions: []string{"do"}}}}
		got := rawToRecipe(raw)
		require.Equal(t, "**Hi**", got.Description)
	})

	t.Run("empty sections array yields empty slice", func(t *testing.T) {
		got := rawToRecipe(rawRecipeResponse{Title: "X"})
		require.Empty(t, got.Sections)
	})
}
