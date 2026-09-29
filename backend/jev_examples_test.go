package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildCategoryExamplesPrefersCompletedDedupesAndExcludesCurrentItem(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	dry := "dry"
	produce := "produce"
	completedOld := now.Add(-72 * time.Hour)
	completedNew := now.Add(-24 * time.Hour)

	todos := []Todo{
		{ID: "1", Name: "havregryn", CreatedAt: now.Add(-10 * 24 * time.Hour), CompletedAt: &completedOld, CategoryID: &dry},
		{ID: "2", Name: "Makaroner", CreatedAt: now.Add(-8 * 24 * time.Hour), CompletedAt: &completedNew, CategoryID: &dry},
		{ID: "3", Name: "makaroner", CreatedAt: now.Add(-7 * 24 * time.Hour), CategoryID: &dry},
		{ID: "4", Name: "maizena", CreatedAt: now.Add(-2 * time.Hour), CategoryID: &dry},
		{ID: "5", Name: "strösocker", CreatedAt: now.Add(-time.Hour), CategoryID: &dry},
		{ID: "6", Name: "broccoli", CreatedAt: now.Add(-time.Hour), CompletedAt: &completedNew, CategoryID: &produce},
		{ID: "7", Name: "uncategorized", CreatedAt: now},
	}

	got := buildCategoryExamples(
		todos,
		[]Category{{ID: dry, Name: "Torrvaror"}, {ID: produce, Name: "Frukt & Grönt"}},
		"strösocker",
		3,
	)

	require.Equal(t, []string{"Makaroner", "havregryn", "maizena"}, got[dry])
	require.Equal(t, []string{"broccoli"}, got[produce])
}

func TestBuildCategoryExamplesIgnoresDeletedCategoriesAndRespectsLimit(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	live := "live"
	deleted := "deleted"

	todos := []Todo{
		{ID: "1", Name: "one", CreatedAt: now.Add(-time.Hour), CategoryID: &live},
		{ID: "2", Name: "two", CreatedAt: now.Add(-2 * time.Hour), CategoryID: &live},
		{ID: "3", Name: "three", CreatedAt: now.Add(-3 * time.Hour), CategoryID: &live},
		{ID: "4", Name: "old deleted item", CreatedAt: now, CategoryID: &deleted},
	}

	got := buildCategoryExamples(todos, []Category{{ID: live, Name: "Live"}}, "", 2)

	require.Len(t, got[live], 2)
	require.NotContains(t, got, deleted)
}
