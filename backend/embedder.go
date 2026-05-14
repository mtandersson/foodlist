package main

import "context"

// Embedder is the minimal surface the auto-categorize hook needs from an
// embedding provider. Declared as an interface here (rather than always
// requiring *EmbeddingClient) so tests can substitute a stub that returns
// deterministic vectors without any network access.
type Embedder interface {
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}

// Compile-time assertion that the real Gemini-backed client satisfies the
// interface. If the EmbeddingClient method set drifts, the build breaks here.
var _ Embedder = (*EmbeddingClient)(nil)
