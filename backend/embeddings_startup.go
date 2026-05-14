package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// embeddingBuilderConfig is the subset of Config that BuildEmbeddingCache needs.
// Kept as its own type so tests can construct one without env parsing.
//
// Model is stamped on each cached entry's provenance.
type embeddingBuilderConfig struct {
	Model     string
	BatchSize int
}

// BuildEmbeddingCache collects every unique normalized todo name from the
// server's projected state, finds those without a cached embedding, and
// fetches them from the injected Embedder in rate-limited batches. Each
// successful embedding is appended to the cache (in-memory + JSONL on
// disk). The call blocks until all uncached items have been processed (or
// a hard error).
//
// The Embedder is injected (rather than constructed inside) so the same
// rate-limited client can be reused by the runtime auto-categorize hook;
// otherwise startup and runtime would maintain two independent RPM
// buckets and double the effective quota cost.
//
// On per-batch failures (after retries), the affected items are skipped and
// will be retried on the next startup; this is logged but not fatal.
func BuildEmbeddingCache(ctx context.Context, cfg embeddingBuilderConfig, client Embedder, server *Server, cache *EmbeddingCache) error {
	if cache == nil {
		return fmt.Errorf("embedding cache is nil")
	}
	if client == nil {
		return fmt.Errorf("embedder is nil")
	}

	start := time.Now()
	allNames := server.state.GetAllNormalizedNames()
	cachedBefore := cache.Len()

	missing := make([]string, 0)
	for _, n := range allNames {
		if !cache.Has(n) {
			missing = append(missing, n)
		}
	}

	if len(missing) == 0 {
		slog.Info("embedding cache up to date",
			"total_items", len(allNames),
			"cached", cachedBefore,
			"newly_embedded", 0,
			"failed", 0,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return nil
	}

	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}
	if batchSize > 100 {
		batchSize = 100
	}
	var added, failed int

	for i := 0; i < len(missing); i += batchSize {
		end := i + batchSize
		if end > len(missing) {
			end = len(missing)
		}
		batch := missing[i:end]

		vectors, err := client.EmbedBatch(ctx, batch)
		if err != nil {
			// Generic log only — the underlying error message may include
			// upstream status codes but never the API key or response body.
			slog.Error("embedding batch failed; items will be retried next startup",
				"batch_size", len(batch),
				"error", err,
			)
			failed += len(batch)
			continue
		}

		now := time.Now().UTC()
		for j, text := range batch {
			vec := vectors[j]
			entry := CachedEmbedding{
				Key:       text,
				Text:      text,
				Model:     cfg.Model,
				Dim:       len(vec),
				Vector:    vec,
				CreatedAt: now,
			}
			if err := cache.Add(entry); err != nil {
				slog.Error("failed to persist embedding entry",
					"error", err,
				)
				failed++
				continue
			}
			added++
		}
	}

	slog.Info("embedding cache build complete",
		"total_items", len(allNames),
		"cached_before", cachedBefore,
		"newly_embedded", added,
		"failed", failed,
		"duration_ms", time.Since(start).Milliseconds(),
	)
	return nil
}
