//go:build fuzzy

// Fuzzy category tests; enable with: go test -tags=fuzzy

package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test exact case-insensitive match
func TestFindBestMatchingCategory_ExactMatch(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")
	store, _ := NewEventStore(filePath)
	server := NewServer(store)

	now := time.Now()
	catID := "cat-1"

	// Create category
	store.Append(CategoryCreated{Type: "CategoryCreated", ID: catID, Name: "Grocery", CreatedAt: now, SortOrder: 1000})

	// Create and complete a todo "Tomater" (Swedish for tomatoes) with category
	store.Append(TodoCreated{Type: "TodoCreated", ID: "1", Name: "Tomater", CreatedAt: now, SortOrder: 1000, CategoryID: &catID})
	store.Append(TodoCompleted{Type: "TodoCompleted", ID: "1", CompletedAt: now})

	server.LoadEvents()

	// Test exact match with different casing
	result := server.findBestMatchingCategoryForName("tomater")
	require.NotNil(t, result)
	assert.Equal(t, catID, *result)

	// Test exact match uppercase
	result = server.findBestMatchingCategoryForName("TOMATER")
	require.NotNil(t, result)
	assert.Equal(t, catID, *result)
}

// Test fuzzy matching with small edit distance
func TestFindBestMatchingCategory_FuzzyMatch(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")
	store, _ := NewEventStore(filePath)
	server := NewServer(store)

	now := time.Now()
	catID := "cat-1"

	// Create category
	store.Append(CategoryCreated{Type: "CategoryCreated", ID: catID, Name: "Grocery", CreatedAt: now, SortOrder: 1000})

	// Create and complete a todo "Tomater" with category
	store.Append(TodoCreated{Type: "TodoCreated", ID: "1", Name: "Tomater", CreatedAt: now, SortOrder: 1000, CategoryID: &catID})
	store.Append(TodoCompleted{Type: "TodoCompleted", ID: "1", CompletedAt: now})

	server.LoadEvents()

	// Test fuzzy match "Tomater" vs "Tomate" (distance = 1)
	result := server.findBestMatchingCategoryForName("Tomate")
	require.NotNil(t, result)
	assert.Equal(t, catID, *result)

	// Test fuzzy match "Tomater" vs "Tomaters" (distance = 1)
	result = server.findBestMatchingCategoryForName("Tomaters")
	require.NotNil(t, result)
	assert.Equal(t, catID, *result)
}

// Test substring matching
func TestFindBestMatchingCategory_SubstringMatch(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")
	store, _ := NewEventStore(filePath)
	server := NewServer(store)

	now := time.Now()
	catID := "cat-1"

	// Create category
	store.Append(CategoryCreated{Type: "CategoryCreated", ID: catID, Name: "Grocery", CreatedAt: now, SortOrder: 1000})

	// Create and complete a todo "Tomater 3st" with category
	store.Append(TodoCreated{Type: "TodoCreated", ID: "1", Name: "Tomater 3st", CreatedAt: now, SortOrder: 1000, CategoryID: &catID})
	store.Append(TodoCompleted{Type: "TodoCompleted", ID: "1", CompletedAt: now})

	server.LoadEvents()

	// Test substring match "Tomater 3st" contains "Tomater"
	result := server.findBestMatchingCategoryForName("Tomater")
	require.NotNil(t, result)
	assert.Equal(t, catID, *result)
}

// Test no match when distance is too large
func TestFindBestMatchingCategory_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")
	store, _ := NewEventStore(filePath)
	server := NewServer(store)

	now := time.Now()
	catID := "cat-1"

	// Create category
	store.Append(CategoryCreated{Type: "CategoryCreated", ID: catID, Name: "Grocery", CreatedAt: now, SortOrder: 1000})

	// Create and complete a todo "Tomater" with category
	store.Append(TodoCreated{Type: "TodoCreated", ID: "1", Name: "Tomater", CreatedAt: now, SortOrder: 1000, CategoryID: &catID})
	store.Append(TodoCompleted{Type: "TodoCompleted", ID: "1", CompletedAt: now})

	server.LoadEvents()

	// Test no match with completely different name
	result := server.findBestMatchingCategoryForName("Bicycle")
	assert.Nil(t, result)
}

