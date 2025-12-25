package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestState_ApplyTodoCreated(t *testing.T) {
	state := NewState()
	now := time.Now().UTC()

	event := TodoCreated{
		Type:      "TodoCreated",
		ID:        "todo-1",
		Name:      "Buy milk",
		CreatedAt: now,
		SortOrder: 1000,
	}

	state.Apply(event)

	todos := state.GetTodos()
	require.Len(t, todos, 1)
	assert.Equal(t, "todo-1", todos[0].ID)
	assert.Equal(t, "Buy milk", todos[0].Name)
	assert.Equal(t, 1000, todos[0].SortOrder)
	assert.False(t, todos[0].Starred)
	assert.Nil(t, todos[0].CompletedAt)
}

func TestState_ApplyTodoCompleted(t *testing.T) {
	state := NewState()
	now := time.Now().UTC()

	// Create todo first
	state.Apply(TodoCreated{
		Type:      "TodoCreated",
		ID:        "todo-1",
		Name:      "Buy milk",
		CreatedAt: now,
		SortOrder: 1000,
	})

	// Complete it
	completedAt := now.Add(time.Hour)
	state.Apply(TodoCompleted{
		Type:        "TodoCompleted",
		ID:          "todo-1",
		CompletedAt: completedAt,
	})

	todos := state.GetTodos()
	require.Len(t, todos, 1)
	require.NotNil(t, todos[0].CompletedAt)
	assert.Equal(t, completedAt, *todos[0].CompletedAt)
}

func TestState_ApplyTodoUncompleted(t *testing.T) {
	state := NewState()
	now := time.Now().UTC()

	// Create and complete todo
	state.Apply(TodoCreated{
		Type:      "TodoCreated",
		ID:        "todo-1",
		Name:      "Buy milk",
		CreatedAt: now,
		SortOrder: 1000,
	})
	state.Apply(TodoCompleted{
		Type:        "TodoCompleted",
		ID:          "todo-1",
		CompletedAt: now.Add(time.Hour),
	})

	// Uncomplete it
	state.Apply(TodoUncompleted{
		Type: "TodoUncompleted",
		ID:   "todo-1",
	})

	todos := state.GetTodos()
	require.Len(t, todos, 1)
	assert.Nil(t, todos[0].CompletedAt)
}

func TestState_ApplyTodoStarred(t *testing.T) {
	state := NewState()
	now := time.Now().UTC()

	// Create two todos
	state.Apply(TodoCreated{
		Type:      "TodoCreated",
		ID:        "todo-1",
		Name:      "Task 1",
		CreatedAt: now,
		SortOrder: 1000,
	})
	state.Apply(TodoCreated{
		Type:      "TodoCreated",
		ID:        "todo-2",
		Name:      "Task 2",
		CreatedAt: now,
		SortOrder: 2000,
	})

	// Star the first one (should move to top with new sortOrder)
	state.Apply(TodoStarred{
		Type:      "TodoStarred",
		ID:        "todo-1",
		SortOrder: 3000,
	})

	todos := state.GetTodos()
	require.Len(t, todos, 2)

	// Find todo-1
	var todo1 *Todo
	for i := range todos {
		if todos[i].ID == "todo-1" {
			todo1 = &todos[i]
			break
		}
	}
	require.NotNil(t, todo1)
	assert.True(t, todo1.Starred)
	assert.Equal(t, 3000, todo1.SortOrder)
}

func TestState_ApplyTodoUnstarred(t *testing.T) {
	state := NewState()
	now := time.Now().UTC()

	// Create and star todo
	state.Apply(TodoCreated{
		Type:      "TodoCreated",
		ID:        "todo-1",
		Name:      "Task 1",
		CreatedAt: now,
		SortOrder: 1000,
	})
	state.Apply(TodoStarred{
		Type:      "TodoStarred",
		ID:        "todo-1",
		SortOrder: 5000,
	})

	// Unstar it
	state.Apply(TodoUnstarred{
		Type: "TodoUnstarred",
		ID:   "todo-1",
	})

	todos := state.GetTodos()
	require.Len(t, todos, 1)
	assert.False(t, todos[0].Starred)
	// SortOrder remains unchanged after unstar
	assert.Equal(t, 5000, todos[0].SortOrder)
}

