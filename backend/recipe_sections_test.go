package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecipeTotalSteps(t *testing.T) {
	assert.Equal(t, 0, recipeTotalSteps(nil))
	assert.Equal(t, 0, recipeTotalSteps([]RecipeSection{}))
	assert.Equal(t, 3, recipeTotalSteps([]RecipeSection{
		{Instructions: []string{"a", "b"}},
		{Instructions: []string{"c"}},
	}))
	assert.Equal(t, 0, recipeTotalSteps([]RecipeSection{
		{Name: "Empty", Ingredients: []Ingredient{{Name: "x"}}},
	}))
}

func TestRecipeTotalIngredients(t *testing.T) {
	assert.Equal(t, 0, recipeTotalIngredients(nil))
	assert.Equal(t, 4, recipeTotalIngredients([]RecipeSection{
		{Ingredients: []Ingredient{{Name: "a"}, {Name: "b"}}},
		{Ingredients: []Ingredient{{Name: "c"}, {Name: "d"}}},
	}))
}
