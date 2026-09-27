package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func cacheTestRecipe(id, title string, created time.Time) Recipe {
	return Recipe{
		ID: id, Title: title, CreatedAt: created,
		Sections: []RecipeSection{{Ingredients: []Ingredient{{Name: "Salt"}}}},
	}
}

func TestRecipeStoreListSnapshotAndRestart(t *testing.T) {
	dir := t.TempDir()
	store, err := NewRecipeStore(filepath.Join(dir, "recipes"), dir, 1_000_000)
	require.NoError(t, err)
	older, err := store.Save(cacheTestRecipe(uuid.NewString(), "Older", time.Now().UTC().Add(-time.Hour)), nil, "")
	require.NoError(t, err)
	newer, err := store.Save(cacheTestRecipe(uuid.NewString(), "Newer", time.Now().UTC()), nil, "")
	require.NoError(t, err)

	list, err := store.List()
	require.NoError(t, err)
	require.Equal(t, []string{newer.ID, older.ID}, []string{list[0].ID, list[1].ID})
	list[0].Title = "caller changed title"
	list = append(list, RecipeMeta{ID: "extra"})
	require.Len(t, list, 3)
	unchanged, err := store.List()
	require.NoError(t, err)
	require.Len(t, unchanged, 2)
	require.Equal(t, "Newer", unchanged[0].Title)

	updated, err := store.Update(older.ID, func(r Recipe) (Recipe, error) {
		r.Title = "Updated"
		return r, nil
	})
	require.NoError(t, err)
	list, err = store.List()
	require.NoError(t, err)
	require.Equal(t, "Updated", list[1].Title)
	require.Equal(t, updated.UpdatedAt, list[1].UpdatedAt)
	require.NoError(t, store.Delete(newer.ID))
	list, err = store.List()
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, older.ID, list[0].ID)
	// Equal creation timestamps must keep the same order after restart,
	// including when one of the tied recipes has been updated.
	for _, id := range []string{
		"ffffffff-ffff-ffff-ffff-ffffffffffff",
		"00000000-0000-0000-0000-000000000001",
	} {
		_, err := store.Save(cacheTestRecipe(id, "Tied", older.CreatedAt), nil, "")
		require.NoError(t, err)
	}
	list, err = store.List()
	require.NoError(t, err)
	require.Len(t, list, 3)
	for i := 1; i < len(list); i++ {
		require.Less(t, list[i-1].ID, list[i].ID)
	}

	// A malformed JSON file and unrelated files must not prevent startup.
	require.NoError(t, os.WriteFile(filepath.Join(store.baseDir, uuid.NewString()+".json"), []byte("{"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(store.baseDir, "notes.json"), []byte("{}"), 0o644))
	restarted, err := NewRecipeStore(store.baseDir, dir, 1_000_000)
	require.NoError(t, err)
	rebuilt, err := restarted.List()
	require.NoError(t, err)
	require.Equal(t, list, rebuilt)

	// Once loaded, listing uses the snapshot and does not read recipe bodies.
	require.NoError(t, os.Remove(filepath.Join(store.baseDir, older.ID+".json")))
	stillCached, err := restarted.List()
	require.NoError(t, err)
	require.Equal(t, rebuilt, stillCached)
}

func TestRecipeListHTTPAndMCPReflectStoreMutations(t *testing.T) {
	srv := newServerWithRecipes(t)
	store := srv.RecipeStore()
	api := NewRecipeAPI(store, nil, srv, "", 100)
	mux := http.NewServeMux()
	api.Register(mux, func(h http.Handler) http.Handler { return h })
	base := initRecipeMCP(t, srv)

	check := func(wantTitle string, wantCount int) {
		t.Helper()
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/recipes", nil))
		require.Equal(t, http.StatusOK, rr.Code)
		var httpList recipeListResponse
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &httpList))
		require.Len(t, httpList.Recipes, wantCount)
		var mcpList []RecipeMeta
		require.NoError(t, json.Unmarshal([]byte(resourceText(t, base, 9, mcpResourceRecipes)), &mcpList))
		require.Len(t, mcpList, wantCount)
		if wantCount > 0 {
			require.Equal(t, wantTitle, httpList.Recipes[0].Title)
			require.Equal(t, wantTitle, mcpList[0].Title)
		}
	}

	check("", 0)
	saved, err := store.Save(cacheTestRecipe(uuid.NewString(), "Soup", time.Time{}), nil, "")
	require.NoError(t, err)
	check("Soup", 1)
	_, err = store.Update(saved.ID, func(r Recipe) (Recipe, error) { r.Title = "Stew"; return r, nil })
	require.NoError(t, err)
	check("Stew", 1)
	require.NoError(t, store.Delete(saved.ID))
	check("", 0)
}

func TestRecipeStoreListConcurrentReadsAndWrites(t *testing.T) {
	dir := t.TempDir()
	store, err := NewRecipeStore(filepath.Join(dir, "recipes"), dir, 1_000_000)
	require.NoError(t, err)
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for reader := 0; reader < 6; reader++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				metas, err := store.List()
				if err != nil {
					errs <- err
					return
				}
				for j, meta := range metas {
					if meta.ID == "" || meta.Title == "" || (j > 0 && meta.CreatedAt.After(metas[j-1].CreatedAt)) {
						errs <- fmt.Errorf("invalid list snapshot: %+v", metas)
						return
					}
				}
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			id := uuid.NewString()
			if _, err := store.Save(cacheTestRecipe(id, "Soup", time.Time{}), nil, ""); err != nil {
				errs <- err
				return
			}
			if _, err := store.Update(id, func(r Recipe) (Recipe, error) { r.Title = "Stew"; return r, nil }); err != nil {
				errs <- err
				return
			}
			if err := store.Delete(id); err != nil {
				errs <- err
				return
			}
		}
	}()
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
}
