package main

import (
	"sort"
	"sync"
)

// CookSessions holds the ephemeral, server-side cook-mode state shared
// across all connected clients. State is lost on restart and is NOT
// persisted to events.jsonl.
//
// Thread safety: every mutation takes the mutex, mutates, then invokes
// the optional broadcast hook while still holding the lock. Holding
// the lock during the (non-blocking) enqueue is what guarantees
// CookStateChanged messages are queued in the same order as their
// mutations - if we released the lock first and let the caller enqueue,
// goroutine scheduling could swap the order between two near-
// simultaneous Check calls.
type CookSessions struct {
	mu        sync.Mutex
	state     map[string]map[int]struct{}
	broadcast func(recipeID string, steps []int)
}

// NewCookSessions returns an empty session table.
func NewCookSessions() *CookSessions {
	return &CookSessions{state: make(map[string]map[int]struct{})}
}

// SetBroadcastHook installs a hook that fires under the cook mutex
// after every successful Check/Uncheck/Reset/PruneAbove/Drop mutation
// that changes observable state. Pass nil to detach (used by tests).
//
// The hook MUST NOT acquire the cook mutex itself (it's already held)
// and MUST NOT block - callers should use a non-blocking enqueue or
// a buffered channel.
func (c *CookSessions) SetBroadcastHook(fn func(recipeID string, steps []int)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.broadcast = fn
}

// Snapshot returns the current state as a {recipeId -> sortedSteps} map.
// Used by CookStateRollup on connect.
func (c *CookSessions) Snapshot() map[string][]int {
	c.mu.Lock()
	defer c.mu.Unlock()

	out := make(map[string][]int, len(c.state))
	for id, steps := range c.state {
		if len(steps) == 0 {
			continue
		}
		out[id] = sortedKeys(steps)
	}
	return out
}

// Check sets stepIndex as checked and returns the resulting sorted list.
func (c *CookSessions) Check(recipeID string, stepIndex int) []int {
	c.mu.Lock()
	defer c.mu.Unlock()

	steps, ok := c.state[recipeID]
	if !ok {
		steps = make(map[int]struct{})
		c.state[recipeID] = steps
	}
	steps[stepIndex] = struct{}{}
	out := sortedKeys(steps)
	if c.broadcast != nil {
		c.broadcast(recipeID, out)
	}
	return out
}

// Uncheck clears stepIndex and returns the resulting sorted list.
// The broadcast hook fires only when the call actually changed state
// (the step had been checked); idempotent unchecks do not produce a
// CookStateChanged.
func (c *CookSessions) Uncheck(recipeID string, stepIndex int) []int {
	c.mu.Lock()
	defer c.mu.Unlock()

	steps, ok := c.state[recipeID]
	if !ok {
		return nil
	}
	if _, had := steps[stepIndex]; !had {
		return sortedKeys(steps)
	}
	delete(steps, stepIndex)
	if len(steps) == 0 {
		delete(c.state, recipeID)
		if c.broadcast != nil {
			c.broadcast(recipeID, []int{})
		}
		return nil
	}
	out := sortedKeys(steps)
	if c.broadcast != nil {
		c.broadcast(recipeID, out)
	}
	return out
}

// Reset clears every step for a recipe. The broadcast hook fires only
// when a session existed; resetting an empty session is a no-op.
func (c *CookSessions) Reset(recipeID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.state[recipeID]; !ok {
		return
	}
	delete(c.state, recipeID)
	if c.broadcast != nil {
		c.broadcast(recipeID, []int{})
	}
}

// PruneAbove removes any stepIndex >= maxSteps for the recipe and
// returns the new sorted list. Used by Update when len(Instructions)
// shrinks. Returns (nil, false) when no session exists for the recipe.
func (c *CookSessions) PruneAbove(recipeID string, maxSteps int) ([]int, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	steps, ok := c.state[recipeID]
	if !ok {
		return nil, false
	}
	changed := false
	for idx := range steps {
		if idx >= maxSteps {
			delete(steps, idx)
			changed = true
		}
	}
	if !changed {
		return sortedKeys(steps), false
	}
	if len(steps) == 0 {
		delete(c.state, recipeID)
		if c.broadcast != nil {
			c.broadcast(recipeID, []int{})
		}
		return nil, true
	}
	out := sortedKeys(steps)
	if c.broadcast != nil {
		c.broadcast(recipeID, out)
	}
	return out, true
}

// Drop removes all session state for a recipe (used on DELETE).
// Returns true if the recipe had an active session. The broadcast
// hook fires only when a session existed, so deleting a recipe that
// nobody had open does not generate a spurious CookStateChanged.
func (c *CookSessions) Drop(recipeID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.state[recipeID]
	delete(c.state, recipeID)
	if ok && c.broadcast != nil {
		c.broadcast(recipeID, []int{})
	}
	return ok
}

func sortedKeys(m map[int]struct{}) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}
