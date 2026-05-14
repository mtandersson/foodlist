package main

import (
	"fmt"
	"math"
	"testing"
	"time"
)

// unit3 returns a unit vector at angle theta in the X-Y plane. Cosine
// similarity between unit3(a) and unit3(b) equals cos(a - b), giving every
// test an exact closed-form similarity value.
func unit3(theta float64) []float32 {
	return []float32{float32(math.Cos(theta)), float32(math.Sin(theta)), 0}
}

// defaultCategorizer is the standard config used by every table case.
// Now is pinned via the returned testClock.
type testClock struct{ now time.Time }

func newTestCategorizer(now time.Time) (Categorizer, *testClock) {
	clk := &testClock{now: now}
	cat := NewCategorizer(Categorizer{
		SimilarityFloor:     0.55,
		RecencyWindow:       30 * 24 * time.Hour,
		RecentWeight:        0.70,
		PopularityWeight:    0.30,
		MaxSimGate:          0.20,
		AcceptanceThreshold: 0.30,
		Now:                 func() time.Time { return clk.now },
	})
	return cat, clk
}

// fixture builds a synthetic state: it allocates a Todo and a cached
// embedding vector for each row, with a per-row creation offset relative to
// `now`. Returns todos, embeddings, and the live-category set.
type fixtureRow struct {
	name       string
	categoryID string
	theta      float64       // angle for the embedding vector
	ageBefore  time.Duration // how long ago this todo was created relative to `now`
	completed  bool
	noEmbed    bool // if true, do not register an embedding (cache miss simulation)
}

func buildFixture(now time.Time, rows []fixtureRow, liveCats ...string) ([]Todo, map[string]CachedEmbedding, map[string]struct{}) {
	todos := make([]Todo, 0, len(rows))
	embs := make(map[string]CachedEmbedding)
	for i, r := range rows {
		id := fmt.Sprintf("todo-%d", i)
		cidCopy := r.categoryID
		var catPtr *string
		if cidCopy != "" {
			catPtr = &cidCopy
		}
		t := Todo{
			ID:         id,
			Name:       r.name,
			CreatedAt:  now.Add(-r.ageBefore),
			CategoryID: catPtr,
		}
		if r.completed {
			c := now.Add(-r.ageBefore + time.Hour)
			t.CompletedAt = &c
		}
		todos = append(todos, t)
		if !r.noEmbed {
			key := normalizeName(r.name)
			embs[key] = CachedEmbedding{
				Key:    key,
				Text:   key,
				Model:  "test",
				Dim:    3,
				Vector: unit3(r.theta),
			}
		}
	}
	live := make(map[string]struct{}, len(liveCats))
	for _, c := range liveCats {
		live[c] = struct{}{}
	}
	return todos, embs, live
}

// Reference angles (cosine similarity vs query at theta=0):
//
//	near = 0.32   -> cos ≈ 0.9492    (floored ≈ 0.3992 vs floor 0.55)
//	mid  = 0.65   -> cos ≈ 0.7961    (floored ≈ 0.2461; above MaxSimGate=0.20)
//	weak = 1.05   -> cos ≈ 0.4976    (below floor → contributes 0)
//	far  = 1.25   -> cos ≈ 0.3153    (well below floor)
const (
	angleNear = 0.32
	angleMid  = 0.65
	angleWeak = 1.05
	angleFar  = 1.25
)

func TestSuggestCategory_FloorDiscardsNoise(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	cat, _ := newTestCategorizer(now)
	todos, embs, live := buildFixture(now, []fixtureRow{
		{name: "noisy", categoryID: "A", theta: angleFar, ageBefore: time.Hour},
	}, "A")

	got := cat.SuggestCategory(unit3(0), todos, embs, live)
	if got != nil {
		t.Fatalf("expected nil suggestion, got %+v", got)
	}
}

func TestSuggestCategory_QualityWinsAtEqualVolume(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	cat, _ := newTestCategorizer(now)
	todos, embs, live := buildFixture(now, []fixtureRow{
		{name: "near", categoryID: "A", theta: angleNear, ageBefore: time.Hour},
		{name: "mid", categoryID: "B", theta: angleMid, ageBefore: time.Hour},
	}, "A", "B")

	got := cat.SuggestCategory(unit3(0), todos, embs, live)
	if got == nil {
		t.Fatalf("expected a suggestion")
	}
	if got.CategoryID != "A" {
		t.Fatalf("expected category A to win, got %q (score=%v)", got.CategoryID, got.Score)
	}
}