// Test returns nil when category has been deleted
func TestFindBestMatchingCategory_DeletedCategory(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")
	store, _ := NewEventStore(filePath)
	server := NewServer(store)

	now := time.Now()
	catID := "cat-1"

	// Create category
	store.Append(CategoryCreated{Type: "CategoryCreated", ID: catID, Name: "Grocery", CreatedAt: now, SortOrder: 1000})

	// Create and complete a todo with category
	store.Append(TodoCreated{Type: "TodoCreated", ID: "1", Name: "Tomater", CreatedAt: now, SortOrder: 1000, CategoryID: &catID})
	store.Append(TodoCompleted{Type: "TodoCompleted", ID: "1", CompletedAt: now})

	// Delete the category
	store.Append(CategoryDeleted{Type: "CategoryDeleted", ID: catID})

	server.LoadEvents()

	// Should return nil because category is deleted
	result := server.findBestMatchingCategoryForName("Tomater")
	assert.Nil(t, result)
}

// Test prefers closer match when multiple candidates exist
func TestFindBestMatchingCategory_PrefersCloserMatch(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")
	store, _ := NewEventStore(filePath)
	server := NewServer(store)

	now := time.Now()
	catID1 := "cat-1"
	catID2 := "cat-2"

	// Create categories
	store.Append(CategoryCreated{Type: "CategoryCreated", ID: catID1, Name: "Grocery", CreatedAt: now, SortOrder: 1000})
	store.Append(CategoryCreated{Type: "CategoryCreated", ID: catID2, Name: "Office", CreatedAt: now, SortOrder: 2000})

	// Create and complete todos with different categories
	store.Append(TodoCreated{Type: "TodoCreated", ID: "1", Name: "Tomater", CreatedAt: now, SortOrder: 1000, CategoryID: &catID1})
	store.Append(TodoCompleted{Type: "TodoCompleted", ID: "1", CompletedAt: now})

	store.Append(TodoCreated{Type: "TodoCreated", ID: "2", Name: "Tomatoe", CreatedAt: now, SortOrder: 2000, CategoryID: &catID2})
	store.Append(TodoCompleted{Type: "TodoCompleted", ID: "2", CompletedAt: now})

	server.LoadEvents()

	// "Tomato" is closer to "Tomatoe" (distance=1) than "Tomater" (distance=2)
	result := server.findBestMatchingCategoryForName("Tomato")
	require.NotNil(t, result)
	assert.Equal(t, catID2, *result) // Should match "Tomatoe" -> catID2
}

// Test integration: CreateTodoCommand with auto-categorization
func TestCreateTodo_AutoCategorization_ExactMatch(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")
	store, _ := NewEventStore(filePath)
	server := NewServer(store)

	now := time.Now()
	catID := "cat-1"

	// Create category
	store.Append(CategoryCreated{Type: "CategoryCreated", ID: catID, Name: "Grocery", CreatedAt: now, SortOrder: 1000})

	// Create and complete a todo "Tomater" with category
	store.Append(TodoCreated{Type: "TodoCreated", ID: "1", Name: "Tomater", CreatedAt: now, SortOrder: 1000, CategoryID: &catID})
	store.Append(TodoCompleted{Type: "TodoCompleted", ID: "1", CompletedAt: now})

	server.LoadEvents()

	// Create new todo with same name but no category specified - should auto-categorize
	cmd := CreateTodoCommand{
		BaseCommand: BaseCommand{Type: "CreateTodo", CommandID: "cmd-1"},
		ID:          "2",
		Name:        "tomater", // Case-insensitive match
		CategoryID:  nil,       // No category specified
	}

	event, err := server.commandToEvent(cmd)
	require.NoError(t, err)
	require.NotNil(t, event)

	todoCreated, ok := event.(TodoCreated)
	require.True(t, ok)
	require.NotNil(t, todoCreated.CategoryID)
	assert.Equal(t, catID, *todoCreated.CategoryID)
}

