package main

import (
	"sort"
	"strings"
)

// AutocompleteLogic manages the state and logic for autocomplete suggestions.
type AutocompleteLogic struct {
	// autocompleteEntries stores the core data for generating suggestions.
	// The key is the normalized (lowercase, emoji-stripped) item name.
	autocompleteEntries map[string]*AutocompleteEntry
	categories          map[string]*Category // A reference to the main state's categories
}

// AutocompleteVariant stores a specific variation of a todo item name.
type AutocompleteVariant struct {
	Name       string
	CategoryID *string
}

// AutocompleteEntry groups all variants for a normalized name.
type AutocompleteEntry struct {
	Frequency                int
	Variants                 []AutocompleteVariant
	OriginalName             string  // The first name seen for this entry
	MostFrequentCategoryID   *string // The category ID that appears most often
	MostFrequentCategoryName *string // The name of the most frequent category
}

// suggestionCandidate holds a suggestion with its ranking score during processing.
type suggestionCandidate struct {
	name         string
	frequency    int
	distance     int
	score        float64
	categoryID   *string
	categoryName *string
}

// NewAutocompleteLogic creates a new instance of AutocompleteLogic.
func NewAutocompleteLogic(categories map[string]*Category) *AutocompleteLogic {
	return &AutocompleteLogic{
		autocompleteEntries: make(map[string]*AutocompleteEntry),
		categories:          categories,
	}
}

func (al *AutocompleteLogic) Reset() {
	al.autocompleteEntries = make(map[string]*AutocompleteEntry)
}

// Apply processes an event to update the autocomplete state.
func (al *AutocompleteLogic) Apply(event Event, todo *Todo) {
	switch e := event.(type) {
	case TodoCreated:
		al.addAutocompleteEntry(e.Name, e.CategoryID)
	case TodoRenamed:
		// A more robust solution would involve storing the original name in the event.
		// For now, we just add the new one, which is what the previous logic did.
		al.addAutocompleteEntry(e.Name, todo.CategoryID)
	case TodoCategorized:
		if todo != nil {
			al.updateAutocompleteCategory(todo.Name, e.CategoryID)
		}
	}
}

// addAutocompleteEntry adds a new item variant to our autocomplete data.
func (al *AutocompleteLogic) addAutocompleteEntry(name string, categoryID *string) {
	normalized := normalizeName(name)
	if normalized == "" {
		return
	}

	entry, exists := al.autocompleteEntries[normalized]
	if !exists {
		entry = &AutocompleteEntry{
			OriginalName: name, // Store the first-seen name
		}
		al.autocompleteEntries[normalized] = entry
	}

	entry.Frequency++

	// Check if this exact variant already exists
	variantExists := false
	for _, v := range entry.Variants {
		if v.Name == name {
			variantExists = true
			break
		}
	}
	if !variantExists {
		entry.Variants = append(entry.Variants, AutocompleteVariant{Name: name, CategoryID: categoryID})
	}

	al.updateMostFrequentCategory(entry)
}

// updateAutocompleteCategory updates the category for a specific variant.
func (al *AutocompleteLogic) updateAutocompleteCategory(name string, categoryID *string) {
	normalized := normalizeName(name)
	if entry, ok := al.autocompleteEntries[normalized]; ok {
		for i, v := range entry.Variants {
			if v.Name == name {
				entry.Variants[i].CategoryID = categoryID
				break
			}
		}
		al.updateMostFrequentCategory(entry)
	}
}

// updateMostFrequentCategory recalculates the most frequent category for an entry.
func (al *AutocompleteLogic) updateMostFrequentCategory(entry *AutocompleteEntry) {
	categoryCounts := make(map[string]int)
	for _, v := range entry.Variants {
		if v.CategoryID != nil {
			categoryCounts[*v.CategoryID]++
		}
	}

	var mostFrequentID *string
	maxCount := 0
	for id, count := range categoryCounts {
		if count > maxCount {
			maxCount = count
			catID := id // Create a new variable to take its address
			mostFrequentID = &catID
		}
	}

	entry.MostFrequentCategoryID = mostFrequentID
	if mostFrequentID != nil {
		if category, ok := al.categories[*mostFrequentID]; ok {
			entry.MostFrequentCategoryName = &category.Name
		} else {
			entry.MostFrequentCategoryName = nil // Category might have been deleted
		}
	} else {
		entry.MostFrequentCategoryName = nil
	}
}

// GetSuggestions returns up to 4 autocomplete suggestions based on a query.
func (al *AutocompleteLogic) GetSuggestions(query string, activeTodoNames []string) []AutocompleteSuggestion {
	activeSet := make(map[string]bool)
	for _, name := range activeTodoNames {
		activeSet[strings.ToLower(name)] = true
	}

	queryLower := strings.ToLower(query)
	var candidates []suggestionCandidate

	for normalizedName, entry := range al.autocompleteEntries {
		if activeSet[normalizedName] {
			continue
		}

		// Find the best variant within the entry
		bestVariant := entry.OriginalName // Default to original
		variantHasCategory := false
		variantHasEmoji := false

		for _, v := range entry.Variants {
			currentHasCategory := v.CategoryID != nil
			currentHasEmoji := containsEmoji(v.Name)

			// Prioritize variant with category
			if currentHasCategory && !variantHasCategory {
				bestVariant = v.Name
				variantHasCategory = true
				variantHasEmoji = currentHasEmoji
				continue
			}

			// If category status is the same, prioritize one with emoji
			if currentHasCategory == variantHasCategory {
				if currentHasEmoji && !variantHasEmoji {
					bestVariant = v.Name
					variantHasEmoji = currentHasEmoji
				}
			}
		}

		var distance int
		score := float64(entry.Frequency)

		if entry.MostFrequentCategoryID != nil {
			score += 1000 // Category bonus
		}
		if containsEmoji(bestVariant) { // Score based on the best variant
			score += 100 // Emoji bonus
		}

		if query == "" {
			distance = 0
		} else {
			if strings.HasPrefix(normalizedName, queryLower) {
				distance = 0
				score += 500 // Prefix bonus
			} else {
				distance = levenshteinDistance(query, normalizedName)
				if distance < 3 { // Only consider close matches
					score -= float64(distance * 10) // Levenshtein distance penalty
				} else {
					continue // Ignore items that are too different
				}
			}
		}

		candidates = append(candidates, suggestionCandidate{
			name:         bestVariant, // Use the best variant
			frequency:    entry.Frequency,
			distance:     distance,
			score:        score,
			categoryID:   entry.MostFrequentCategoryID,
			categoryName: entry.MostFrequentCategoryName,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	var suggestions []AutocompleteSuggestion
	seen := make(map[string]bool)
	for _, c := range candidates {
		if len(suggestions) >= 4 {
			break
		}
		// Use the normalized name of the best variant for uniqueness check
		normalizedBestVariant := normalizeName(c.name)
		if !seen[normalizedBestVariant] {
			suggestion := AutocompleteSuggestion{
				Name:         c.name,
				CategoryID:   c.categoryID,
				CategoryName: c.categoryName,
			}
			suggestions = append(suggestions, suggestion)
			seen[normalizedBestVariant] = true
		}
	}

	return suggestions
}

// Clear resets the autocomplete entries.
func (al *AutocompleteLogic) Clear() {
	al.autocompleteEntries = make(map[string]*AutocompleteEntry)
}