func TestState_ApplyTodoReordered(t *testing.T) {
	state := NewState()
	now := time.Now().UTC()

	state.Apply(TodoCreated{
		Type:      "TodoCreated",
		ID:        "todo-1",
		Name:      "Task 1",
		CreatedAt: now,
		SortOrder: 1000,
	})

	state.Apply(TodoReordered{
		Type:      "TodoReordered",
		ID:        "todo-1",
		SortOrder: 5000,
	})

	todos := state.GetTodos()
	require.Len(t, todos, 1)
	assert.Equal(t, 5000, todos[0].SortOrder)
}

func TestState_ApplyTodoRenamed(t *testing.T) {
	state := NewState()
	now := time.Now().UTC()

	state.Apply(TodoCreated{
		Type:      "TodoCreated",
		ID:        "todo-1",
		Name:      "Original name",
		CreatedAt: now,
		SortOrder: 1000,
	})

	state.Apply(TodoRenamed{
		Type: "TodoRenamed",
		ID:   "todo-1",
		Name: "New name",
	})

	todos := state.GetTodos()
	require.Len(t, todos, 1)
	assert.Equal(t, "New name", todos[0].Name)
}

func TestState_GetTodosSortedBySortOrder(t *testing.T) {
	state := NewState()
	now := time.Now().UTC()

	// Create todos in random order
	state.Apply(TodoCreated{
		Type:      "TodoCreated",
		ID:        "todo-1",
		Name:      "Task 1",
		CreatedAt: now,
		SortOrder: 1000,
	})
	state.Apply(TodoCreated{
		Type:      "TodoCreated",
		ID:        "todo-3",
		Name:      "Task 3",
		CreatedAt: now,
		SortOrder: 3000,
	})
	state.Apply(TodoCreated{
		Type:      "TodoCreated",
		ID:        "todo-2",
		Name:      "Task 2",
		CreatedAt: now,
		SortOrder: 2000,
	})

	todos := state.GetTodos()
	require.Len(t, todos, 3)

	// Should be sorted by sortOrder descending (highest first)
	assert.Equal(t, "todo-3", todos[0].ID)
	assert.Equal(t, "todo-2", todos[1].ID)
	assert.Equal(t, "todo-1", todos[2].ID)
}

func TestState_GetHighestSortOrder(t *testing.T) {
	state := NewState()

	// Empty state should return 0
	assert.Equal(t, 0, state.GetHighestSortOrder())

	now := time.Now().UTC()
	state.Apply(TodoCreated{
		Type:      "TodoCreated",
		ID:        "todo-1",
		Name:      "Task 1",
		CreatedAt: now,
		SortOrder: 1000,
	})
	assert.Equal(t, 1000, state.GetHighestSortOrder())

	state.Apply(TodoCreated{
		Type:      "TodoCreated",
		ID:        "todo-2",
		Name:      "Task 2",
		CreatedAt: now,
		SortOrder: 3000,
	})
	assert.Equal(t, 3000, state.GetHighestSortOrder())
}

func TestState_ApplyEvents(t *testing.T) {
	state := NewState()
	now := time.Now().UTC()

	events := []Event{
		TodoCreated{Type: "TodoCreated", ID: "1", Name: "Task 1", CreatedAt: now, SortOrder: 1000},
		TodoCreated{Type: "TodoCreated", ID: "2", Name: "Task 2", CreatedAt: now, SortOrder: 2000},
		TodoCompleted{Type: "TodoCompleted", ID: "1", CompletedAt: now},
	}

	state.ApplyEvents(events)

	todos := state.GetTodos()
	require.Len(t, todos, 2)

	// Find completed todo
	var todo1 *Todo
	for i := range todos {
		if todos[i].ID == "1" {
			todo1 = &todos[i]
			break
		}
	}
	require.NotNil(t, todo1)
	assert.NotNil(t, todo1.CompletedAt)
}

