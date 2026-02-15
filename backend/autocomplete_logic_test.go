package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAutocompleteLogic_Priority(t *testing.T) {
	categories := map[string]*Category{
		"cat-1": {ID: "cat-1", Name: "Bageri"},
		"cat-2": {ID: "cat-2", Name: "Mejeri"},
	}
	al := NewAutocompleteLogic(categories)

	// --- Test Data ---
	// Bröd: High frequency, has category, has emoji variant
	al.addAutocompleteEntry("Bröd", &categories["cat-1"].ID)
	al.addAutocompleteEntry("Bröd", &categories["cat-1"].ID)
	al.addAutocompleteEntry("Bröd 🍞", &categories["cat-1"].ID)
	// Ost: Medium frequency, has category, has emoji variant
	al.addAutocompleteEntry("Ost", &categories["cat-2"].ID)
	al.addAutocompleteEntry("Ost 🧀", &categories["cat-2"].ID)
	// Oliver: Low frequency, no category, has emoji
	al.addAutocompleteEntry("Oliver 🫒", nil)
	// Mjölk: Low frequency, no category, no emoji
	al.addAutocompleteEntry("Mjölk", nil)

	t.Run("Prioritizes items with categories, emojis, and frequency correctly", func(t *testing.T) {
		// --- Test Case 1: Empty Query ---
		// Expected order:
		// 1. Bröd 🍞 (Category + Emoji + High Frequency)
		// 2. Ost 🧀 (Category + Emoji + Medium Frequency)
		// 3. Oliver 🫒 (Emoji)
		// 4. Mjölk (Base)
		suggestions := al.GetSuggestions("", []string{})
		assert.Len(t, suggestions, 4, "Should return 4 suggestions for empty query")

		suggestionNames := make([]string, len(suggestions))
		for i, s := range suggestions {
			suggestionNames[i] = s.Name
		}
		expectedOrder := []string{"Bröd 🍞", "Ost 🧀", "Oliver 🫒", "Mjölk"}
		assert.Equal(t, expectedOrder, suggestionNames, "Suggestions for empty query are not in the correct priority order")

		// Verify category for the top item
		assert.NotNil(t, suggestions[0].CategoryName, "Highest priority for 'Bröd' should have a category")
		assert.Equal(t, "Bageri", *suggestions[0].CategoryName, "Highest priority for 'Bröd' should have the 'Bageri' category")

		// --- Test Case 2: Query 'br' ---
		// Expected: "Bröd 🍞" should be the top hit due to prefix match and high score
		suggestions = al.GetSuggestions("br", []string{})
		assert.NotEmpty(t, suggestions, "Should get suggestions for query 'br'")
		assert.Equal(t, "Bröd 🍞", suggestions[0].Name, "Prefix match 'br' should prioritize 'Bröd 🍞'")
		assert.NotNil(t, suggestions[0].CategoryName, "Category should be present for 'Bröd 🍞'")
		assert.Equal(t, "Bageri", *suggestions[0].CategoryName)

		// --- Test Case 3: Active items should be filtered out ---
		suggestions = al.GetSuggestions("", []string{"Bröd", "Ost"})
		assert.Len(t, suggestions, 2, "Should filter out active items")
		assert.Equal(t, "Oliver 🫒", suggestions[0].Name)
		assert.Equal(t, "Mjölk", suggestions[1].Name)
	})
}

func TestAutocompleteLogic_CategoryConsolidation(t *testing.T) {
	categories := map[string]*Category{
		"cat-1": {ID: "cat-1", Name: "Bageri"},
		"cat-2": {ID: "cat-2", Name: "Annan"},
	}
	al := NewAutocompleteLogic(categories)

	// --- Test Data ---
	// Add "Bröd" multiple times, once without a category, and twice with "Bageri"
	al.addAutocompleteEntry("Bröd", nil)
	al.addAutocompleteEntry("Bröd", &categories["cat-1"].ID)
	al.addAutocompleteEntry("Bröd", &categories["cat-1"].ID)
	al.addAutocompleteEntry("Bröd 🍞", &categories["cat-1"].ID) // With emoji

	// Add another item with a less frequent category
	al.addAutocompleteEntry("Bröd", &categories["cat-2"].ID)

	t.Run("Consolidates item with most frequent category", func(t *testing.T) {
		suggestions := al.GetSuggestions("", []string{})
		assert.Len(t, suggestions, 1, "Should return 1 consolidated suggestion")

		suggestion := suggestions[0]
		assert.Equal(t, "Bröd 🍞", suggestion.Name, "Suggestion should have emoji")
		assert.NotNil(t, suggestion.CategoryName, "Suggestion should have a category")
		assert.Equal(t, "Bageri", *suggestion.CategoryName, "Suggestion should have the most frequent category")
	})
}
