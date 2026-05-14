package main

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"golang.org/x/sync/singleflight"
)

// autoCategorizeTimeout caps how long a single suggestion may take,
// including the embedding round-trip. Generous because Gemini batchEmbed
// can be ~1s under load, and we never block any user-visible request on it.
const autoCategorizeTimeout = 10 * time.Second

// errAutoCategorizeMissingDeps is returned from suggestCategory when a
// dependency is nil. Callers should treat this as a silent no-op (it just
// means the feature is disabled).
var errAutoCategorizeMissingDeps = errors.New("auto-categorize dependencies not configured")

// maybeStartAutoCategorize fires a fire-and-forget suggestion goroutine when
// the given event is a TodoCreated with no category. MUST be called by the
// caller AFTER the TodoCreated has been published on s.broadcast, so any
// follow-up TodoCategorized always arrives strictly after on the wire.
//
// No-op when any dependency is missing (feature disabled, or partially
// wired in tests).
func (s *Server) maybeStartAutoCategorize(event Event) {
	created, ok := event.(TodoCreated)
	if !ok {
		return
	}
	if created.CategoryID != nil {
		return
	}
	if s.embeddingCache == nil || s.embeddingClient == nil ||
		s.categorizer == nil || s.suggestFlight == nil ||
		s.autoCategorizeMetrics == nil {
		return
	}
	go s.suggestCategoryAsync(created.ID, created.Name)
}

// suggestCategoryAsync wraps suggestCategory with the metrics increment and
// generic error logging. Errors are intentionally swallowed (logged only)
// because the user-visible TodoCreated has already been broadcast — we
// must never bubble auto-categorize failures back to the user.
func (s *Server) suggestCategoryAsync(todoID, name string) {
	s.autoCategorizeMetrics.Attempted.Add(1)
	if err := s.suggestCategory(todoID, name); err != nil {
		// Errors are already logged at the right granularity inside
		// suggestCategory. This branch only catches programming errors.
		slog.Debug("auto_categorize_unexpected_error", "todo_id", todoID, "error", err)
	}
}

