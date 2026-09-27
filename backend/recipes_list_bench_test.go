package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// BenchmarkRecipeStoreList measures list refreshes with 100 recipes whose
// bodies contain representative ingredient and instruction text.
func BenchmarkRecipeStoreList(b *testing.B) {
	dir := b.TempDir()
	store, err := NewRecipeStore(filepath.Join(dir, "recipes"), dir, 1_000_000)
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		_, err := store.Save(Recipe{
			ID: uuid.NewString(), Title: fmt.Sprintf("Recipe %03d", i),
			Sections: []RecipeSection{{
				Ingredients:  []Ingredient{{Name: strings.Repeat("ingredient ", 20)}},
				Instructions: []string{strings.Repeat("instruction ", 50)},
			}},
		}, nil, "")
		if err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metas, err := store.List()
		if err != nil || len(metas) != 100 {
			b.Fatalf("List returned %d recipes: %v", len(metas), err)
		}
	}
}
