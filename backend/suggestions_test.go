package main

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// unitVec returns a unit vector at angle theta in the X-Y plane. Cosine
// similarity between unitVec(a) and unitVec(b) equals cos(a - b), so tests
// can place items at known similarity distances.
func unitVec(theta float64) []float32 {
	return []float32{float32(math.Cos(theta)), float32(math.Sin(theta)), 0}
}

// newTestEngine builds an engine with a pinned clock and the documented
// defaults. Most tests override one or two fields.
func newTestEngine(now time.Time) *SuggestionEngine {
	return NewSuggestionEngine(SuggestionEngineConfig{
		MinPurchases:    DefaultSuggestionsMinPurchases,
		MaxInterval:     time.Duration(DefaultSuggestionsMaxIntervalDays) * 24 * time.Hour,
		DueFraction:     DefaultSuggestionsDueFraction,
		DedupSimilarity: DefaultSuggestionsDedupSimilarity,
		RecentLimit:     DefaultSuggestionsRecentLimit,
		Now:             func() time.Time { return now },
	})
}

// makeCompleted returns a completed todo with the given name and absolute
// completion time. All "purchases" use TodoCompleted semantics.
func makeCompleted(id, name string, completedAt time.Time) Todo {
	t := completedAt
	return Todo{
		ID:          id,
		Name:        name,
		CreatedAt:   completedAt.Add(-time.Hour),
		CompletedAt: &t,
	}
}

// TestSuggestions_TooFewPurchases verifies that an item bought fewer than
// MinPurchases times never appears, even with a perfect interval and being
// long-overdue.
func TestSuggestions_TooFewPurchases(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	engine := newTestEngine(now)

	todos := []Todo{
		makeCompleted("a", "Mjölk", now.Add(-21*24*time.Hour)),
		makeCompleted("b", "Mjölk", now.Add(-14*24*time.Hour)),
	}
	added, _ := engine.Recompute(todos, nil, nil)
	require.Empty(t, added, "should not suggest items below MinPurchases")
}

// TestSuggestions_IntervalTooLong asserts that items bought less than once
// every MAX_INTERVAL_DAYS are filtered out.
func TestSuggestions_IntervalTooLong(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	engine := newTestEngine(now)

	// Bought once a year — far below "once every 3 months".
	todos := []Todo{
		makeCompleted("a", "Julgran", now.Add(-3*365*24*time.Hour)),
		makeCompleted("b", "Julgran", now.Add(-2*365*24*time.Hour)),
		makeCompleted("c", "Julgran", now.Add(-365*24*time.Hour)),
	}
	added, _ := engine.Recompute(todos, nil, nil)
	require.Empty(t, added, "items bought once a year should be filtered out")
}

// TestSuggestions_NotYetDue asserts the 2/3 due-fraction guard. An item
// bought regularly but very recently should not be suggested yet.
func TestSuggestions_NotYetDue(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	engine := newTestEngine(now)

	// Weekly cadence; last purchase 1 day ago — < 2/3 * 7d = 4.67d.
	todos := []Todo{
		makeCompleted("a", "Mjölk", now.Add(-22*24*time.Hour)),
		makeCompleted("b", "Mjölk", now.Add(-15*24*time.Hour)),
		makeCompleted("c", "Mjölk", now.Add(-8*24*time.Hour)),
		makeCompleted("d", "Mjölk", now.Add(-24*time.Hour)),
	}
	added, _ := engine.Recompute(todos, nil, nil)
	require.Empty(t, added, "items not yet due should not be suggested")
}