func TestSuggestCategory_VolumeWinsOverSingleGreatMatch(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	cat, _ := newTestCategorizer(now)
	rows := []fixtureRow{
		{name: "near-only", categoryID: "B", theta: angleNear, ageBefore: time.Hour},
	}
	for i := 0; i < 5; i++ {
		rows = append(rows, fixtureRow{
			name: fmt.Sprintf("mid-%d", i), categoryID: "A", theta: angleMid, ageBefore: time.Hour,
		})
	}
	todos, embs, live := buildFixture(now, rows, "A", "B")

	got := cat.SuggestCategory(unit3(0), todos, embs, live)
	if got == nil {
		t.Fatalf("expected a suggestion")
	}
	if got.CategoryID != "A" {
		t.Fatalf("expected volume-heavy A to win, got %q (score=%v)", got.CategoryID, got.Score)
	}
}

func TestSuggestCategory_RecentDominatesOldSameQuality(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	cat, _ := newTestCategorizer(now)
	rows := []fixtureRow{}
	for i := 0; i < 3; i++ {
		rows = append(rows, fixtureRow{
			name: fmt.Sprintf("a-%d", i), categoryID: "A", theta: angleNear, ageBefore: time.Hour,
		})
		rows = append(rows, fixtureRow{
			name: fmt.Sprintf("b-%d", i), categoryID: "B", theta: angleNear, ageBefore: 90 * 24 * time.Hour,
		})
	}
	todos, embs, live := buildFixture(now, rows, "A", "B")

	got := cat.SuggestCategory(unit3(0), todos, embs, live)
	if got == nil {
		t.Fatalf("expected a suggestion")
	}
	if got.CategoryID != "A" {
		t.Fatalf("expected recent A to win over ancient B, got %q (score=%v)", got.CategoryID, got.Score)
	}
}

func TestSuggestCategory_OldStrongLosesToRecentWeaker(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	cat, _ := newTestCategorizer(now)
	rows := []fixtureRow{}
	for i := 0; i < 3; i++ {
		rows = append(rows, fixtureRow{
			name: fmt.Sprintf("a-old-%d", i), categoryID: "A", theta: angleNear, ageBefore: 90 * 24 * time.Hour,
		})
		rows = append(rows, fixtureRow{
			name: fmt.Sprintf("b-new-%d", i), categoryID: "B", theta: angleMid, ageBefore: time.Hour,
		})
	}
	todos, embs, live := buildFixture(now, rows, "A", "B")

	got := cat.SuggestCategory(unit3(0), todos, embs, live)
	if got == nil {
		t.Fatalf("expected a suggestion")
	}
	// Manually compute both scores from the formula to verify the cross-over.
	flNear := flooredSim(float32(math.Cos(angleNear)), 0.55)
	flMid := flooredSim(float32(math.Cos(angleMid)), 0.55)
	// A: 3 items, ancient → only pass B contributes.
	aBlended := blend(0, 3*flNear, 0.70)
	aFinal := aBlended * popularity(3, 0.30)
	// B: 3 items, recent → both passes equal.
	bRecent := 3 * flMid
	bAll := 3 * flMid
	bBlended := blend(bRecent, bAll, 0.70)
	bFinal := bBlended * popularity(3, 0.30)
	if !(bFinal > aFinal) {
		t.Fatalf("test setup wrong: bFinal=%v aFinal=%v", bFinal, aFinal)
	}
	if got.CategoryID != "B" {
		t.Fatalf("expected recent-weaker B to win (B=%v A=%v), got %q score=%v",
			bFinal, aFinal, got.CategoryID, got.Score)
	}
}

