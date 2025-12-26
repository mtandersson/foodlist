// Package main implements the FoodList backend server with WebSocket support and event sourcing.
package main

import (
	"sort"
	"strings"
	"unicode"
)

// levenshteinDistance calculates the Levenshtein edit distance between two strings
// using the Wagner-Fischer algorithm with O(min(m,n)) space complexity
func levenshteinDistance(s1, s2 string) int {
	// Convert to lowercase for case-insensitive comparison
	s1 = strings.ToLower(s1)
	s2 = strings.ToLower(s2)

	// Ensure s1 is the shorter string to minimize space usage
	if len(s1) > len(s2) {
		s1, s2 = s2, s1
	}

	m, n := len(s1), len(s2)
	if m == 0 {
		return n
	}
	if n == 0 {
		return m
	}

	// Use two rows instead of full matrix
	prevRow := make([]int, m+1)
	currRow := make([]int, m+1)

	// Initialize first row
	for j := 0; j <= m; j++ {
		prevRow[j] = j
	}

	for i := 1; i <= n; i++ {
		currRow[0] = i
		for j := 1; j <= m; j++ {
			cost := 1
			if s2[i-1] == s1[j-1] {
				cost = 0
			}
			currRow[j] = min(
				prevRow[j]+1,      // deletion
				currRow[j-1]+1,    // insertion
				prevRow[j-1]+cost, // substitution
			)
		}
		prevRow, currRow = currRow, prevRow
	}

	return prevRow[m]
}

// suggestionCandidate holds a suggestion with its ranking score
type suggestionCandidate struct {
	name         string
	frequency    int
	distance     int
	score        float64
	categoryID   *string
	categoryName *string
}

// containsEmoji checks if a string contains any emoji characters
// Emojis are more fun, so we prefer items with emojis in autocomplete
func containsEmoji(s string) bool {
	for _, r := range s {
		// Check for common emoji ranges
		// Emoticons, Dingbats, Symbols, etc.
		if r >= 0x1F300 && r <= 0x1F9FF { // Miscellaneous Symbols and Pictographs, Emoticons, etc.
			return true
		}
		if r >= 0x2600 && r <= 0x26FF { // Miscellaneous Symbols
			return true
		}
		if r >= 0x2700 && r <= 0x27BF { // Dingbats
			return true
		}
		if r >= 0x1F600 && r <= 0x1F64F { // Emoticons
			return true
		}
		if r >= 0x1F680 && r <= 0x1F6FF { // Transport and Map Symbols
			return true
		}
		if r >= 0x1F1E0 && r <= 0x1F1FF { // Flags
			return true
		}
		// Also check if it's a symbol that's not a standard letter/number/punctuation
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) && !unicode.IsPunct(r) && !unicode.IsSpace(r) && r > 127 {
			return true
		}
	}
	return false
}

// getAutocompleteSuggestions returns up to 4 autocomplete suggestions based on query
// It uses fuzzy matching with Levenshtein distance and ranks by frequency + recency of category
func (s *Server) getAutocompleteSuggestions(query string) []AutocompleteSuggestion {
	// Get all todo names from history with their frequencies
	nameFrequency := s.state.GetNameFrequency()

	// Get active todo names to filter out (case-insensitive)
	activeTodos := s.state.GetActiveTodoNames()
	activeSet := make(map[string]bool)
	for _, name := range activeTodos {
		activeSet[strings.ToLower(name)] = true
	}

	queryLower := strings.ToLower(query)
	var candidates []suggestionCandidate

	for name, freq := range nameFrequency {
		nameLower := strings.ToLower(name)

		// Skip if this name is already in the active todo list
		if activeSet[nameLower] {
			continue
		}

		var distance int
		var matchScore float64

		if query == "" {
			// Empty query: match all, score based on frequency only
			distance = 0
			matchScore = float64(freq) * 1000
		} else {
			// Check for prefix match first (higher priority)
			switch {
			case strings.HasPrefix(nameLower, queryLower):
				distance = 0
				matchScore = float64(freq)*1000 + 500 // Bonus for prefix match
			case strings.Contains(nameLower, queryLower):
				// Substring match
				distance = 0
				matchScore = float64(freq)*1000 + 250 // Bonus for substring match
			default:
				// Calculate Levenshtein distance
				distance = levenshteinDistance(query, name)

				// Only include if distance <= 3
				if distance > 3 {
					continue
				}

				matchScore = float64(freq)*1000 - float64(distance)*100
			}
		}

		// Attach last known category (recency-based suggestion)
		var categoryID *string
		var categoryName *string
		if lastCat := s.state.GetLastCategoryForName(name); lastCat != nil {
			categoryID = lastCat
			if cat, ok := s.state.GetCategory(*lastCat); ok {
				// Store a copy of the string value to avoid dangling pointer
				nameCopy := cat.Name
				categoryName = &nameCopy
				// Prefer categorized suggestions slightly
				matchScore += 200
			}
		}

		// Bonus for items with emojis - they're more fun! 🎉
		if containsEmoji(name) {
			matchScore += 300
		}

		candidates = append(candidates, suggestionCandidate{
			name:         name,
			frequency:    freq,
			distance:     distance,
			score:        matchScore,
			categoryID:   categoryID,
			categoryName: categoryName,
		})
	}

	// Sort by score (descending)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	// Take top 4
	result := make([]AutocompleteSuggestion, 0, 4)
	for i := 0; i < len(candidates) && i < 4; i++ {
		result = append(result, AutocompleteSuggestion{
			Name:         candidates[i].name,
			CategoryID:   candidates[i].categoryID,
			CategoryName: candidates[i].categoryName,
		})
	}

	return result
}