// TestSuggestions_Due verifies the happy path: a frequently-bought item
// whose due-fraction has elapsed becomes a suggestion with the correct
// derived fields.
func TestSuggestions_Due(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	engine := newTestEngine(now)

	// Weekly cadence; last purchase 5 days ago — past 2/3 * 7d = 4.67d.
	todos := []Todo{
		makeCompleted("a", "Mjölk", now.Add(-26*24*time.Hour)),
		makeCompleted("b", "Mjölk", now.Add(-19*24*time.Hour)),
		makeCompleted("c", "Mjölk", now.Add(-12*24*time.Hour)),
		makeCompleted("d", "Mjölk", now.Add(-5*24*time.Hour)),
	}
	added, _ := engine.Recompute(todos, nil, nil)
	require.Len(t, added, 1)
	s := added[0]
	require.Equal(t, "Mjölk", s.Name)
	require.Equal(t, 4, s.PurchaseCount)
	require.InDelta(t, 7*86400, s.AvgIntervalSeconds, 0.1)
	require.Equal(t, suggestionIDFor("Mjölk"), s.ID)
}

// TestSuggestions_DedupAgainstActiveByName ensures suggestions never include
// an item that is already in the active shopping list (even without
// embeddings).
func TestSuggestions_DedupAgainstActiveByName(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	engine := newTestEngine(now)

	todos := []Todo{
		makeCompleted("a", "Mjölk", now.Add(-26*24*time.Hour)),
		makeCompleted("b", "Mjölk", now.Add(-19*24*time.Hour)),
		makeCompleted("c", "Mjölk", now.Add(-12*24*time.Hour)),
		makeCompleted("d", "Mjölk", now.Add(-5*24*time.Hour)),
		// Active todo with the same name — should suppress the suggestion.
		{
			ID:          "active",
			Name:        "Mjölk",
			CreatedAt:   now.Add(-time.Hour),
			CompletedAt: nil,
		},
	}
	added, _ := engine.Recompute(todos, nil, nil)
	require.Empty(t, added, "suggestion should be deduped against the active todo")
}

// TestSuggestions_DedupAgainstActiveByEmbedding verifies that semantically
// similar active items (e.g. "lättmjölk" vs "mjölk") also block the
// suggestion via embeddings.
func TestSuggestions_DedupAgainstActiveByEmbedding(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	engine := newTestEngine(now)

	embeddings := map[string]CachedEmbedding{
		"mjölk":     {Key: "mjölk", Vector: unitVec(0)},
		"lättmjölk": {Key: "lättmjölk", Vector: unitVec(0.1)}, // cos ~ 0.995
	}
	todos := []Todo{
		makeCompleted("a", "Mjölk", now.Add(-26*24*time.Hour)),
		makeCompleted("b", "Mjölk", now.Add(-19*24*time.Hour)),
		makeCompleted("c", "Mjölk", now.Add(-12*24*time.Hour)),
		makeCompleted("d", "Mjölk", now.Add(-5*24*time.Hour)),
		{
			ID:          "active",
			Name:        "Lättmjölk",
			CreatedAt:   now.Add(-time.Hour),
			CompletedAt: nil,
		},
	}
	added, _ := engine.Recompute(todos, embeddings, nil)
	require.Empty(t, added, "embedding-similar active item should suppress suggestion")
}

// TestSuggestions_EmbeddingClustering verifies that two near-synonyms collapse
// into a single suggestion sharing one ID, with the most-purchased variant
// chosen as canonical name.
func TestSuggestions_EmbeddingClustering(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	engine := newTestEngine(now)

	embeddings := map[string]CachedEmbedding{
		"mjölk":     {Key: "mjölk", Vector: unitVec(0)},
		"lättmjölk": {Key: "lättmjölk", Vector: unitVec(0.1)},
	}
	todos := []Todo{
		// "mjölk" twice
		makeCompleted("m1", "mjölk", now.Add(-28*24*time.Hour)),
		makeCompleted("m2", "mjölk", now.Add(-21*24*time.Hour)),
		// "lättmjölk" three times — should win as canonical (more
		// completions in this cluster).
		makeCompleted("l1", "lättmjölk", now.Add(-14*24*time.Hour)),
		makeCompleted("l2", "lättmjölk", now.Add(-7*24*time.Hour)),
		makeCompleted("l3", "lättmjölk", now.Add(-5*24*time.Hour)),
	}
	added, _ := engine.Recompute(todos, embeddings, nil)
	require.Len(t, added, 1, "cluster should collapse into a single suggestion")
	require.Equal(t, "lättmjölk", added[0].Name)
	require.Equal(t, 5, added[0].PurchaseCount)
}

