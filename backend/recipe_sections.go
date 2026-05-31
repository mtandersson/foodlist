package main

// RecipeSection groups a related list of ingredients and instructions under
// an optional heading (e.g. "Sås", "Sallad"). A recipe with no logical
// grouping uses a single section with Name == "".
type RecipeSection struct {
	Name         string       `json:"name"`
	Ingredients  []Ingredient `json:"ingredients"`
	Instructions []string     `json:"instructions"`
}

// recipeTotalSteps returns the number of instructions across every
// section. Used by every consumer that needs to map a flat cook-mode
// step index back to a per-section row (server.go bounds check,
// recipes_api.go PruneAbove, MCP tools, frontend global indexing).
func recipeTotalSteps(sections []RecipeSection) int {
	n := 0
	for _, s := range sections {
		n += len(s.Instructions)
	}
	return n
}

// recipeTotalIngredients mirrors recipeTotalSteps for ingredients. The
// flat order is section[0].Ingredients ++ section[1].Ingredients ++ ...
// and is exposed to MCP as a 1-based global index.
func recipeTotalIngredients(sections []RecipeSection) int {
	n := 0
	for _, s := range sections {
		n += len(s.Ingredients)
	}
	return n
}
