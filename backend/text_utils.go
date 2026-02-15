package main

import (
	"strings"
	"unicode"
)

// levenshteinDistance calculates the Levenshtein edit distance between two strings.
func levenshteinDistance(s1, s2 string) int {
	s1 = strings.ToLower(s1)
	s2 = strings.ToLower(s2)
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
	prevRow := make([]int, m+1)
	currRow := make([]int, m+1)
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
			currRow[j] = min(prevRow[j]+1, currRow[j-1]+1, prevRow[j-1]+cost)
		}
		prevRow, currRow = currRow, prevRow
	}
	return prevRow[m]
}

// containsEmoji checks if a string contains any emoji characters.
func containsEmoji(s string) bool {
	for _, r := range s {
		if r >= 0x1F300 && r <= 0x1F9FF {
			return true
		}
		if r >= 0x2600 && r <= 0x26FF {
			return true
		}
		if r >= 0x2700 && r <= 0x27BF {
			return true
		}
		if r >= 0x1F600 && r <= 0x1F64F {
			return true
		}
		if r >= 0x1F680 && r <= 0x1F6FF {
			return true
		}
		if r >= 0x1F1E0 && r <= 0x1F1FF {
			return true
		}
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) && !unicode.IsPunct(r) && !unicode.IsSpace(r) && r > 127 {
			return true
		}
	}
	return false
}

// stripEmojis removes emoji characters from a string.
func stripEmojis(s string) string {
	var result strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) || (unicode.IsPunct(r) && r < 127) {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// normalizeName cleans a string for use as a key in the autocomplete map.
func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(stripEmojis(name)))
}

func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
	}
	if b < c {
		return b
	}
	return c
}