func TestSuggestCategory_MovedCategoryUsesCurrentLabel(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	cat, _ := newTestCategorizer(now)
	// One near-similar item that originally was in A but is now in B
	// (CategoryID reflects the latest TodoCategorized). State.Apply already
	// updates the field; we just present the current state.
	todos, embs, live := buildFixture(now, []fixtureRow{
		{name: "moved", categoryID: "B", theta: angleNear, ageBefore: 24 * time.Hour},
	}, "A", "B")

	got := cat.SuggestCategory(unit3(0), todos, embs, live)
	if got == nil {
		t.Fatalf("expected a suggestion")
	}
	if got.CategoryID != "B" {
		t.Fatalf("expected moved item to credit its CURRENT category B, got %q", got.CategoryID)
	}
}

func TestSuggestCategory_CheeseDairyVsBakeryPopularityBreaksTie(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	cat, _ := newTestCategorizer(now)
	rows := []fixtureRow{
		{name: "milk", categoryID: "dairy", theta: angleNear, ageBefore: 24 * time.Hour},
		{name: "bread", categoryID: "bakery", theta: angleNear, ageBefore: 24 * time.Hour},
	}
	for i := 0; i < 9; i++ {
		rows = append(rows, fixtureRow{
			name: fmt.Sprintf("dairy-filler-%d", i), categoryID: "dairy", theta: angleFar, ageBefore: 24 * time.Hour,
		})
	}
	for i := 0; i < 3; i++ {
		rows = append(rows, fixtureRow{
			name: fmt.Sprintf("bakery-filler-%d", i), categoryID: "bakery", theta: angleFar, ageBefore: 24 * time.Hour,
		})
	}
	todos, embs, live := buildFixture(now, rows, "dairy", "bakery")

	got := cat.SuggestCategory(unit3(0), todos, embs, live)
	if got == nil {
		t.Fatalf("expected a suggestion")
	}
	if got.CategoryID != "dairy" {
		t.Fatalf("expected dairy to win on popularity (N=10 vs 4), got %q score=%v",
			got.CategoryID, got.Score)
	}
	// Confirm the score ratio matches the popularity ratio (semantic score
	// is identical for both categories: a single matching item).
	expectedRatio := popularity(10, 0.30) / popularity(4, 0.30)
	// We don't see the loser's score directly. Recompute it.
	flNear := flooredSim(float32(math.Cos(angleNear)), 0.55)
	dairyBlended := blend(flNear, flNear, 0.70)
	bakeryBlended := blend(flNear, flNear, 0.70)
	dairyFinal := dairyBlended * popularity(10, 0.30)
	bakeryFinal := bakeryBlended * popularity(4, 0.30)
	gotRatio := dairyFinal / bakeryFinal
	if !approxEqual(gotRatio, expectedRatio, 1e-5) {
		t.Fatalf("expected score ratio %v, got %v", expectedRatio, gotRatio)
	}
	if !approxEqual(got.Score, dairyFinal, 1e-5) {
		t.Fatalf("expected winning score %v, got %v", dairyFinal, got.Score)
	}
}

func TestSuggestCategory_QualityOverridesPopularity(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	cat, _ := newTestCategorizer(now)
	rows := []fixtureRow{
		{name: "milk", categoryID: "dairy", theta: angleMid, ageBefore: 24 * time.Hour},
		{name: "bread", categoryID: "bakery", theta: angleNear, ageBefore: 24 * time.Hour},
	}
	for i := 0; i < 9; i++ {
		rows = append(rows, fixtureRow{
			name: fmt.Sprintf("dairy-filler-%d", i), categoryID: "dairy", theta: angleFar, ageBefore: 24 * time.Hour,
		})
	}
	for i := 0; i < 3; i++ {
		rows = append(rows, fixtureRow{
			name: fmt.Sprintf("bakery-filler-%d", i), categoryID: "bakery", theta: angleFar, ageBefore: 24 * time.Hour,
		})
	}
	todos, embs, live := buildFixture(now, rows, "dairy", "bakery")

	got := cat.SuggestCategory(unit3(0), todos, embs, live)
	if got == nil {
		t.Fatalf("expected a suggestion")
	}
	if got.CategoryID != "bakery" {
		t.Fatalf("expected bakery (clearly higher per-item sim) to beat dairy's size advantage, got %q score=%v",
			got.CategoryID, got.Score)
	}
}