func TestState_IgnoreUnknownTodoID(t *testing.T) {
	state := NewState()

	// Should not panic when applying event for unknown todo
	state.Apply(TodoCompleted{
		Type:        "TodoCompleted",
		ID:          "nonexistent",
		CompletedAt: time.Now().UTC(),
	})

	todos := state.GetTodos()
	assert.Len(t, todos, 0)
}

func TestState_GetTodo(t *testing.T) {
	state := NewState()
	now := time.Now().UTC()

	// Add a todo
	state.Apply(TodoCreated{
		Type:      "TodoCreated",
		ID:        "test-todo",
		Name:      "Test Task",
		CreatedAt: now,
		SortOrder: 1000,
	})

	// Get existing todo
	todo, exists := state.GetTodo("test-todo")
	require.True(t, exists)
	assert.Equal(t, "test-todo", todo.ID)
	assert.Equal(t, "Test Task", todo.Name)

	// Get non-existent todo
	_, exists = state.GetTodo("nonexistent")
	assert.False(t, exists)
}

func TestState_TodoCount(t *testing.T) {
	state := NewState()

	// Empty state
	assert.Equal(t, 0, state.TodoCount())

	now := time.Now().UTC()

	// Add one todo
	state.Apply(TodoCreated{
		Type:      "TodoCreated",
		ID:        "todo-1",
		Name:      "Task 1",
		CreatedAt: now,
		SortOrder: 1000,
	})
	assert.Equal(t, 1, state.TodoCount())

	// Add another todo
	state.Apply(TodoCreated{
		Type:      "TodoCreated",
		ID:        "todo-2",
		Name:      "Task 2",
		CreatedAt: now,
		SortOrder: 2000,
	})
	assert.Equal(t, 2, state.TodoCount())
}

func TestState_ApplyListTitleChanged(t *testing.T) {
	state := NewState()

	// Check default title
	assert.Equal(t, "My Todo List", state.GetListTitle())

	// Apply ListTitleChanged event
	event := ListTitleChanged{
		Type:  "ListTitleChanged",
		Title: "Shopping List",
	}
	state.Apply(event)

	// Verify title was updated
	assert.Equal(t, "Shopping List", state.GetListTitle())

	// Apply another ListTitleChanged event
	event2 := ListTitleChanged{
		Type:  "ListTitleChanged",
		Title: "Work Tasks",
	}
	state.Apply(event2)

	// Verify title was updated again
	assert.Equal(t, "Work Tasks", state.GetListTitle())
}

func TestState_GetListTitle(t *testing.T) {
	state := NewState()

	// Test default title
	assert.Equal(t, "My Todo List", state.GetListTitle())
}

func TestState_CategoryProjectionAndCategorization(t *testing.T) {
	state := NewState()
	now := time.Now().UTC()
	ptr := func(s string) *string { return &s }

	state.Apply(CategoryCreated{
		Type:      "CategoryCreated",
		ID:        "cat-1",
		Name:      "Inbox",
		CreatedAt: now,
		SortOrder: 2000,
	})
	state.Apply(CategoryCreated{
		Type:      "CategoryCreated",
		ID:        "cat-2",
		Name:      "Work",
		CreatedAt: now,
		SortOrder: 1000,
	})

	cats := state.GetCategories()
	require.Len(t, cats, 2)
	assert.Equal(t, "cat-1", cats[0].ID) // highest sortOrder first

	// Create todo in cat-1
	state.Apply(TodoCreated{
		Type:       "TodoCreated",
		ID:         "todo-1",
		Name:       "Task",
		CreatedAt:  now,
		SortOrder:  100,
		CategoryID: ptr("cat-1"),
	})
	assert.True(t, state.CategoryHasTodos("cat-1"))

	// Move to uncategorized
	state.Apply(TodoCategorized{
		Type:       "TodoCategorized",
		ID:         "todo-1",
		CategoryID: nil,
	})
	assert.False(t, state.CategoryHasTodos("cat-1"))
}

