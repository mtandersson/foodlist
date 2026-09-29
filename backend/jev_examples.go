package main

import (
	"sort"
	"strings"
	"time"
)

const defaultJevExamplesPerCategory = 4

type categoryExampleCandidate struct {
	name      string
	completed bool
	at        time.Time
}

// buildCategoryExamples selects a small, deterministic set of grocery names
// from Foodlist's projected history for each live category. Completed items
// are preferred because they have made it through the shopping flow; other
// categorized items are used only when needed to fill the bounded set.
func buildCategoryExamples(todos []Todo, categories []Category, currentItem string, limit int) map[string][]string {
	if limit <= 0 || len(categories) == 0 {
		return nil
	}

	live := make(map[string]struct{}, len(categories))
	for _, category := range categories {
		live[category.ID] = struct{}{}
	}

	currentKey := normalizeName(currentItem)
	candidates := make(map[string][]categoryExampleCandidate, len(categories))
	for _, todo := range todos {
		if todo.CategoryID == nil {
			continue
		}
		categoryID := *todo.CategoryID
		if _, ok := live[categoryID]; !ok {
			continue
		}
		name := strings.TrimSpace(todo.Name)
		key := normalizeName(name)
		if name == "" || len(name) > maxJevExampleNameBytes || key == "" || key == currentKey {
			continue
		}

		at := todo.CreatedAt
		completed := todo.CompletedAt != nil
		if completed {
			at = *todo.CompletedAt
		}
		candidates[categoryID] = append(candidates[categoryID], categoryExampleCandidate{
			name:      name,
			completed: completed,
			at:        at,
		})
	}

	result := make(map[string][]string, len(candidates))
	for categoryID, items := range candidates {
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].completed != items[j].completed {
				return items[i].completed
			}
			if !items[i].at.Equal(items[j].at) {
				return items[i].at.After(items[j].at)
			}
			return normalizeName(items[i].name) < normalizeName(items[j].name)
		})

		seen := make(map[string]struct{}, len(items))
		for _, item := range items {
			key := normalizeName(item.name)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			result[categoryID] = append(result[categoryID], item.name)
			if len(result[categoryID]) == limit {
				break
			}
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
