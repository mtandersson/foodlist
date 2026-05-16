package main

import (
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Default tuning constants for the SuggestionEngine. Mirrored as env-var
// defaults in main.go.
const (
	DefaultSuggestionsMinPurchases    = 3
	DefaultSuggestionsMaxIntervalDays = 90
	DefaultSuggestionsDueFraction     = float32(0.667)
	DefaultSuggestionsDedupSimilarity = float32(0.85)
	DefaultSuggestionsRecentLimit     = 6
	DefaultSuggestionsRecomputeHours  = 6
)

// suggestionNamespace is the deterministic UUID v5 namespace used to derive
// stable suggestion IDs from canonical normalized names. Generated once
// and pinned here so IDs stay stable across restarts and clients.
var suggestionNamespace = uuid.MustParse("8b1c4a0a-1f6e-4f2a-9b1a-2e1e7d5c0b21")

// SuggestionEngineConfig holds all tunables for the engine. Zero values
// fall back to the documented defaults via NewSuggestionEngine.
type SuggestionEngineConfig struct {
	MinPurchases     int
	MaxInterval      time.Duration
	DueFraction      float32
	DedupSimilarity  float32
	RecentLimit      int
	RecomputeEvery   time.Duration
	Now              func() time.Time
}

// SuggestionEngine maintains the in-memory map of active grocery suggestions
// and exposes diffing for delta broadcast. It is safe for concurrent use.
//
// All recomputes are pure (in: snapshot of todos + embeddings + categories;
// out: new map). The engine itself only stores the latest snapshot and the
// config; it does not touch the EventStore or the Server.
type SuggestionEngine struct {
	cfg SuggestionEngineConfig

	mu       sync.RWMutex
	current  map[string]Suggestion
}

// NewSuggestionEngine constructs an engine with the given config. Missing
// fields are populated with the documented defaults.
func NewSuggestionEngine(cfg SuggestionEngineConfig) *SuggestionEngine {
	if cfg.MinPurchases <= 0 {
		cfg.MinPurchases = DefaultSuggestionsMinPurchases
	}
	if cfg.MaxInterval <= 0 {
		cfg.MaxInterval = time.Duration(DefaultSuggestionsMaxIntervalDays) * 24 * time.Hour
	}
	if cfg.DueFraction <= 0 {
		cfg.DueFraction = DefaultSuggestionsDueFraction
	}
	if cfg.DedupSimilarity <= 0 {
		cfg.DedupSimilarity = DefaultSuggestionsDedupSimilarity
	}
	if cfg.RecentLimit <= 0 {
		cfg.RecentLimit = DefaultSuggestionsRecentLimit
	}
	if cfg.RecomputeEvery <= 0 {
		cfg.RecomputeEvery = time.Duration(DefaultSuggestionsRecomputeHours) * time.Hour
	}
	if cfg.Now == nil {
		cfg.Now = func() time.Time { return time.Now().UTC() }
	}
	return &SuggestionEngine{
		cfg:     cfg,
		current: make(map[string]Suggestion),
	}
}

// Config returns the active configuration (copy).
func (e *SuggestionEngine) Config() SuggestionEngineConfig { return e.cfg }

// Snapshot returns a sorted copy of the current suggestions (frequent first,
// shorter interval first, then name). Safe for concurrent calls.
func (e *SuggestionEngine) Snapshot() []Suggestion {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]Suggestion, 0, len(e.current))
	for _, s := range e.current {
		out = append(out, s)
	}
	sortSuggestions(out)
	return out
}

// Get returns the current suggestion for the given ID, if present.
func (e *SuggestionEngine) Get(id string) (Suggestion, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	s, ok := e.current[id]
	return s, ok
}

// MarkPurchased optimistically removes any suggestion that matches the
// given freshly-created todo name. The match is heuristic:
//
//  1. First the direct UUID-v5 derived from normalizeName(name).
//  2. If that misses, scan current suggestions and match by exact
//     normalized name on the suggestion's display name. This catches the
//     case where the user added a synonym ("lättmjölk") whose cluster
//     canonical name is something different ("mjölk").
//
// Returns the removed suggestion ID and true if a row was evicted. The
// next full recompute reconciles canonical state, so any miss here is
// merely a UI snappiness regression, not a correctness bug.
func (e *SuggestionEngine) MarkPurchased(name string) (string, bool) {
	norm := normalizeName(name)
	if norm == "" {
		return "", false
	}
	id := suggestionIDFor(name)

	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.current[id]; ok {
		delete(e.current, id)
		return id, true
	}
	// Fallback: look for an existing suggestion whose display name (after
	// normalization) matches the added item. This is O(n) over the current
	// suggestion set which is small (typically << 100).
	for existingID, s := range e.current {
		if normalizeName(s.Name) == norm {
			delete(e.current, existingID)
			return existingID, true
		}
	}
	return "", false
}