// Test integration: CreateTodoCommand with auto-categorization using fuzzy match
func TestCreateTodo_AutoCategorization_FuzzyMatch(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")
	store, _ := NewEventStore(filePath)
	server := NewServer(store)

	now := time.Now()
	catID := "cat-1"

	// Create category
	store.Append(CategoryCreated{Type: "CategoryCreated", ID: catID, Name: "Grocery", CreatedAt: now, SortOrder: 1000})

	// Create and complete todos "Tomater" with category
	store.Append(TodoCreated{Type: "TodoCreated", ID: "1", Name: "Tomater", CreatedAt: now, SortOrder: 1000, CategoryID: &catID})
	store.Append(TodoCompleted{Type: "TodoCompleted", ID: "1", CompletedAt: now})

	server.LoadEvents()

	// Create new todo with fuzzy match name "Tomater 3st" - should auto-categorize
	cmd := CreateTodoCommand{
		BaseCommand: BaseCommand{Type: "CreateTodo", CommandID: "cmd-1"},
		ID:          "2",
		Name:        "Tomater 3st", // Fuzzy match
		CategoryID:  nil,           // No category specified
	}

	event, err := server.commandToEvent(cmd)
	require.NoError(t, err)
	require.NotNil(t, event)

	todoCreated, ok := event.(TodoCreated)
	require.True(t, ok)
	require.NotNil(t, todoCreated.CategoryID)
	assert.Equal(t, catID, *todoCreated.CategoryID)
}

// Test integration: CreateTodoCommand respects explicit category
func TestCreateTodo_AutoCategorization_ExplicitCategoryTakesPrecedence(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")
	store, _ := NewEventStore(filePath)
	server := NewServer(store)

	now := time.Now()
	catID1 := "cat-1"
	catID2 := "cat-2"

	// Create categories
	store.Append(CategoryCreated{Type: "CategoryCreated", ID: catID1, Name: "Grocery", CreatedAt: now, SortOrder: 1000})
	store.Append(CategoryCreated{Type: "CategoryCreated", ID: catID2, Name: "Office", CreatedAt: now, SortOrder: 2000})

	// Create and complete a todo "Tomater" with catID1
	store.Append(TodoCreated{Type: "TodoCreated", ID: "1", Name: "Tomater", CreatedAt: now, SortOrder: 1000, CategoryID: &catID1})
	store.Append(TodoCompleted{Type: "TodoCompleted", ID: "1", CompletedAt: now})

	server.LoadEvents()

	// Create new todo with explicit category - should NOT auto-categorize
	cmd := CreateTodoCommand{
		BaseCommand: BaseCommand{Type: "CreateTodo", CommandID: "cmd-1"},
		ID:          "2",
		Name:        "Tomater",
		CategoryID:  &catID2, // Explicit category specified
	}

	event, err := server.commandToEvent(cmd)
	require.NoError(t, err)
	require.NotNil(t, event)

	todoCreated, ok := event.(TodoCreated)
	require.True(t, ok)
	require.NotNil(t, todoCreated.CategoryID)
	assert.Equal(t, catID2, *todoCreated.CategoryID) // Should use explicit category
}

// Test integration: CreateTodoCommand with no match - no category assigned
func TestCreateTodo_AutoCategorization_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "events.jsonl")
	store, _ := NewEventStore(filePath)
	server := NewServer(store)

	now := time.Now()
	catID := "cat-1"

	// Create category
	store.Append(CategoryCreated{Type: "CategoryCreated", ID: catID, Name: "Grocery", CreatedAt: now, SortOrder: 1000})

	// Create and complete a todo "Tomater" with category
	store.Append(TodoCreated{Type: "TodoCreated", ID: "1", Name: "Tomater", CreatedAt: now, SortOrder: 1000, CategoryID: &catID})
	store.Append(TodoCompleted{Type: "TodoCompleted", ID: "1", CompletedAt: now})

	server.LoadEvents()

	// Create new todo with completely different name - should NOT auto-categorize
	cmd := CreateTodoCommand{
		BaseCommand: BaseCommand{Type: "CreateTodo", CommandID: "cmd-1"},
		ID:          "2",
		Name:        "Bicycle",
		CategoryID:  nil,
	}

	event, err := server.commandToEvent(cmd)
	require.NoError(t, err)
	require.NotNil(t, event)

	todoCreated, ok := event.(TodoCreated)
	require.True(t, ok)
	assert.Nil(t, todoCreated.CategoryID) // Should have no category
}
