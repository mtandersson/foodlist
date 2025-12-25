package main

import (
	"sort"
	"strings"
)

// State holds the current state of all todos, projected from events
type State struct {
	todos             map[string]*Todo
	categories        map[string]*Category
	deletedCategories map[string]string // Maps deleted category IDs to their names (case-sensitive)
	listTitle         string
	nameFrequency     map[string]int     // Tracks frequency of todo names (case-insensitive key -> count)
	nameCanonical     map[string]string  // Maps lowercase name to most recent casing
	nameLastCategory  map[string]*string // Tracks last categoryId used for a name (lowercase)
}

// NewState creates a new empty state
func NewState() *State {
	return &State{
		todos:             make(map[string]*Todo),
		categories:        make(map[string]*Category),
		deletedCategories: make(map[string]string),
		listTitle:         "My Todo List",
		nameFrequency:     make(map[string]int),
		nameCanonical:     make(map[string]string),
		nameLastCategory:  make(map[string]*string),
	}
}

// Apply applies a single event to the state
func (s *State) Apply(event Event) {
	switch e := event.(type) {
	case TodoCreated:
		s.todos[e.ID] = &Todo{
			ID:         e.ID,
			Name:       e.Name,
			CreatedAt:  e.CreatedAt,
			SortOrder:  e.SortOrder,
			Starred:    false,
			CategoryID: e.CategoryID,
		}
		// Track name frequency for autocomplete
		s.trackNameFrequency(e.Name)
		s.trackLastCategory(e.Name, e.CategoryID)

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
			s.trackLastCategory(e.Name, todo.CategoryID)
		}

	case ListTitleChanged:
		s.listTitle = e.Title

	case TodoCategorized:
		if todo, ok := s.todos[e.ID]; ok {
			todo.CategoryID = e.CategoryID
			s.trackLastCategory(todo.Name, e.CategoryID)
		}

	case CategoryCreated:
		s.categories[e.ID] = &Category{
			ID:        e.ID,
			Name:      e.Name,
			CreatedAt: e.CreatedAt,
			SortOrder: e.SortOrder,
		}
		// Remove from deleted categories if it was deleted before
		delete(s.deletedCategories, e.ID)

	case CategoryRenamed:
		if cat, ok := s.categories[e.ID]; ok {
			cat.Name = e.Name
		}

	case CategoryDeleted:
		// Before deleting, store the category name for potential reuse
		if cat, ok := s.categories[e.ID]; ok {
			s.deletedCategories[e.ID] = cat.Name
		}
		delete(s.categories, e.ID)

	case CategoryReordered:
		if cat, ok := s.categories[e.ID]; ok {
			cat.SortOrder = e.SortOrder
		}
	}
}

// trackNameFrequency increments the frequency count for a name
func (s *State) trackNameFrequency(name string) {
	nameLower := strings.ToLower(name)
	s.nameFrequency[nameLower]++
	// Always update canonical to most recent casing
	s.nameCanonical[nameLower] = name
}

// trackLastCategory remembers the most recent category assignment for a name
func (s *State) trackLastCategory(name string, categoryID *string) {
	nameLower := strings.ToLower(name)
	if categoryID == nil {
		s.nameLastCategory[nameLower] = nil
		return
	}
	// Store a copy of the string value to avoid dangling pointer
	valueCopy := *categoryID
	s.nameLastCategory[nameLower] = &valueCopy
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

// GetHighestCategorySortOrder returns the highest sortOrder among categories
func (s *State) GetHighestCategorySortOrder() int {
	highest := 0
	for _, cat := range s.categories {
		if cat.SortOrder > highest {
			highest = cat.SortOrder
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

// GetCategories returns all categories sorted by sortOrder (descending)
func (s *State) GetCategories() []Category {
	cats := make([]Category, 0, len(s.categories))
	for _, cat := range s.categories {
		cats = append(cats, *cat)
	}
	sort.Slice(cats, func(i, j int) bool {
		return cats[i].SortOrder > cats[j].SortOrder
	})
	return cats
}

// GetCategory returns a single category by ID (returns pointer to internal state, not a copy)
func (s *State) GetCategory(id string) (*Category, bool) {
	cat, ok := s.categories[id]
	if !ok {
		return nil, false
	}
	return cat, true
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

// CategoryHasTodos returns true if any todo is assigned to the given categoryId
func (s *State) CategoryHasTodos(categoryID string) bool {
	for _, todo := range s.todos {
		if todo.CategoryID != nil && *todo.CategoryID == categoryID {
			return true
		}
	}
	return false
}

// GetLastCategoryForName returns the last category used for a given name (if any)
func (s *State) GetLastCategoryForName(name string) *string {
	return s.nameLastCategory[strings.ToLower(name)]
}

// FindDeletedCategoryByName returns the ID of a deleted category with the given name (case-sensitive)
// Returns empty string if no such deleted category exists
func (s *State) FindDeletedCategoryByName(name string) string {
	for id, deletedName := range s.deletedCategories {
		if deletedName == name {
			return id
		}
	}
	return ""
}

// CategoryNameExists checks if an active category with the given name exists (case-sensitive)
func (s *State) CategoryNameExists(name string) bool {
	for _, cat := range s.categories {
		if cat.Name == name {
			return true
		}
	}
	return false
}