// Recompute rebuilds the suggestion set from scratch using the provided
// snapshot of every todo, the embedding cache, and the set of categories.
// Returns the lists of added/removed suggestions versus the previously
// stored set, and atomically swaps in the new state.
func (e *SuggestionEngine) Recompute(
	todos []Todo,
	embeddings map[string]CachedEmbedding,
	categories []Category,
) (added []Suggestion, removed []string) {
	fresh := e.computeSuggestions(todos, embeddings, categories)

	e.mu.Lock()
	defer e.mu.Unlock()
	added, removed = diffSuggestions(e.current, fresh)
	e.current = fresh
	return added, removed
}

// computeSuggestions runs the pure algorithm without touching mutable state.
// Exposed (lowercase) as a method only so tests can exercise it without a
// lock, but the result is what Recompute swaps in.
func (e *SuggestionEngine) computeSuggestions(
	todos []Todo,
	embeddings map[string]CachedEmbedding,
	categories []Category,
) map[string]Suggestion {
	now := e.cfg.Now()
	catName := make(map[string]string, len(categories))
	for _, c := range categories {
		catName[c.ID] = c.Name
	}

	// 1) Group completed todos by normalized name.
	type rawGroup struct {
		name          string
		normalized    string
		completedAt   []time.Time
		categoryVotes map[string]int
	}
	groups := make(map[string]*rawGroup)
	for _, t := range todos {
		if t.CompletedAt == nil {
			continue
		}
		norm := normalizeName(t.Name)
		if norm == "" {
			continue
		}
		g, ok := groups[norm]
		if !ok {
			g = &rawGroup{
				name:          t.Name,
				normalized:    norm,
				categoryVotes: make(map[string]int),
			}
			groups[norm] = g
		}
		g.completedAt = append(g.completedAt, *t.CompletedAt)
		if t.CategoryID != nil {
			g.categoryVotes[*t.CategoryID]++
		}
	}

	// 2) Cluster groups by embedding similarity (>= DedupSimilarity). The
	// canonical representative of a cluster is the group with the most
	// completions (ties broken alphabetically on normalized name for
	// determinism).
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	type cluster struct {
		canonical    string
		memberKeys   []string
		totalCount   int
		allCompleted []time.Time
		categoryVotes map[string]int
		canonicalName string
	}
	parent := make(map[string]string, len(keys))
	for _, k := range keys {
		parent[k] = k
	}
	var find func(string) string
	find = func(x string) string {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}
	union := func(a, b string) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[ra] = rb
		}
	}
	// O(n^2) is fine; suggestion grouping is small. Skip pairs without an
	// embedding (we can't compare them, so they stay separate).
	for i := 0; i < len(keys); i++ {
		ei, ok := embeddings[keys[i]]
		if !ok {
			continue
		}
		for j := i + 1; j < len(keys); j++ {
			ej, ok := embeddings[keys[j]]
			if !ok {
				continue
			}
			if CosineSimilarity(ei.Vector, ej.Vector) >= e.cfg.DedupSimilarity {
				union(keys[i], keys[j])
			}
		}
	}
	clusters := make(map[string]*cluster)
	for _, k := range keys {
		root := find(k)
		c, ok := clusters[root]
		if !ok {
			c = &cluster{categoryVotes: make(map[string]int)}
			clusters[root] = c
		}
		g := groups[k]
		c.memberKeys = append(c.memberKeys, k)
		c.totalCount += len(g.completedAt)
		c.allCompleted = append(c.allCompleted, g.completedAt...)
		for cat, n := range g.categoryVotes {
			c.categoryVotes[cat] += n
		}
	}
	// Resolve canonical (most-completed) member per cluster.
	for _, c := range clusters {
		// Determine which member to use as representative.
		bestKey := c.memberKeys[0]
		bestCount := len(groups[bestKey].completedAt)
		for _, k := range c.memberKeys[1:] {
			n := len(groups[k].completedAt)
			if n > bestCount || (n == bestCount && k < bestKey) {
				bestKey = k
				bestCount = n
			}
		}
		c.canonical = bestKey
		c.canonicalName = groups[bestKey].name
	}

	// 3-5) Filter and build suggestions.
	out := make(map[string]Suggestion)
	maxIntervalSec := e.cfg.MaxInterval.Seconds()
	for _, c := range clusters {
		if c.totalCount < e.cfg.MinPurchases {
			continue
		}
		sort.Slice(c.allCompleted, func(i, j int) bool {
			return c.allCompleted[i].Before(c.allCompleted[j])
		})
		recent := c.allCompleted
		if len(recent) > e.cfg.RecentLimit {
			recent = recent[len(recent)-e.cfg.RecentLimit:]
		}
		if len(recent) < 2 {
			// Need at least two completions to compute an interval. With
			// MinPurchases=3 this is normally fine, but guard anyway.
			continue
		}
		var totalDelta float64
		for i := 1; i < len(recent); i++ {
			totalDelta += recent[i].Sub(recent[i-1]).Seconds()
		}
		avgIntervalSec := totalDelta / float64(len(recent)-1)
		if avgIntervalSec <= 0 || avgIntervalSec > maxIntervalSec {
			continue
		}
		lastPurchased := recent[len(recent)-1]
		sinceLast := now.Sub(lastPurchased).Seconds()
		if sinceLast < float64(e.cfg.DueFraction)*avgIntervalSec {
			continue
		}

		// Resolve a preferred category from votes (most votes wins, ties
		// broken by lexicographic ID for determinism). Skip categories that
		// no longer exist.
		var bestCat *string
		var bestCatName *string
		if len(c.categoryVotes) > 0 {
			ids := make([]string, 0, len(c.categoryVotes))
			for id := range c.categoryVotes {
				if _, ok := catName[id]; !ok {
					continue
				}
				ids = append(ids, id)
			}
			sort.Strings(ids)
			best := -1
			for _, id := range ids {
				if c.categoryVotes[id] > best {
					best = c.categoryVotes[id]
					cid := id
					bestCat = &cid
					nm := catName[id]
					bestCatName = &nm
				}
			}
		}

		id := suggestionIDFor(c.canonical)
		out[id] = Suggestion{
			ID:                 id,
			Name:               c.canonicalName,
			CategoryID:         bestCat,
			CategoryName:       bestCatName,
			LastPurchasedAt:    lastPurchased.UTC(),
			PurchaseCount:      c.totalCount,
			AvgIntervalSeconds: avgIntervalSec,
		}
	}

	// 6) Dedup against active todos. Skip suggestions whose canonical
	// embedding is highly similar to any active todo's embedding. Exact
	// normalized-name matches are skipped even without embeddings.
	activeNormalized := make(map[string]struct{})
	activeEmbeddings := make([][]float32, 0)
	for _, t := range todos {
		if t.CompletedAt != nil {
			continue
		}
		n := normalizeName(t.Name)
		if n == "" {
			continue
		}
		activeNormalized[n] = struct{}{}
		if entry, ok := embeddings[n]; ok {
			activeEmbeddings = append(activeEmbeddings, entry.Vector)
		}
	}
	for id, s := range out {
		canonNorm := normalizeName(s.Name)
		if _, ok := activeNormalized[canonNorm]; ok {
			delete(out, id)
			continue
		}
		entry, ok := embeddings[canonNorm]
		if !ok {
			continue
		}
		for _, av := range activeEmbeddings {
			if CosineSimilarity(entry.Vector, av) >= e.cfg.DedupSimilarity {
				delete(out, id)
				break
			}
		}
	}

	return out
}

