package main

import (
	"sort"
	"strings"
)

// State holds the current state of all todos, projected from events
type State struct {
	todos         map[string]*Todo
	listTitle     string
	nameFrequency map[string]int // Tracks frequency of todo names (case-insensitive key -> count)
	nameCanonical map[string]string // Maps lowercase name to most recent casing
}

// NewState creates a new empty state
func NewState() *State {
	return &State{
		todos:         make(map[string]*Todo),
		listTitle:     "My Todo List",
		nameFrequency: make(map[string]int),
		nameCanonical: make(map[string]string),
	}
}

// Apply applies a single event to the state
func (s *State) Apply(event Event) {
	switch e := event.(type) {
	case TodoCreated:
		s.todos[e.ID] = &Todo{
			ID:        e.ID,
			Name:      e.Name,
			CreatedAt: e.CreatedAt,
			SortOrder: e.SortOrder,
			Starred:   false,
		}
		// Track name frequency for autocomplete
		s.trackNameFrequency(e.Name)

	case TodoCompleted:
		if todo, ok := s.todos[e.ID]; ok {
			todo.CompletedAt = &e.CompletedAt
		}

	case TodoUncompleted:
		if todo, ok := s.todos[e.ID]; ok {
			todo.CompletedAt = nil
		}

	case TodoStarred:
		if todo, ok := s.todos[e.ID]; ok {
			todo.Starred = true
			todo.SortOrder = e.SortOrder
		}

	case TodoUnstarred:
		if todo, ok := s.todos[e.ID]; ok {
			todo.Starred = false
		}

	case TodoReordered:
		if todo, ok := s.todos[e.ID]; ok {
			todo.SortOrder = e.SortOrder
		}

	case TodoRenamed:
		if todo, ok := s.todos[e.ID]; ok {
			todo.Name = e.Name
			// Track name frequency for autocomplete
			s.trackNameFrequency(e.Name)
		}

	case ListTitleChanged:
		s.listTitle = e.Title
	}
}

// trackNameFrequency increments the frequency count for a name
func (s *State) trackNameFrequency(name string) {
	nameLower := strings.ToLower(name)
	s.nameFrequency[nameLower]++
	// Always update canonical to most recent casing
	s.nameCanonical[nameLower] = name
}

// ApplyEvents applies multiple events to the state
func (s *State) ApplyEvents(events []Event) {
	for _, event := range events {
		s.Apply(event)
	}
}

// GetTodos returns all todos sorted by sortOrder (descending - highest first)
func (s *State) GetTodos() []Todo {
	todos := make([]Todo, 0, len(s.todos))
	for _, todo := range s.todos {
		todos = append(todos, *todo)
	}

	// Sort by sortOrder descending
	sort.Slice(todos, func(i, j int) bool {
		return todos[i].SortOrder > todos[j].SortOrder
	})

	return todos
}

// GetTodo returns a single todo by ID
func (s *State) GetTodo(id string) (*Todo, bool) {
	todo, ok := s.todos[id]
	if !ok {
		return nil, false
	}
	// Return a copy
	todoCopy := *todo
	return &todoCopy, true
}

// GetHighestSortOrder returns the highest sortOrder among all todos
func (s *State) GetHighestSortOrder() int {
	highest := 0
	for _, todo := range s.todos {
		if todo.SortOrder > highest {
			highest = todo.SortOrder
		}
	}
	return highest
}

// TodoCount returns the number of todos
func (s *State) TodoCount() int {
	return len(s.todos)
}

// GetListTitle returns the current list title
func (s *State) GetListTitle() string {
	return s.listTitle
}

// GetNameFrequency returns a map of todo names (canonical casing) to their frequency count
func (s *State) GetNameFrequency() map[string]int {
	result := make(map[string]int)
	for nameLower, count := range s.nameFrequency {
		canonicalName := s.nameCanonical[nameLower]
		result[canonicalName] = count
	}
	return result
}

// GetActiveTodoNames returns the names of all active (not completed) todos
func (s *State) GetActiveTodoNames() []string {
	names := make([]string, 0)
	for _, todo := range s.todos {
		if todo.CompletedAt == nil {
			names = append(names, todo.Name)
		}
	}
	return names
}