// TestSuggestions_StableID asserts that the same canonical name yields the
// same suggestion ID across repeated recomputes — which is the contract the
// delta-broadcast logic depends on.
func TestSuggestions_StableID(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	engine := newTestEngine(now)

	todos := []Todo{
		makeCompleted("a", "Mjölk", now.Add(-26*24*time.Hour)),
		makeCompleted("b", "Mjölk", now.Add(-19*24*time.Hour)),
		makeCompleted("c", "Mjölk", now.Add(-12*24*time.Hour)),
		makeCompleted("d", "Mjölk", now.Add(-5*24*time.Hour)),
	}
	added1, _ := engine.Recompute(todos, nil, nil)
	require.Len(t, added1, 1)
	id1 := added1[0].ID

	added2, removed2 := engine.Recompute(todos, nil, nil)
	require.Empty(t, added2, "no diff expected on identical inputs")
	require.Empty(t, removed2)

	// Manually verify a third recompute still produces the same ID
	// (sanity check that re-init from scratch is deterministic).
	engine2 := newTestEngine(now)
	added3, _ := engine2.Recompute(todos, nil, nil)
	require.Len(t, added3, 1)
	require.Equal(t, id1, added3[0].ID, "ID must be stable across engines")
}

// TestSuggestions_DiffAddedRemoved verifies that adding an active todo for a
// previously-suggested item causes a SuggestionRemoved delta on the next
// recompute, and removing it later restores the suggestion.
func TestSuggestions_DiffAddedRemoved(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	engine := newTestEngine(now)

	completed := []Todo{
		makeCompleted("a", "Mjölk", now.Add(-26*24*time.Hour)),
		makeCompleted("b", "Mjölk", now.Add(-19*24*time.Hour)),
		makeCompleted("c", "Mjölk", now.Add(-12*24*time.Hour)),
		makeCompleted("d", "Mjölk", now.Add(-5*24*time.Hour)),
	}
	added1, _ := engine.Recompute(completed, nil, nil)
	require.Len(t, added1, 1)
	mjölkID := added1[0].ID

	// User adds Mjölk to the active list: suggestion should disappear.
	withActive := append([]Todo{}, completed...)
	withActive = append(withActive, Todo{
		ID:        "active",
		Name:      "Mjölk",
		CreatedAt: now.Add(-time.Hour),
	})
	added2, removed2 := engine.Recompute(withActive, nil, nil)
	require.Empty(t, added2)
	require.Equal(t, []string{mjölkID}, removed2)

	// Active is removed again: suggestion comes back, same ID.
	added3, removed3 := engine.Recompute(completed, nil, nil)
	require.Empty(t, removed3)
	require.Len(t, added3, 1)
	require.Equal(t, mjölkID, added3[0].ID)
}

