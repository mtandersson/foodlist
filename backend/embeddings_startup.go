package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// embeddingBuilderConfig is the subset of Config that BuildEmbeddingCache needs.
// Kept as its own type so tests can construct one without env parsing.
type embeddingBuilderConfig struct {
	Model     string
	APIKey    string
	BatchSize int
	RPM       int
}

// BuildEmbeddingCache collects every unique normalized todo name from the
// server's projected state, finds those without a cached embedding, and
// fetches them from the Gemini API in rate-limited batches. Each successful
// embedding is appended to the cache (in-memory + JSONL on disk). The call
// blocks until all uncached items have been processed (or a hard error).
//
// On per-batch failures (after retries), the affected items are skipped and
// will be retried on the next startup; this is logged but not fatal.
func BuildEmbeddingCache(ctx context.Context, cfg embeddingBuilderConfig, server *Server, cache *EmbeddingCache) error {
	if cache == nil {
		return fmt.Errorf("embedding cache is nil")
	}
	if cfg.APIKey == "" {
		return fmt.Errorf("gemini api key is empty")
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

	client := NewEmbeddingClient(cfg.APIKey, cfg.Model, cfg.BatchSize, cfg.RPM)
	defer client.Close()

	batchSize := client.BatchSize()
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