func TestSuggestCategory_PopularityDoesNotPromoteUnrelated(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	cat, _ := newTestCategorizer(now)
	rows := []fixtureRow{
		{name: "bread", categoryID: "bakery", theta: angleNear, ageBefore: 24 * time.Hour},
	}
	for i := 0; i < 50; i++ {
		rows = append(rows, fixtureRow{
			name: fmt.Sprintf("dairy-noise-%d", i), categoryID: "dairy", theta: angleFar, ageBefore: 24 * time.Hour,
		})
	}
	todos, embs, live := buildFixture(now, rows, "dairy", "bakery")

	got := cat.SuggestCategory(unit3(0), todos, embs, live)
	if got == nil {
		t.Fatalf("expected a suggestion")
	}
	if got.CategoryID != "bakery" {
		t.Fatalf("expected smaller related bakery to beat huge unrelated dairy, got %q", got.CategoryID)
	}
}

func TestSuggestCategory_MaxSimGateBlocksVolumeSpam(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	cat, _ := newTestCategorizer(now)
	// 50 items at sim ~0.60 (floored ≈ 0.05, below MaxSimGate=0.20).
	// Pure volume would give blended ≈ 2.5; popularity bumps it well above 0.30.
	weakTheta := math.Acos(0.60) // gives cos ≈ 0.60 exactly
	rows := []fixtureRow{
		{name: "single-strong", categoryID: "B", theta: angleNear, ageBefore: time.Hour},
	}
	for i := 0; i < 50; i++ {
		rows = append(rows, fixtureRow{
			name: fmt.Sprintf("a-weak-%d", i), categoryID: "A", theta: weakTheta, ageBefore: time.Hour,
		})
	}
	todos, embs, live := buildFixture(now, rows, "A", "B")

	got := cat.SuggestCategory(unit3(0), todos, embs, live)
	if got == nil {
		t.Fatalf("expected a suggestion")
	}
	if got.CategoryID != "B" {
		t.Fatalf("expected gate to filter out volume-only A and let B win, got %q score=%v",
			got.CategoryID, got.Score)
	}
}

func TestSuggestCategory_TieBreakLexicographic(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	cat, _ := newTestCategorizer(now)
	rows := []fixtureRow{
		{name: "left", categoryID: "alpha", theta: angleNear, ageBefore: time.Hour},
		{name: "right", categoryID: "beta", theta: angleNear, ageBefore: time.Hour},
	}
	todos, embs, live := buildFixture(now, rows, "alpha", "beta")

	got := cat.SuggestCategory(unit3(0), todos, embs, live)
	if got == nil {
		t.Fatalf("expected a suggestion")
	}
	if got.CategoryID != "alpha" {
		t.Fatalf("expected lexicographic winner 'alpha', got %q", got.CategoryID)
	}
}

func TestSuggestCategory_AcceptThresholdRejectsWeakWinner(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	cat, _ := newTestCategorizer(now)
	rows := []fixtureRow{
		{name: "single-mid", categoryID: "A", theta: angleMid, ageBefore: time.Hour},
	}
	todos, embs, live := buildFixture(now, rows, "A")

	got := cat.SuggestCategory(unit3(0), todos, embs, live)
	if got != nil {
		t.Fatalf("expected nil (score below threshold), got %+v", got)
	}
}

func TestSuggestCategory_DeletedCategoryFiltered(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	cat, _ := newTestCategorizer(now)
	rows := []fixtureRow{
		{name: "orphan", categoryID: "deleted", theta: angleNear, ageBefore: time.Hour},
	}
	todos, embs, _ := buildFixture(now, rows)
	live := map[string]struct{}{"other": {}}

	got := cat.SuggestCategory(unit3(0), todos, embs, live)
	if got != nil {
		t.Fatalf("expected nil when only category is deleted, got %+v", got)
	}
}

func TestSuggestCategory_MissingEmbeddingSkipped(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	cat, _ := newTestCategorizer(now)
	rows := []fixtureRow{
		{name: "has-embed", categoryID: "A", theta: angleNear, ageBefore: time.Hour},
		{name: "no-embed", categoryID: "A", theta: angleNear, ageBefore: time.Hour, noEmbed: true},
	}
	todos, embs, live := buildFixture(now, rows, "A")

	got := cat.SuggestCategory(unit3(0), todos, embs, live)
	if got == nil {
		t.Fatalf("expected a suggestion (one item still has an embedding)")
	}
	if got.N != 2 {
		t.Fatalf("expected N_c=2 (popularity counts every member), got %d", got.N)
	}
}

