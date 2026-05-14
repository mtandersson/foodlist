package main

import (
	"sort"
	"time"
)

// Default tuning constants. Exposed as package-level vars so main.go can
// reference them when env vars are unset.
const (
	DefaultSimilarityFloor     float32 = 0.55
	DefaultRecencyWindow               = 30 * 24 * time.Hour
	DefaultRecentWeight        float32 = 0.70
	DefaultPopularityWeight    float32 = 0.30
	DefaultMaxSimGate          float32 = 0.20
	DefaultAcceptanceThreshold float32 = 0.30
)

// Categorizer is the pure scorer that picks a category for a freshly-created
// todo given its embedding and the existing categorized items. It performs
// no I/O — all inputs are passed in as plain data so the type is trivially
// unit-testable.
//
// Algorithm (see plan):
//
//	floored(sim)    = max(0, sim - SimilarityFloor)
//	catScore(S)     = Σ floored(sim) over items in S
//	popularity(N_c) = 1 + PopularityWeight * log(1 + N_c)
//	maxSim(S)       = max floored(sim) over items in S
//	blended[c]      = RecentWeight * passRecent[c] + (1-RecentWeight) * passAll[c]
//	final[c]        = blended[c] * popularity(N_c) if eligible(c) else 0
//	eligible(c)     = blended[c] > 0 AND maxSim(allItems_c) >= MaxSimGate
//
// The winner is argmax over final[c]; ties (within 1e-9) break by ascending
// category ID. A suggestion is returned only if its score >= AcceptanceThreshold.
type Categorizer struct {
	SimilarityFloor     float32
	RecencyWindow       time.Duration
	RecentWeight        float32
	PopularityWeight    float32
	MaxSimGate          float32
	AcceptanceThreshold float32
	Now                 func() time.Time
}

// NewCategorizer returns a Categorizer with the documented defaults filled
// in for any zero-valued field. The returned value is safe to copy.
func NewCategorizer(c Categorizer) Categorizer {
	if c.SimilarityFloor <= 0 {
		c.SimilarityFloor = DefaultSimilarityFloor
	}
	if c.RecencyWindow <= 0 {
		c.RecencyWindow = DefaultRecencyWindow
	}
	if c.RecentWeight <= 0 {
		c.RecentWeight = DefaultRecentWeight
	}
	if c.PopularityWeight < 0 {
		c.PopularityWeight = DefaultPopularityWeight
	}
	if c.MaxSimGate <= 0 {
		c.MaxSimGate = DefaultMaxSimGate
	}
	if c.AcceptanceThreshold <= 0 {
		c.AcceptanceThreshold = DefaultAcceptanceThreshold
	}
	if c.Now == nil {
		c.Now = func() time.Time { return time.Now().UTC() }
	}
	return c
}

// CategorySuggestion is the result of SuggestCategory when one is found.
// Score is the final post-popularity score.
type CategorySuggestion struct {
	CategoryID    string
	Score         float32
	BlendedRecent float32
	BlendedAll    float32
	Popularity    float32
	N             int
	MaxSim        float32
	Candidates    int // number of categories that had at least one floored>0 item
}

// SuggestCategory returns a category suggestion for the given embedding
// vector, or nil if no category clears AcceptanceThreshold + MaxSimGate.
//
//   - todos: every todo with a CategoryID (active + completed). Items with
//     nil CategoryID are silently skipped. Items whose CategoryID is not in
//     liveCategoryIDs are also skipped (covers deleted categories).
//   - embeddings: keyed by normalizeName(todo.Name). Items missing an
//     embedding are silently skipped (cache may be lagging).
//   - liveCategoryIDs: set of currently-valid category IDs.
//
// This is a pure function; it takes no locks and performs no I/O.
func (c Categorizer) SuggestCategory(
	vec []float32,
	todos []Todo,
	embeddings map[string]CachedEmbedding,
	liveCategoryIDs map[string]struct{},
) *CategorySuggestion {
	if len(vec) == 0 || len(todos) == 0 || len(embeddings) == 0 {
		return nil
	}

	now := c.Now()
	windowStart := now.Add(-c.RecencyWindow)

	// Per-category accumulators. We don't iterate on the map for argmax;
	// we sort the keys for deterministic tie-break.
	type catAcc struct {
		recentFloored []float32
		allFloored    []float32
		maxFloored    float32
		total         int // N_c (every member, regardless of similarity)
	}
	acc := make(map[string]*catAcc)

	for _, t := range todos {
		if t.CategoryID == nil {
			continue
		}
		cid := *t.CategoryID
		if _, ok := liveCategoryIDs[cid]; !ok {
			continue
		}
		// Count every member toward N_c even if its embedding is missing —
		// the popularity signal is "how loaded is this category", which
		// doesn't depend on cache freshness.
		a, ok := acc[cid]
		if !ok {
			a = &catAcc{}
			acc[cid] = a
		}
		a.total++

		entry, ok := embeddings[normalizeName(t.Name)]
		if !ok {
			continue
		}
		sim := CosineSimilarity(vec, entry.Vector)
		fl := flooredSim(sim, c.SimilarityFloor)

		a.allFloored = append(a.allFloored, fl)
		if !t.CreatedAt.Before(windowStart) {
			a.recentFloored = append(a.recentFloored, fl)
		}
		if fl > a.maxFloored {
			a.maxFloored = fl
		}
	}

	if len(acc) == 0 {
		return nil
	}

	// Sorted iteration over category IDs makes the final argmax tie-break
	// deterministic.
	ids := make([]string, 0, len(acc))
	for id := range acc {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var best *CategorySuggestion
	candidates := 0
	const tieEps = 1e-9
	for _, id := range ids {
		a := acc[id]
		recentSum := sumFlooredSlice(a.recentFloored)
		allSum := sumFlooredSlice(a.allFloored)
		blended := blend(recentSum, allSum, c.RecentWeight)
		if blended > 0 {
			candidates++
		}

		// Eligibility gates.
		if blended <= 0 {
			continue
		}
		if a.maxFloored < c.MaxSimGate {
			continue
		}

		pop := popularity(a.total, c.PopularityWeight)
		final := blended * pop

		// Strict-greater because IDs are sorted ascending; on a true tie
		// (within epsilon) we keep the earlier (smaller) ID, which is the
		// documented contract.
		if best == nil || float64(final)-float64(best.Score) > tieEps {
			best = &CategorySuggestion{
				CategoryID:    id,
				Score:         final,
				BlendedRecent: recentSum,
				BlendedAll:    allSum,
				Popularity:    pop,
				N:             a.total,
				MaxSim:        a.maxFloored,
			}
		}
	}

	if best == nil {
		return nil
	}
	best.Candidates = candidates
	if best.Score < c.AcceptanceThreshold {
		return nil
	}
	return best
}

// sumFlooredSlice sums a slice of already-floored values. Kept separate
// from sumFloored so callers can amortize the floor computation when
// caching per-item floored similarities (as SuggestCategory does).
func sumFlooredSlice(vals []float32) float32 {
	var sum float32
	for _, v := range vals {
		sum += v
	}
	return sum
}
