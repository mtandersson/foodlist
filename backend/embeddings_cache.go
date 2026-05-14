package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// CachedEmbedding represents a single persisted embedding entry.
type CachedEmbedding struct {
	Key       string    `json:"key"`
	Text      string    `json:"text"`
	Model     string    `json:"model"`
	Dim       int       `json:"dim"`
	Vector    []float32 `json:"vector"`
	CreatedAt time.Time `json:"createdAt"`
}

// EmbeddingCache is an in-memory cache of embeddings backed by an append-only
// JSONL file on disk. The lookup key is the normalized text (output of
// normalizeName) and the value is the embedding plus provenance metadata.
//
// Concurrency: all public methods are safe for concurrent use. Add takes a
// write lock and writes one JSONL line then updates the in-memory map under
// the same lock so memory and disk stay consistent.
type EmbeddingCache struct {
	mu      sync.RWMutex
	entries map[string]CachedEmbedding
	file    *os.File
	path    string
}

// NewEmbeddingCache opens (or creates) the JSONL file at path and replays it
// into memory. The returned cache holds an open append handle to the file
// until Close is called.
func NewEmbeddingCache(path string) (*EmbeddingCache, error) {
	c := &EmbeddingCache{
		entries: make(map[string]CachedEmbedding),
		path:    path,
	}

	if err := c.load(); err != nil {
		return nil, fmt.Errorf("failed to load embedding cache: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to open embedding cache for append: %w", err)
	}
	c.file = f

	return c, nil
}

// load reads the JSONL file from disk and populates the in-memory map. Last
// line wins for any duplicate key. Missing file is not an error.
func (c *EmbeddingCache) load() error {
	f, err := os.Open(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	lineNum := 0
	for {
		lineNum++
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			// Trim trailing newline if present
			if line[len(line)-1] == '\n' {
				line = line[:len(line)-1]
			}
			if len(line) == 0 {
				if err == io.EOF {
					break
				}
				continue
			}
			var entry CachedEmbedding
			if jerr := json.Unmarshal(line, &entry); jerr != nil {
				// Skip malformed lines but don't abort - this matches the
				// resilient behavior of the events.jsonl loader. Avoid
				// logging the line content to prevent leaking unexpected data.
				if err == io.EOF {
					break
				}
				continue
			}
			if entry.Key != "" {
				c.entries[entry.Key] = entry
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// Close flushes any buffered writes and closes the append handle.
func (c *EmbeddingCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.file == nil {
		return nil
	}
	err := c.file.Close()
	c.file = nil
	return err
}

// Has reports whether the given key has a cached embedding.
func (c *EmbeddingCache) Has(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.entries[key]
	return ok
}

// Get returns the cached embedding for key, if present.
func (c *EmbeddingCache) Get(key string) (CachedEmbedding, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[key]
	return e, ok
}

// All returns a snapshot copy of all cached entries.
func (c *EmbeddingCache) All() map[string]CachedEmbedding {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string]CachedEmbedding, len(c.entries))
	for k, v := range c.entries {
		out[k] = v
	}
	return out
}

// Len returns the number of cached entries.
func (c *EmbeddingCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// Add appends a new entry to the JSONL file and updates the in-memory map.
// If an entry with the same key already exists it is overwritten in memory
// and a new line is appended (last line wins on next load).
func (c *EmbeddingCache) Add(entry CachedEmbedding) error {
	if entry.Key == "" {
		return fmt.Errorf("embedding cache: empty key")
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("embedding cache: marshal failed: %w", err)
	}
	data = append(data, '\n')

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.file == nil {
		return fmt.Errorf("embedding cache: file handle closed")
	}
	if _, err := c.file.Write(data); err != nil {
		return fmt.Errorf("embedding cache: write failed: %w", err)
	}
	if err := c.file.Sync(); err != nil {
		return fmt.Errorf("embedding cache: sync failed: %w", err)
	}
	c.entries[entry.Key] = entry
	return nil
}