// TestSuggestions_CategoryFromVotes confirms the suggestion picks the most
// frequently-used category from past completions, and only references
// categories that still exist.
func TestSuggestions_CategoryFromVotes(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	engine := newTestEngine(now)

	dairy := "11111111-1111-4111-8111-111111111111"
	bread := "22222222-2222-4222-8222-222222222222"
	dairyPtr := dairy
	breadPtr := bread

	mkCat := func(todoID, categoryID string, when time.Time) Todo {
		t := when
		cid := categoryID
		return Todo{
			ID:          todoID,
			Name:        "Mjölk",
			CreatedAt:   when.Add(-time.Hour),
			CompletedAt: &t,
			CategoryID:  &cid,
		}
	}
	_ = dairyPtr
	_ = breadPtr

	todos := []Todo{
		// 3 votes for dairy, 1 for bread (7d cadence, due now)
		mkCat("t1", dairy, now.Add(-28*24*time.Hour)),
		mkCat("t2", dairy, now.Add(-21*24*time.Hour)),
		mkCat("t3", dairy, now.Add(-14*24*time.Hour)),
		mkCat("t4", bread, now.Add(-7*24*time.Hour)),
	}
	cats := []Category{
		{ID: dairy, Name: "Mejeri"},
		{ID: bread, Name: "Bröd"},
	}
	added, _ := engine.Recompute(todos, nil, cats)
	require.Len(t, added, 1)
	require.NotNil(t, added[0].CategoryID)
	require.Equal(t, dairy, *added[0].CategoryID)
	require.NotNil(t, added[0].CategoryName)
	require.Equal(t, "Mejeri", *added[0].CategoryName)
}

// TestSuggestions_MarkPurchased verifies the optimistic short-circuit used
// when the engine sees a TodoCreated for an item that currently has a
// matching suggestion.
func TestSuggestions_MarkPurchased(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	engine := newTestEngine(now)

	todos := []Todo{
		makeCompleted("a", "Mjölk", now.Add(-26*24*time.Hour)),
		makeCompleted("b", "Mjölk", now.Add(-19*24*time.Hour)),
		makeCompleted("c", "Mjölk", now.Add(-12*24*time.Hour)),
		makeCompleted("d", "Mjölk", now.Add(-5*24*time.Hour)),
	}
	added, _ := engine.Recompute(todos, nil, nil)
	require.Len(t, added, 1)

	id, ok := engine.MarkPurchased("Mjölk")
	require.True(t, ok)
	require.Equal(t, added[0].ID, id)

	// Subsequent calls for the same item return false.
	_, ok = engine.MarkPurchased("Mjölk")
	require.False(t, ok)

	// Unknown items return false too.
	_, ok = engine.MarkPurchased("Bananer")
	require.False(t, ok)
}

// TestSuggestions_RecentLimit ensures only the last N completions feed the
// interval calculation. Without this cap, very old, sparse historical data
// would drag the average out and exclude items that are now bought weekly.
func TestSuggestions_RecentLimit(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	cfg := SuggestionEngineConfig{
		MinPurchases:    3,
		MaxInterval:     90 * 24 * time.Hour,
		DueFraction:     0.667,
		DedupSimilarity: 0.85,
		RecentLimit:     6,
		Now:             func() time.Time { return now },
	}
	engine := NewSuggestionEngine(cfg)

	// 4 very old purchases (one a year, decades ago) plus 6 weekly recent
	// ones. Without RecentLimit the average would be enormous; with it the
	// interval should be ~7 days.
	todos := []Todo{
		makeCompleted("old1", "Mjölk", now.Add(-365*24*4*time.Hour)),
		makeCompleted("old2", "Mjölk", now.Add(-365*24*3*time.Hour)),
		makeCompleted("old3", "Mjölk", now.Add(-365*24*2*time.Hour)),
		makeCompleted("old4", "Mjölk", now.Add(-365*24*time.Hour)),
		makeCompleted("r1", "Mjölk", now.Add(-42*24*time.Hour)),
		makeCompleted("r2", "Mjölk", now.Add(-35*24*time.Hour)),
		makeCompleted("r3", "Mjölk", now.Add(-28*24*time.Hour)),
		makeCompleted("r4", "Mjölk", now.Add(-21*24*time.Hour)),
		makeCompleted("r5", "Mjölk", now.Add(-14*24*time.Hour)),
		makeCompleted("r6", "Mjölk", now.Add(-7*24*time.Hour)),
	}
	added, _ := engine.Recompute(todos, nil, nil)
	require.Len(t, added, 1)
	require.InDelta(t, 7*86400, added[0].AvgIntervalSeconds, 0.1)
	require.Equal(t, 10, added[0].PurchaseCount, "purchase count uses every completion")
}