func TestSuggestCategory_EmptyInputs(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	cat, _ := newTestCategorizer(now)
	if got := cat.SuggestCategory(nil, nil, nil, nil); got != nil {
		t.Fatalf("expected nil for empty inputs, got %+v", got)
	}
	todos, embs, live := buildFixture(now, []fixtureRow{
		{name: "x", categoryID: "A", theta: angleNear, ageBefore: time.Hour},
	}, "A")
	if got := cat.SuggestCategory(unit3(0), nil, embs, live); got != nil {
		t.Fatalf("expected nil for empty todos, got %+v", got)
	}
	if got := cat.SuggestCategory(unit3(0), todos, nil, live); got != nil {
		t.Fatalf("expected nil for empty embeddings, got %+v", got)
	}
}

func TestSuggestCategory_DeterministicAcrossRuns(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	cat, _ := newTestCategorizer(now)
	rows := []fixtureRow{
		{name: "a1", categoryID: "alpha", theta: angleNear, ageBefore: time.Hour},
		{name: "a2", categoryID: "alpha", theta: angleNear, ageBefore: time.Hour},
		{name: "b1", categoryID: "beta", theta: angleNear, ageBefore: time.Hour},
		{name: "b2", categoryID: "beta", theta: angleNear, ageBefore: time.Hour},
	}
	todos, embs, live := buildFixture(now, rows, "alpha", "beta")

	var first *CategorySuggestion
	for i := 0; i < 25; i++ {
		got := cat.SuggestCategory(unit3(0), todos, embs, live)
		if got == nil {
			t.Fatalf("iteration %d: nil suggestion", i)
		}
		if first == nil {
			first = got
			continue
		}
		if got.CategoryID != first.CategoryID {
			t.Fatalf("non-deterministic winner: iter %d %q != %q", i, got.CategoryID, first.CategoryID)
		}
		if !approxEqual(got.Score, first.Score, 1e-7) {
			t.Fatalf("non-deterministic score: iter %d %v != %v", i, got.Score, first.Score)
		}
	}
}

func TestSuggestCategory_TwoPassBlendFormula(t *testing.T) {
	// Single category, mix of recent and ancient items.
	// Spec assertion: combined == RecentWeight*recentSum + (1-RecentWeight)*allSum.
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	cat, _ := newTestCategorizer(now)
	rows := []fixtureRow{
		{name: "recent-1", categoryID: "A", theta: angleNear, ageBefore: time.Hour},
		{name: "recent-2", categoryID: "A", theta: angleMid, ageBefore: 24 * time.Hour},
		{name: "ancient-1", categoryID: "A", theta: angleNear, ageBefore: 90 * 24 * time.Hour},
	}
	todos, embs, live := buildFixture(now, rows, "A")

	got := cat.SuggestCategory(unit3(0), todos, embs, live)
	if got == nil {
		t.Fatalf("expected a suggestion")
	}

	flNear := flooredSim(float32(math.Cos(angleNear)), 0.55)
	flMid := flooredSim(float32(math.Cos(angleMid)), 0.55)
	// Recent pass = recent-1 + recent-2.
	recentSum := flNear + flMid
	// All pass = recent-1 + recent-2 + ancient-1.
	allSum := flNear + flMid + flNear
	expectedBlended := blend(recentSum, allSum, 0.70)
	expectedFinal := expectedBlended * popularity(3, 0.30)

	if !approxEqual(got.BlendedRecent, recentSum, 1e-5) {
		t.Fatalf("BlendedRecent = %v, want %v", got.BlendedRecent, recentSum)
	}
	if !approxEqual(got.BlendedAll, allSum, 1e-5) {
		t.Fatalf("BlendedAll = %v, want %v", got.BlendedAll, allSum)
	}
	if !approxEqual(got.Score, expectedFinal, 1e-5) {
		t.Fatalf("Score = %v, want %v (independently recomputed)", got.Score, expectedFinal)
	}
}