// sortSuggestions orders suggestions for stable, user-friendly display.
// Highest frequency first (purchase count desc), then shortest interval
// (more urgent), then name for determinism.
func sortSuggestions(s []Suggestion) {
	sort.SliceStable(s, func(i, j int) bool {
		if s[i].PurchaseCount != s[j].PurchaseCount {
			return s[i].PurchaseCount > s[j].PurchaseCount
		}
		if s[i].AvgIntervalSeconds != s[j].AvgIntervalSeconds {
			return s[i].AvgIntervalSeconds < s[j].AvgIntervalSeconds
		}
		return s[i].Name < s[j].Name
	})
}

// diffSuggestions returns add/remove lists between old and new states. Added
// suggestions are returned sorted (for deterministic broadcasts); removed
// IDs are returned sorted lexicographically.
func diffSuggestions(oldSet, newSet map[string]Suggestion) (added []Suggestion, removed []string) {
	for id, s := range newSet {
		prev, ok := oldSet[id]
		if !ok || !sameSuggestion(prev, s) {
			added = append(added, s)
		}
	}
	for id := range oldSet {
		if _, ok := newSet[id]; !ok {
			removed = append(removed, id)
		}
	}
	sortSuggestions(added)
	sort.Strings(removed)
	return added, removed
}

// sameSuggestion compares the user-visible fields. Times are compared at
// second granularity (matches our JSON encoding) to avoid noisy diffs when
// a recompute reads the same completedAt twice.
func sameSuggestion(a, b Suggestion) bool {
	if a.ID != b.ID || a.Name != b.Name || a.PurchaseCount != b.PurchaseCount {
		return false
	}
	if !a.LastPurchasedAt.Equal(b.LastPurchasedAt) {
		return false
	}
	if a.AvgIntervalSeconds != b.AvgIntervalSeconds {
		return false
	}
	if strPtrEq(a.CategoryID, b.CategoryID) && strPtrEq(a.CategoryName, b.CategoryName) {
		return true
	}
	return false
}

func strPtrEq(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// suggestionIDFor returns the deterministic UUID v5 for a normalized name.
// Empty input returns an empty string.
func suggestionIDFor(name string) string {
	n := normalizeName(name)
	if n == "" {
		return ""
	}
	return uuid.NewSHA1(suggestionNamespace, []byte(n)).String()
}
