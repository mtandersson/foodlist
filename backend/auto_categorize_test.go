package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

// stubEmbedder is a deterministic, controllable Embedder for tests. The
// vector returned for each call is taken from `vectors` (defaulting to a
// canned near-vector). `release` (if non-nil) is awaited before responding
// so tests can race other work against an in-flight embedder call.
type stubEmbedder struct {
	mu         sync.Mutex
	calls      atomic.Uint64
	vectors    map[string][]float32 // key -> vector to return
	defaultVec []float32
	err        error
	// release, when set, is awaited before returning. The test signals
	// completion by closing this channel. Each call awaits its own copy.
	release chan struct{}
	// observed records every key requested, in order, for assertions.
	observed []string
}

func newStubEmbedder() *stubEmbedder {
	return &stubEmbedder{
		vectors:    map[string][]float32{},
		defaultVec: unit3(angleNear),
	}
}

func (s *stubEmbedder) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	s.calls.Add(1)
	s.mu.Lock()
	s.observed = append(s.observed, texts...)
	rel := s.release
	err := s.err
	s.mu.Unlock()

	if rel != nil {
		select {
		case <-rel:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if err != nil {
		return nil, err
	}
	out := make([][]float32, len(texts))
	for i, t := range texts {
		s.mu.Lock()
		v, ok := s.vectors[t]
		s.mu.Unlock()
		if !ok {
			v = s.defaultVec
		}
		// Copy so callers can't mutate our state.
		vc := make([]float32, len(v))
		copy(vc, v)
		out[i] = vc
	}
	return out, nil
}

func (s *stubEmbedder) setRelease(ch chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.release = ch
}

func (s *stubEmbedder) setError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.err = err
}

func (s *stubEmbedder) callCount() uint64 { return s.calls.Load() }

func (s *stubEmbedder) observedKeys() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.observed))
	copy(out, s.observed)
	return out
}

// autoCategorizeFixture stands up a server with the auto-categorize feature
// wired to a stub embedder and a freshly-loaded cache. The caller can
// pre-populate `existingTodos` (each gets a corresponding cache entry) so
// the categorizer has something to compare against.
type autoCategorizeFixture struct {
	server    *Server
	embed     *stubEmbedder
	cache     *EmbeddingCache
	tmpDir    string
	cachePath string
}

type seedTodo struct {
	id         string
	name       string
	theta      float64
	categoryID string
	ageBefore  time.Duration
}

func newAutoCategorizeFixture(t *testing.T, seed []seedTodo, cats []string) *autoCategorizeFixture {
	t.Helper()
	tmpDir := t.TempDir()
	eventsPath := filepath.Join(tmpDir, "events.jsonl")
	cachePath := filepath.Join(tmpDir, "embeddings.jsonl")

	store, err := NewEventStore(eventsPath)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	server := NewServer(store)

	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	for _, c := range cats {
		require.NoError(t, server.store.Append(CategoryCreated{
			Type:      "CategoryCreated",
			ID:        c,
			Name:      c,
			CreatedAt: now,
			SortOrder: 1000,
		}))
	}
	for i, s := range seed {
		cid := s.categoryID
		var catPtr *string
		if cid != "" {
			catPtr = &cid
		}
		require.NoError(t, server.store.Append(TodoCreated{
			Type:       "TodoCreated",
			ID:         s.id,
			Name:       s.name,
			CreatedAt:  now.Add(-s.ageBefore),
			SortOrder:  1000 + i,
			CategoryID: catPtr,
		}))
	}
	require.NoError(t, server.LoadEvents())

	cache, err := NewEmbeddingCache(cachePath)
	require.NoError(t, err)
	t.Cleanup(func() { cache.Close() })
	for _, s := range seed {
		key := normalizeName(s.name)
		require.NoError(t, cache.Add(CachedEmbedding{
			Key:    key,
			Text:   key,
			Model:  "test",
			Dim:    3,
			Vector: unit3(s.theta),
		}))
	}

	embed := newStubEmbedder()
	server.SetEmbeddingCache(cache)
	server.SetEmbeddingClient(embed)
	cat := NewCategorizer(Categorizer{
		Now: func() time.Time { return now },
	})
	server.SetCategorizer(&cat)

	go server.Run()

	return &autoCategorizeFixture{
		server:    server,
		embed:     embed,
		cache:     cache,
		tmpDir:    tmpDir,
		cachePath: cachePath,
	}
}

