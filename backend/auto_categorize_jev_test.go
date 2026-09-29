package main

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type stubCategoryDecider struct {
	decision *CategoryDecision
	err      error
	calls    atomic.Uint64
}

func (s *stubCategoryDecider) DecideCategory(_ context.Context, _ string, _ []Category, _ map[string][]string) (*CategoryDecision, error) {
	s.calls.Add(1)
	if s.err != nil {
		return nil, s.err
	}
	return s.decision, nil
}

func TestAutoCategorizeJevPreferredOverEmbedding(t *testing.T) {
	f := newAutoCategorizeFixture(t, []seedTodo{
		{id: "seed-1", name: "milk", theta: angleNear, categoryID: "dairy", ageBefore: time.Hour},
		{id: "seed-2", name: "yogurt", theta: angleNear, categoryID: "dairy", ageBefore: time.Hour},
		{id: "seed-3", name: "butter", theta: angleNear, categoryID: "dairy", ageBefore: time.Hour},
	}, []string{"dairy", "produce"})

	jev := &stubCategoryDecider{decision: &CategoryDecision{
		CategoryID: "produce", Confidence: 0.92, Probability: 0.95, Model: "jev-test",
	}}
	f.server.SetCategoryDecider(jev)

	f.createTodoSync(t, "new-jev", "avokado")

	todo, ok := f.server.state.GetTodo("new-jev")
	require.True(t, ok)
	require.NotNil(t, todo.CategoryID)
	require.Equal(t, "produce", *todo.CategoryID)
	require.Equal(t, uint64(1), jev.calls.Load())
	require.Equal(t, uint64(0), f.embed.callCount(), "embedding provider should not run after a confident Jev decision")
}

func TestAutoCategorizeFallsBackToEmbeddingWhenJevFails(t *testing.T) {
	f := newAutoCategorizeFixture(t, []seedTodo{
		{id: "seed-1", name: "milk", theta: angleNear, categoryID: "dairy", ageBefore: time.Hour},
		{id: "seed-2", name: "yogurt", theta: angleNear, categoryID: "dairy", ageBefore: time.Hour},
		{id: "seed-3", name: "butter", theta: angleNear, categoryID: "dairy", ageBefore: time.Hour},
	}, []string{"dairy"})

	key := normalizeName("cheese")
	require.NoError(t, f.cache.Add(CachedEmbedding{
		Key: key, Text: key, Model: "test", Dim: 3, Vector: unit3(angleNear),
	}))

	jev := &stubCategoryDecider{err: errors.New("upstream unavailable")}
	f.server.SetCategoryDecider(jev)

	f.createTodoSync(t, "new-fallback", "cheese")

	todo, ok := f.server.state.GetTodo("new-fallback")
	require.True(t, ok)
	require.NotNil(t, todo.CategoryID)
	require.Equal(t, "dairy", *todo.CategoryID)
	require.Equal(t, uint64(1), jev.calls.Load())
}

func TestAutoCategorizeFallsBackToEmbeddingWhenJevAbstains(t *testing.T) {
	f := newAutoCategorizeFixture(t, []seedTodo{
		{id: "seed-1", name: "milk", theta: angleNear, categoryID: "dairy", ageBefore: time.Hour},
		{id: "seed-2", name: "yogurt", theta: angleNear, categoryID: "dairy", ageBefore: time.Hour},
		{id: "seed-3", name: "butter", theta: angleNear, categoryID: "dairy", ageBefore: time.Hour},
	}, []string{"dairy"})

	key := normalizeName("cheese")
	require.NoError(t, f.cache.Add(CachedEmbedding{
		Key: key, Text: key, Model: "test", Dim: 3, Vector: unit3(angleNear),
	}))

	jev := &stubCategoryDecider{decision: nil}
	f.server.SetCategoryDecider(jev)

	f.createTodoSync(t, "new-abstain", "cheese")

	todo, ok := f.server.state.GetTodo("new-abstain")
	require.True(t, ok)
	require.NotNil(t, todo.CategoryID)
	require.Equal(t, "dairy", *todo.CategoryID)
	require.Equal(t, uint64(1), jev.calls.Load())
}

func TestAutoCategorizeWorksWithJevOnly(t *testing.T) {
	store, err := NewEventStore(filepath.Join(t.TempDir(), "events.jsonl"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })

	server := NewServer(store)
	jev := &stubCategoryDecider{decision: &CategoryDecision{
		CategoryID: "dry", Confidence: 0.90, Probability: 0.94, Model: "jev-test",
	}}
	server.SetCategoryDecider(jev)
	require.True(t, server.AutoCategorizeEnabled())

	require.NoError(t, server.ExecuteCommand(CreateCategoryCommand{
		BaseCommand: BaseCommand{Type: "CreateCategory", CommandID: "cat-cmd"},
		ID:          "dry",
		Name:        "Torrvaror",
	}))
	require.NoError(t, server.ExecuteCommand(CreateTodoCommand{
		BaseCommand: BaseCommand{Type: "CreateTodo", CommandID: "todo-cmd"},
		ID:          "jev-only",
		Name:        "maizena",
	}))

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		todo, ok := server.state.GetTodo("jev-only")
		if ok && todo.CategoryID != nil {
			require.Equal(t, "dry", *todo.CategoryID)
			require.Equal(t, uint64(1), jev.calls.Load())
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("timed out waiting for Jev-only auto-categorization")
}
