package main

import (
	"path/filepath"
	"testing"
)

func TestValidateCachePath(t *testing.T) {
	tmp := t.TempDir()

	t.Run("inside_data_dir", func(t *testing.T) {
		got, err := validateCachePath(filepath.Join(tmp, "embeddings.jsonl"), tmp)
		if err != nil {
			t.Fatalf("expected accept, got error: %v", err)
		}
		want, _ := filepath.Abs(filepath.Join(tmp, "embeddings.jsonl"))
		if got != want {
			t.Fatalf("resolved path = %q, want %q", got, want)
		}
	})

	t.Run("nested_subdirectory_accepted", func(t *testing.T) {
		if _, err := validateCachePath(filepath.Join(tmp, "subdir", "cache.jsonl"), tmp); err != nil {
			t.Fatalf("expected accept of nested path, got %v", err)
		}
	})

	t.Run("escapes_via_dotdot", func(t *testing.T) {
		if _, err := validateCachePath(filepath.Join(tmp, "..", "evil.jsonl"), tmp); err == nil {
			t.Fatalf("expected rejection of escaped path")
		}
	})

	t.Run("absolute_outside_data_dir", func(t *testing.T) {
		other := t.TempDir()
		if _, err := validateCachePath(filepath.Join(other, "cache.jsonl"), tmp); err == nil {
			t.Fatalf("expected rejection of absolute path outside data dir")
		}
	})

	t.Run("prefix_collision_not_accepted", func(t *testing.T) {
		// /tmp/X must not be accepted as inside /tmp/X-sibling.
		if _, err := validateCachePath(tmp+"foo/cache.jsonl", tmp); err == nil {
			t.Fatalf("expected rejection of prefix-collision path")
		}
	})
}