// createTodoSync drives a CreateTodo through ExecuteCommand and waits for
// the auto-categorize hook to finish by polling the metrics counters until
// the attempt is fully resolved. Returns the produced todo ID.
func (f *autoCategorizeFixture) createTodoSync(t *testing.T, id, name string) {
	t.Helper()
	startAttempts := f.server.autoCategorizeMetrics.Attempted.Load()
	require.NoError(t, f.server.ExecuteCommand(CreateTodoCommand{
		BaseCommand: BaseCommand{Type: "CreateTodo", CommandID: "cmd-" + id},
		ID:          id,
		Name:        name,
	}))
	// Wait for the async hook to complete (the attempt counter increments
	// at the start, but all metric updates happen before the goroutine
	// returns, so we wait for one of the terminal counters to advance).
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		attempts := f.server.autoCategorizeMetrics.Attempted.Load()
		if attempts <= startAttempts {
			time.Sleep(5 * time.Millisecond)
			continue
		}
		// Attempt was registered; now wait until the goroutine has had a
		// chance to record a terminal outcome by polling the snapshot.
		snap := f.server.autoCategorizeMetrics.Snapshot()
		terminal := snap.Suggested + snap.RejectedThreshold + snap.RejectedGate +
			snap.RejectedNoSignal + snap.SkippedUserSet + snap.SkippedDeleted +
			snap.EmbedFailed + snap.EmitFailed
		if terminal >= attempts {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for auto-categorize to complete for todo %q", id)
}

func TestAutoCategorize_AboveThresholdSuggests(t *testing.T) {
	f := newAutoCategorizeFixture(t, []seedTodo{
		{id: "seed-1", name: "milk", theta: angleNear, categoryID: "dairy", ageBefore: time.Hour},
		{id: "seed-2", name: "yogurt", theta: angleNear, categoryID: "dairy", ageBefore: time.Hour},
		{id: "seed-3", name: "butter", theta: angleNear, categoryID: "dairy", ageBefore: time.Hour},
	}, []string{"dairy"})

	// Pre-seed cache for the new item so we know its vector exactly (and
	// can keep this test independent of any embedder fetch).
	key := normalizeName("cheese")
	require.NoError(t, f.cache.Add(CachedEmbedding{
		Key: key, Text: key, Model: "test", Dim: 3, Vector: unit3(angleNear),
	}))

	f.createTodoSync(t, "new-1", "cheese")

	todo, ok := f.server.state.GetTodo("new-1")
	require.True(t, ok)
	require.NotNil(t, todo.CategoryID, "expected auto-categorize to assign a category")
	require.Equal(t, "dairy", *todo.CategoryID)

	snap := f.server.autoCategorizeMetrics.Snapshot()
	require.Equal(t, uint64(1), snap.Suggested)
	require.Equal(t, uint64(0), snap.EmbedFailed)
}

func TestAutoCategorize_BelowThresholdDoesNotEmit(t *testing.T) {
	f := newAutoCategorizeFixture(t, []seedTodo{
		// Only one mid-similarity item -> blended too low to clear threshold.
		{id: "seed-1", name: "tofu", theta: angleMid, categoryID: "produce", ageBefore: time.Hour},
	}, []string{"produce"})

	key := normalizeName("apple")
	require.NoError(t, f.cache.Add(CachedEmbedding{
		Key: key, Text: key, Model: "test", Dim: 3, Vector: unit3(0),
	}))

	f.createTodoSync(t, "new-1", "apple")

	todo, ok := f.server.state.GetTodo("new-1")
	require.True(t, ok)
	require.Nil(t, todo.CategoryID, "expected no category to be assigned")

	snap := f.server.autoCategorizeMetrics.Snapshot()
	require.Equal(t, uint64(0), snap.Suggested)
	require.GreaterOrEqual(t, snap.RejectedNoSignal, uint64(1))
}

func TestAutoCategorize_MaxSimGateBlocks(t *testing.T) {
	// Many items at weak-but-above-floor similarity. Without the gate
	// they'd sum to a final score > threshold. With it they're rejected.
	seed := []seedTodo{}
	for i := 0; i < 30; i++ {
		seed = append(seed, seedTodo{
			id:         fmt.Sprintf("seed-%d", i),
			name:       fmt.Sprintf("weak-%d", i),
			theta:      1.0,
			categoryID: "junk",
			ageBefore:  time.Hour,
		})
	}
	f := newAutoCategorizeFixture(t, seed, []string{"junk"})

	key := normalizeName("new")
	require.NoError(t, f.cache.Add(CachedEmbedding{
		Key: key, Text: key, Model: "test", Dim: 3, Vector: unit3(0),
	}))

	f.createTodoSync(t, "new-1", "new")

	todo, ok := f.server.state.GetTodo("new-1")
	require.True(t, ok)
	require.Nil(t, todo.CategoryID, "max-sim gate should have blocked the weak-volume category")
}

func TestAutoCategorize_CacheMissCallsEmbedderOnce(t *testing.T) {
	f := newAutoCategorizeFixture(t, []seedTodo{
		{id: "seed-1", name: "milk", theta: angleNear, categoryID: "dairy", ageBefore: time.Hour},
		{id: "seed-2", name: "yogurt", theta: angleNear, categoryID: "dairy", ageBefore: time.Hour},
		{id: "seed-3", name: "butter", theta: angleNear, categoryID: "dairy", ageBefore: time.Hour},
	}, []string{"dairy"})

	// New name with no cache entry → triggers the embedder. Configure the
	// stub to return a near-vector so dairy wins.
	f.embed.mu.Lock()
	f.embed.vectors[normalizeName("cheese")] = unit3(angleNear)
	f.embed.mu.Unlock()

	f.createTodoSync(t, "new-1", "cheese")

	require.Equal(t, uint64(1), f.embed.callCount(), "expected exactly one embedder call")

	keys := f.embed.observedKeys()
	require.Equal(t, []string{normalizeName("cheese")}, keys)

	// Cache must have a new row for the key.
	entry, ok := f.cache.Get(normalizeName("cheese"))
	require.True(t, ok, "expected new embedding to be persisted")
	require.Equal(t, 3, entry.Dim)
}

func TestAutoCategorize_CacheHitDoesNotCallEmbedder(t *testing.T) {
	f := newAutoCategorizeFixture(t, []seedTodo{
		{id: "seed-1", name: "milk", theta: angleNear, categoryID: "dairy", ageBefore: time.Hour},
	}, []string{"dairy"})

	key := normalizeName("cheese")
	require.NoError(t, f.cache.Add(CachedEmbedding{
		Key: key, Text: key, Model: "test", Dim: 3, Vector: unit3(angleNear),
	}))

	f.createTodoSync(t, "new-1", "cheese")

	require.Equal(t, uint64(0), f.embed.callCount(), "expected embedder to NOT be called on cache hit")
}

func TestAutoCategorize_SingleFlightDedup(t *testing.T) {
	f := newAutoCategorizeFixture(t, []seedTodo{
		{id: "seed-1", name: "milk", theta: angleNear, categoryID: "dairy", ageBefore: time.Hour},
		{id: "seed-2", name: "yogurt", theta: angleNear, categoryID: "dairy", ageBefore: time.Hour},
		{id: "seed-3", name: "butter", theta: angleNear, categoryID: "dairy", ageBefore: time.Hour},
	}, []string{"dairy"})

	// Block the embedder until all concurrent flights have arrived.
	release := make(chan struct{})
	f.embed.setRelease(release)
	f.embed.mu.Lock()
	f.embed.vectors[normalizeName("cheese")] = unit3(angleNear)
	f.embed.mu.Unlock()

	const concurrency = 5
	startAttempts := f.server.autoCategorizeMetrics.Attempted.Load()
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("todo-%d", i)
			// Each todo has a distinct ID but the SAME normalized name,
			// so the singleflight key collides.
			require.NoError(t, f.server.ExecuteCommand(CreateTodoCommand{
				BaseCommand: BaseCommand{Type: "CreateTodo", CommandID: "cmd-" + id},
				ID:          id,
				Name:        "cheese",
			}))
		}(i)
	}

	// Wait until all attempt counters have ticked (goroutines spawned).
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if f.server.autoCategorizeMetrics.Attempted.Load() >= startAttempts+concurrency {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	require.GreaterOrEqual(t, f.server.autoCategorizeMetrics.Attempted.Load(), startAttempts+concurrency)

	// Give callers time to enter singleflight. The embedder blocks until
	// we release, so any goroutine that got past Has() is parked inside
	// Do(). 50ms is generous.
	time.Sleep(50 * time.Millisecond)
	close(release)

	wg.Wait()

	// Poll until all attempts are terminal.
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		snap := f.server.autoCategorizeMetrics.Snapshot()
		terminal := snap.Suggested + snap.RejectedThreshold + snap.RejectedGate +
			snap.RejectedNoSignal + snap.SkippedUserSet + snap.SkippedDeleted +
			snap.EmbedFailed + snap.EmitFailed
		if terminal >= concurrency {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	require.Equal(t, uint64(1), f.embed.callCount(),
		"expected exactly one embedder call across %d concurrent suggestions", concurrency)

	// Cache must contain exactly one entry for the key.
	entry, ok := f.cache.Get(normalizeName("cheese"))
	require.True(t, ok)
	require.Equal(t, 3, entry.Dim)

	snap := f.server.autoCategorizeMetrics.Snapshot()
	require.GreaterOrEqual(t, snap.Deduped, uint64(1),
		"expected at least one dedup hit (got %+v)", snap)
}

func TestAutoCategorize_UserOverrideRace(t *testing.T) {
	f := newAutoCategorizeFixture(t, []seedTodo{
		{id: "seed-1", name: "milk", theta: angleNear, categoryID: "dairy", ageBefore: time.Hour},
		{id: "seed-2", name: "yogurt", theta: angleNear, categoryID: "dairy", ageBefore: time.Hour},
		{id: "seed-3", name: "butter", theta: angleNear, categoryID: "dairy", ageBefore: time.Hour},
	}, []string{"dairy", "treats"})

	// Block the embedder so the auto-categorize goroutine is parked
	// behind the embed fetch while we issue the manual categorize.
	release := make(chan struct{})
	f.embed.setRelease(release)
	f.embed.mu.Lock()
	f.embed.vectors[normalizeName("cheese")] = unit3(angleNear)
	f.embed.mu.Unlock()

	startAttempts := f.server.autoCategorizeMetrics.Attempted.Load()
	require.NoError(t, f.server.ExecuteCommand(CreateTodoCommand{
		BaseCommand: BaseCommand{Type: "CreateTodo", CommandID: "cmd-new"},
		ID:          "new-1",
		Name:        "cheese",
	}))

	// Wait for the auto goroutine to have started.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if f.server.autoCategorizeMetrics.Attempted.Load() > startAttempts {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}

	// User manually categorizes BEFORE the auto suggestion resolves.
	userCategory := "treats"
	require.NoError(t, f.server.ExecuteCommand(CategorizeTodoCommand{
		BaseCommand: BaseCommand{Type: "CategorizeTodo", CommandID: "cmd-user"},
		ID:          "new-1",
		CategoryID:  &userCategory,
	}))

	// Now release the embedder to let the auto goroutine continue. It
	// must observe the user-set category and back off.
	close(release)

	// Wait for terminal state.
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		snap := f.server.autoCategorizeMetrics.Snapshot()
		if snap.SkippedUserSet >= 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	todo, ok := f.server.state.GetTodo("new-1")
	require.True(t, ok)
	require.NotNil(t, todo.CategoryID)
	require.Equal(t, "treats", *todo.CategoryID, "auto-categorize must not overwrite user choice")

	snap := f.server.autoCategorizeMetrics.Snapshot()
	require.Equal(t, uint64(1), snap.SkippedUserSet)
	require.Equal(t, uint64(0), snap.Suggested)
}

func TestAutoCategorize_FeatureOffNoOp(t *testing.T) {
	// Set up a server with NO auto-categorize wiring at all.
	tmpDir := t.TempDir()
	store, err := NewEventStore(filepath.Join(tmpDir, "events.jsonl"))
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	server := NewServer(store)
	go server.Run()

	require.False(t, server.AutoCategorizeEnabled())

	require.NoError(t, server.ExecuteCommand(CreateTodoCommand{
		BaseCommand: BaseCommand{Type: "CreateTodo", CommandID: "cmd-1"},
		ID:          "new-1",
		Name:        "anything",
	}))

	// Briefly wait; there's no metric to poll, but we want to confirm no
	// background categorize is happening.
	time.Sleep(50 * time.Millisecond)

	todo, ok := server.state.GetTodo("new-1")
	require.True(t, ok)
	require.Nil(t, todo.CategoryID)
}

func TestAutoCategorize_EmbedderErrorIsSwallowed(t *testing.T) {
	f := newAutoCategorizeFixture(t, []seedTodo{
		{id: "seed-1", name: "milk", theta: angleNear, categoryID: "dairy", ageBefore: time.Hour},
	}, []string{"dairy"})

	f.embed.setError(errors.New("upstream 500"))

	require.NoError(t, f.server.ExecuteCommand(CreateTodoCommand{
		BaseCommand: BaseCommand{Type: "CreateTodo", CommandID: "cmd-1"},
		ID:          "new-1",
		Name:        "novel-item",
	}))

	// Wait for the failure to register.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if f.server.autoCategorizeMetrics.EmbedFailed.Load() >= 1 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	todo, ok := f.server.state.GetTodo("new-1")
	require.True(t, ok)
	require.Nil(t, todo.CategoryID, "embedder failure must not leave a partial assignment")

	snap := f.server.autoCategorizeMetrics.Snapshot()
	require.Equal(t, uint64(1), snap.EmbedFailed)
	require.Equal(t, uint64(0), snap.Suggested)
}

func TestAutoCategorize_BroadcastOrdering(t *testing.T) {
	f := newAutoCategorizeFixture(t, []seedTodo{
		{id: "seed-1", name: "milk", theta: angleNear, categoryID: "dairy", ageBefore: time.Hour},
		{id: "seed-2", name: "yogurt", theta: angleNear, categoryID: "dairy", ageBefore: time.Hour},
		{id: "seed-3", name: "butter", theta: angleNear, categoryID: "dairy", ageBefore: time.Hour},
	}, []string{"dairy"})

	// Pre-seed the cache so the embedder isn't invoked; this keeps the
	// test fast and avoids any extra latency hiding the race.
	key := normalizeName("cheese")
	require.NoError(t, f.cache.Add(CachedEmbedding{
		Key: key, Text: key, Model: "test", Dim: 3, Vector: unit3(angleNear),
	}))

	ts := httptest.NewServer(http.HandlerFunc(f.server.HandleWebSocket))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Drain StateRollup + ClientCount.
	for i := 0; i < 2; i++ {
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, _, err := conn.ReadMessage()
		require.NoError(t, err)
	}

	require.NoError(t, conn.WriteJSON(CreateTodoCommand{
		BaseCommand: BaseCommand{Type: "CreateTodo", CommandID: "cmd-1"},
		ID:          "new-1",
		Name:        "cheese",
	}))

	// Read messages until we've seen both TodoCreated and TodoCategorized
	// (or hit the deadline). Record the order.
	sawCreated := false
	sawCategorized := false
	categorizedBeforeCreated := false
	deadline := time.Now().Add(3 * time.Second)
	for !sawCategorized && time.Now().Before(deadline) {
		conn.SetReadDeadline(deadline)
		_, msg, err := conn.ReadMessage()
		require.NoError(t, err)
		var head struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		}
		if err := json.Unmarshal(msg, &head); err != nil {
			continue
		}
		switch head.Type {
		case "TodoCreated":
			if head.ID == "new-1" {
				sawCreated = true
			}
		case "TodoCategorized":
			if head.ID == "new-1" {
				sawCategorized = true
				if !sawCreated {
					categorizedBeforeCreated = true
				}
			}
		}
	}

	require.True(t, sawCreated, "expected TodoCreated for new-1")
	require.True(t, sawCategorized, "expected TodoCategorized for new-1 (auto-suggest)")
	require.False(t, categorizedBeforeCreated,
		"broadcast ordering violated: TodoCategorized arrived before TodoCreated")
}
