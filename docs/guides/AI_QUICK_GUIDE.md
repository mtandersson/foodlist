# AI Assistant Instructions: TDD Event-Sourced Real-Time Apps

## Core Workflow

### 1. Schema First

- Create `schema/` with JSON Schema for all events
- Generate types for backend + frontend
- Single source of truth prevents drift

### 2. TDD Cycle (Always!)

```
Write Test → Run (Fail) → Implement → Run (Pass) → Refactor → Repeat
```

### 3. Build Order (with tests at each step)

#### Backend

1. **Event Store** (JSONL append-only, channels not locks)
   - Tests: write, read, concurrent access, persistence
2. **State Projection** (events → current state)
   - Tests: each event type, sorting, filtering
3. **WebSocket Server** (broadcast, state rollup)
   - Tests: connect, disconnect, broadcast, rollup

#### Frontend

4. **WebSocket Client** (reconnection, queuing)
   - Tests: connect, send, receive, reconnect, queue
5. **State Store** (optimistic updates, reconciliation)
   - Tests: apply events, optimistic flow, sorting
6. **UI Components** (reactive, animated)
   - Tests: render, interactions, animations

### 4. Test Coverage Requirements

**Unit Tests:**

- ✓ Event serialization/parsing
- ✓ State projection (all event types)
- ✓ WebSocket lifecycle
- ✓ Optimistic updates
- ✓ Sorting/filtering

**Integration Tests:**

- ✓ Event persistence
- ✓ Client-server communication
- ✓ Multi-client broadcast

## Event Sourcing Rules

1. **Events describe what happened** (past tense)

   - ✓ `TodoCreated`, `TodoCompleted`
   - ✗ `CreateTodo`, `CompleteTodo`

2. **Events are immutable** (append-only)
3. **State is derived** from event replay
4. **Each event has:** type, id, timestamp, data

## Optimistic Update Pattern

```typescript
// 1. Apply locally (instant UX)
localState.apply(event)

// 2. Send to server
ws.send(event)

// 3. Server confirms → broadcast
ws.onMessage((confirmedEvent) => {
  // Already applied, just reconcile if needed
  if (confirmedEvent.id !== event.id) {
    localState.reconcile(event, confirmedEvent)
  }
})
```

## Concurrency Patterns

**Go:** Single writer goroutine + channels

```go
writeCh := make(chan Event)
go func() {
  for event := range writeCh {
    file.Write(event)
  }
}()
```

**TypeScript:** Async/await + queues

```typescript
if (!connected) {
  queue.push(event)
} else {
  await ws.send(event)
}
```

## CSS Best Practices

**No Magic Numbers - Always Use Variables:**

```css
/* ✗ BAD - Magic numbers everywhere */
.button {
  padding: 12px 16px;
  border-radius: 8px;
  font-size: 16px;
  margin-bottom: 24px;
}

/* ✓ GOOD - Semantic constants */
.button {
  padding: var(--spacing-md) var(--spacing-lg);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-base);
  margin-bottom: var(--spacing-2xl);
}
```

**Constant Categories:**
- Spacing: `--spacing-xs` through `--spacing-5xl`
- Radii: `--radius-sm`, `--radius-md`, `--radius-lg`, `--radius-full`
- Fonts: `--font-size-xs` through `--font-size-2xl`
- Colors: `--primary-color`, `--text-primary`, etc.
- Shadows: `--shadow-sm` through `--shadow-2xl`
- Durations: `--duration-fast`, `--duration-normal`, etc.

**Define all constants in `frontend/src/app.css` under `:root`**

## Critical Checklist

Before marking complete:

- [ ] All tests pass (backend, frontend)
- [ ] No console errors
- [ ] Events persist correctly
- [ ] Multi-client sync works
- [ ] Handles reconnection
- [ ] Animations smooth (60fps)
- [ ] No memory leaks
- [ ] Documentation complete
- [ ] No magic numbers in CSS (use variables)
- [ ] Consistent spacing/sizing via design system

## Common Issues & Solutions

| Issue                       | Solution                                    |
| --------------------------- | ------------------------------------------- |
| Tests not failing first     | Write test before code, verify it fails     |
| State drift between clients | Check event ordering, use sequence numbers  |
| Race conditions             | Use channels (Go) or single event loop (JS) |
| Memory leaks                | Close connections, cleanup subscriptions    |
| Slow animations             | Use CSS transforms (GPU), virtualize lists  |
| Magic numbers in CSS        | Use CSS custom properties (variables)       |
| Inconsistent spacing        | Use design system constants from :root      |

## Development Flow Example

```bash
# 1. Write failing test
echo "func TestFeature(t *testing.T) { ... }" >> feature_test.go
go test ./... # ✗ FAIL

# 2. Implement feature
echo "func Feature() { ... }" >> feature.go
go test ./... # ✓ PASS

# 3. Refactor if needed
# ... improve code ...
go test ./... # ✓ STILL PASS

# 4. Repeat for next feature
```

## Quick Commands

```bash
# Run all backend tests
go test ./...

# Run all frontend tests
npm test

# Dev servers (2 terminals)
./backend/server &
npm run dev -w frontend
```

## Non-Negotiables

1. **Tests first, always** - No coding before failing test exists
2. **Channels over locks** - Go concurrency via channels
3. **Optimistic updates** - Don't wait for server
4. **Event sourcing** - Append-only, immutable events
5. **Type safety** - Generated types from schema
6. **Multi-client sync** - Test with multiple tabs
8. **No magic numbers in CSS** - Always use CSS variables
9. **Design system consistency** - Use predefined constants

## Success = Green Tests + Working App

- Backend tests: ✓
- Frontend tests: ✓
- Manual testing: ✓
- Performance: ✓
- Documentation: ✓

**Then and only then: Ship it! 🚀**