func (s *Server) suggestCategory(todoID, name string) error {
	if s.embeddingCache == nil || s.embeddingClient == nil ||
		s.categorizer == nil || s.suggestFlight == nil ||
		s.autoCategorizeMetrics == nil {
		return errAutoCategorizeMissingDeps
	}

	ctx, cancel := context.WithTimeout(context.Background(), autoCategorizeTimeout)
	defer cancel()

	start := time.Now()
	key := normalizeName(name)
	if key == "" {
		// Empty key after normalization (e.g. an emoji-only name). Nothing
		// to do; CreateTodo validation should reject these earlier anyway.
		slog.Debug("auto_categorize_skipped_empty_key", "todo_id", todoID)
		return nil
	}

	// Log entry without leaking the raw item name. key_len is a coarse
	// hint for debugging only.
	slog.Info("auto_categorize_started",
		"todo_id", todoID,
		"key_len", len(key),
		"cache_hit_expected", s.embeddingCache.Has(key),
	)

	vec, cacheHit, err := s.resolveEmbedding(ctx, key)
	if err != nil {
		s.autoCategorizeMetrics.EmbedFailed.Add(1)
		slog.Error("auto_categorize_embed_failed",
			"todo_id", todoID,
			"error", err.Error(),
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return nil
	}
	if !cacheHit {
		s.autoCategorizeMetrics.EmbedMiss.Add(1)
	}

	// Snapshot the inputs needed by the pure scorer.
	todos, embeddings, liveCats := s.snapshotForCategorize()

	suggestion := s.categorizer.SuggestCategory(vec, todos, embeddings, liveCats)
	if suggestion == nil {
		s.recordRejection(todoID, cacheHit, start)
		return nil
	}

	// Precondition re-check: the todo must still exist and still be
	// uncategorized. This prevents the auto-categorize from clobbering a
	// user choice made between TodoCreated and now.
	cur, ok := s.state.GetTodo(todoID)
	if !ok {
		s.autoCategorizeMetrics.SkippedDeleted.Add(1)
		slog.Info("auto_categorize_skipped_deleted",
			"todo_id", todoID,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return nil
	}
	if cur.CategoryID != nil {
		s.autoCategorizeMetrics.SkippedUserSet.Add(1)
		slog.Info("auto_categorize_skipped_user_set",
			"todo_id", todoID,
			"existing_category_id", *cur.CategoryID,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return nil
	}

	cid := suggestion.CategoryID
	cmd := CategorizeTodoCommand{
		BaseCommand: BaseCommand{
			Type:      "CategorizeTodo",
			CommandID: "auto-" + todoID,
		},
		ID:         todoID,
		CategoryID: &cid,
	}

	if err := s.ExecuteCommand(cmd); err != nil {
		s.autoCategorizeMetrics.EmitFailed.Add(1)
		slog.Error("auto_categorize_emit_failed",
			"todo_id", todoID,
			"category_id", cid,
			"error", err.Error(),
		)
		return nil
	}

	s.autoCategorizeMetrics.Suggested.Add(1)
	slog.Info("auto_categorize_suggested",
		"todo_id", todoID,
		"category_id", cid,
		"score", suggestion.Score,
		"blended_recent", suggestion.BlendedRecent,
		"blended_all", suggestion.BlendedAll,
		"popularity_factor", suggestion.Popularity,
		"n_c", suggestion.N,
		"max_sim", suggestion.MaxSim,
		"candidates", suggestion.Candidates,
		"cache_hit", cacheHit,
		"duration_ms", time.Since(start).Milliseconds(),
	)
	return nil
}

// resolveEmbedding returns the vector for key, fetching it via the embedder
// on miss and persisting to the cache. Concurrent callers for the same key
// share one upstream fetch via singleflight.
func (s *Server) resolveEmbedding(ctx context.Context, key string) (vec []float32, cacheHit bool, err error) {
	if entry, ok := s.embeddingCache.Get(key); ok {
		return entry.Vector, true, nil
	}

	// singleflight returns (value, err, shared). `shared=true` means more
	// than one caller waited on this flight: count those as deduped.
	v, err, shared := s.suggestFlight.Do(key, func() (any, error) {
		// Double-check inside the flight: a concurrent flight may have
		// already populated the cache by the time we run.
		if entry, ok := s.embeddingCache.Get(key); ok {
			return entry.Vector, nil
		}
		vectors, err := s.embeddingClient.EmbedBatch(ctx, []string{key})
		if err != nil {
			return nil, err
		}
		if len(vectors) != 1 || len(vectors[0]) == 0 {
			return nil, errors.New("embedder returned empty vector")
		}
		entry := CachedEmbedding{
			Key:    key,
			Text:   key,
			Model:  s.categorizerModelHint(),
			Dim:    len(vectors[0]),
			Vector: vectors[0],
		}
		if addErr := s.embeddingCache.Add(entry); addErr != nil {
			// Cache write failure is non-fatal: we still have the vector
			// and can return it for this round. Future requests will
			// retry the embed.
			slog.Warn("auto_categorize_cache_write_failed", "error", addErr.Error())
		}
		return vectors[0], nil
	})
	if shared {
		s.autoCategorizeMetrics.Deduped.Add(1)
		slog.Info("auto_categorize_dedup", "key_len", len(key))
	}
	if err != nil {
		return nil, false, err
	}
	out, ok := v.([]float32)
	if !ok {
		return nil, false, errors.New("singleflight returned wrong type")
	}
	return out, false, nil
}

// categorizerModelHint returns the model name to stamp on freshly-cached
// embeddings. Pulled from the embedding client when available; falls back
// to "unknown" when a stub is wired (tests).
func (s *Server) categorizerModelHint() string {
	if real, ok := s.embeddingClient.(*EmbeddingClient); ok {
		return real.model
	}
	return "unknown"
}

// snapshotForCategorize copies the data the scorer needs while holding
// only short-lived read locks. The pure SuggestCategory then runs without
// touching State or the cache again.
func (s *Server) snapshotForCategorize() (todos []Todo, embs map[string]CachedEmbedding, live map[string]struct{}) {
	todos = s.state.GetTodos()
	embs = s.embeddingCache.All()
	cats := s.state.GetCategories()
	live = make(map[string]struct{}, len(cats))
	for _, c := range cats {
		live[c.ID] = struct{}{}
	}
	return todos, embs, live
}

// recordRejection updates metrics and emits a structured log when
// SuggestCategory returns nil. We cannot tell from the nil return value
// whether the cause was threshold, gate, or no candidates, so we recompute
// a coarse reason from the snapshot. This is intentionally cheap and
// observational.
func (s *Server) recordRejection(todoID string, cacheHit bool, start time.Time) {
	// We don't have direct visibility into which gate failed without
	// running the algorithm again, so report a generic "no suggestion".
	// Operators can correlate counts (RejectedThreshold + RejectedGate +
	// RejectedNoSignal) over time via the categorizer's own debug logs in
	// the future; for now keep the field present but coarse.
	s.autoCategorizeMetrics.RejectedNoSignal.Add(1)
	slog.Info("auto_categorize_no_suggestion",
		"todo_id", todoID,
		"cache_hit", cacheHit,
		"duration_ms", time.Since(start).Milliseconds(),
	)
}

// Compile-time check: keep singleflight referenced so the import survives
// even if a future refactor temporarily stops using it directly here.
var _ = singleflight.Group{}
