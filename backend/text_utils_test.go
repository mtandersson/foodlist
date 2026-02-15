package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLevenshteinDistance(t *testing.T) {
	assert.Equal(t, 3, levenshteinDistance("kitten", "sitting"))
	assert.Equal(t, 2, levenshteinDistance("book", "back"))
	assert.Equal(t, 0, levenshteinDistance("test", "test"))
	assert.Equal(t, 1, levenshteinDistance("test", "tests"))
	assert.Equal(t, 7, levenshteinDistance("", "testing"))
	assert.Equal(t, 7, levenshteinDistance("testing", ""))
	assert.Equal(t, 0, levenshteinDistance("", ""))
}

func TestContainsEmoji(t *testing.T) {
	assert.True(t, containsEmoji("Hello 👋"))
	assert.True(t, containsEmoji("Go is awesome 🚀"))
	assert.False(t, containsEmoji("Just a regular string"))
	assert.False(t, containsEmoji("12345"))
	assert.True(t, containsEmoji("🍎"))
}

func TestStripEmojis(t *testing.T) {
	assert.Equal(t, "Hello ", stripEmojis("Hello 👋"))
	assert.Equal(t, "Go is awesome ", stripEmojis("Go is awesome 🚀"))
	assert.Equal(t, "Just a regular string", stripEmojis("Just a regular string"))
	assert.Equal(t, " ", stripEmojis("🍎 🍌"))
}

func TestNormalizeName(t *testing.T) {
	assert.Equal(t, "hello", normalizeName("  Hello 👋  "))
	assert.Equal(t, "go is awesome", normalizeName("Go is awesome 🚀"))
	assert.Equal(t, "just a regular string", normalizeName("Just a regular string"))
}
