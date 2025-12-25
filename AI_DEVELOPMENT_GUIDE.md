# AI Development Instructions: Event-Sourced Real-Time Applications

## Overview

These instructions guide AI assistants (Cursor, Aider, etc.) through building event-sourced, real-time applications using Test-Driven Development (TDD) and comprehensive testing strategies.

## Core Principles

### 1. Test-Driven Development (TDD)

- **Always write tests first** before implementing features
- Follow the Red-Green-Refactor cycle:
  1. Write a failing test
  2. Write minimal code to make it pass
  3. Refactor while keeping tests green
- Test coverage should include:
  - Unit tests for individual functions/methods
  - Integration tests for component interactions
  - E2E tests for complete user flows

### 2. Schema-First Design

- Define data structures in a single source of truth (JSON Schema, Protocol Buffers, etc.)
- Generate types for all languages from the schema
- Keep frontend and backend types in sync
- Benefits:
  - No type drift between client/server
  - Automatic validation
  - Self-documenting API

### 3. Event Sourcing Architecture

- Store only events, not current state
- Build state by replaying events
- Events are append-only and immutable
- Each event describes what happened, not what should be done
- Event store pattern:
  ```
  TodoCreated → TodoCompleted → TodoStarred
  ```
- Benefits:
  - Complete audit trail
  - Time-travel debugging
  - Easy to implement undo/redo
  - Natural fit for real-time sync

## Step-by-Step Development Process

### Phase 0: Schema Definition (15 minutes)

1. Create a `schema/` directory
2. Define all event types in JSON Schema or similar
3. Set up code generation scripts for:
   - Backend types (Go structs, Java classes, etc.)
   - Frontend types (TypeScript interfaces)
4. Verify generated code compiles

**Example Structure:**

```
schema/
  ├── events.schema.json
  └── generate.sh
```

**Test:** Run generation script, verify types compile in both languages

### Phase 1: Backend Foundation (1-2 hours)

#### 1.1 Event Store (TDD)

**Write tests first:**

```go
func TestEventStore_WriteAndRead(t *testing.T)
func TestEventStore_ConcurrentWrites(t *testing.T)
func TestEventStore_PersistenceAcrossRestart(t *testing.T)
```

**Implementation requirements:**

- Append-only file storage (JSONL format)
- Concurrency via channels (Go) or similar patterns (not locks)
- Single writer goroutine pattern
- Each line is one JSON event

**Run tests:** All tests must pass before proceeding

#### 1.2 State Projection (TDD)

**Write tests first:**

```go
func TestState_ApplyTodoCreated(t *testing.T)
func TestState_ApplyTodoCompleted(t *testing.T)
func TestState_GetTodosSortedBySortOrder(t *testing.T)
```

**Implementation:**

- Build current state from event stream
- Handle all event types
- Return sorted, filtered views

**Run tests:** Verify all state transformations work correctly

#### 1.3 WebSocket Server (TDD)

**Write tests first:**

```go
func TestServer_AcceptConnection(t *testing.T)
func TestServer_SendStateRollupOnConnect(t *testing.T)
func TestServer_BroadcastEventToAllClients(t *testing.T)
func TestServer_HandleClientDisconnect(t *testing.T)
```

**Implementation:**

- WebSocket endpoint
- On connect: send full state rollup
- On event: persist → broadcast to all clients
- Handle disconnects gracefully

**Run tests:** Ensure WebSocket lifecycle works correctly

### Phase 2: Frontend Foundation (1-2 hours)

#### 2.1 WebSocket Client (TDD)

**Write tests first:**

```typescript
it("should connect to WebSocket server")
it("should receive and parse messages")
it("should send events to server")
it("should attempt reconnection on disconnect")
it("should queue messages while reconnecting")
```

**Implementation:**

- Auto-reconnection with exponential backoff
- Message queuing during disconnection
- Type-safe message parsing

**Run tests:** `npm test` - all tests must pass

#### 2.2 State Store with Optimistic Updates (TDD)

**Write tests first:**

```typescript
it("should apply StateRollup to initialize todos")
it("should apply TodoCreated event to add new todo")
it("should send optimistic create and update on confirmation")
it("should sort todos by sortOrder descending")
```

**Implementation:**

- Reactive store (Svelte stores, Redux, Zustand, etc.)
- Optimistic updates: apply locally before server confirmation
- Reconciliation: handle server responses
- Derived computed values (active todos, completed todos)

**Run tests:** Verify optimistic updates and reconciliation work

### Phase 3: Feature Implementation (2-3 hours)

For each feature, follow this pattern:

#### Feature Template (e.g., "Create Todo")

1. **Backend test:**
   ```go
   func TestState_TodoCreated_IncreasesTodCount(t *testing.T)
   ```
2. **Frontend test:**
   ```typescript
   it("should create todo and show immediately")
   ```
3. **Implement backend** - make tests pass
4. **Implement frontend** - make tests pass
5. **Run all tests** - ensure no regressions
6. **Manual test in browser** - verify UX

#### Priority Feature Order

1. Create todo (with sortOrder calculation)
2. Complete/uncomplete todo
3. Star/unstar todo (moves to top)
4. Drag & drop reordering
5. Rename todo (inline editing)
6. Delete todo

**After each feature:**

- Run backend tests: `go test ./...`
- Run frontend tests: `npm test`
- Check for regressions

### Phase 4: UI Implementation (1-2 hours)

#### 4.1 Component Structure

```
components/
  ├── TodoList.svelte/tsx    # Main container
  ├── TodoItem.svelte/tsx    # Individual todo
  └── AddTodo.svelte/tsx     # Input for new todos
```

#### 4.2 CSS Animations

Add transitions for:

- Todo creation (fade in)
- Todo completion (fade/slide)
- Reordering (smooth position changes)
- Starring (scale animation)
- Collapsible sections (slide)

**Use CSS transitions or framework-specific animation libraries**

#### 4.3 Accessibility

- Proper ARIA labels
- Keyboard navigation
- Focus management
- Screen reader support

### Phase 5: End-to-End Testing (1 hour)

#### 5.1 Setup Cypress/Playwright

```bash
npm install -D cypress
npx cypress open
```

#### 5.2 Write E2E Tests

```javascript
describe("Todo App", () => {
  it("should create a new todo")
  it("should complete a todo")
  it("should star a todo and move to top")
  it("should persist todos after reload")
  it("should sync between tabs")
  it("should handle offline/reconnection")
})
```

**Each test should:**

- Start from clean state
- Perform user actions
- Verify UI updates
- Check persistence
- Test real-time sync

#### 5.3 Multi-Client Testing

```javascript
it("should sync between two tabs", () => {
  // Create todo in tab 1
  // Verify appears in tab 2
  // Complete in tab 2
  // Verify completed in tab 1
})
```

### Phase 6: Integration & Polish (1 hour)

1. **Run all tests together:**

   ```bash
   cd backend && go test ./...
   cd frontend && npm test
   cd e2e && npm test
   ```

2. **Fix any flaky tests**
3. **Add missing test cases**
4. **Performance testing:**

   - Many concurrent clients
   - Large number of todos
   - Network interruptions

5. **Documentation:**
   - API/event documentation
   - Setup instructions
   - Development guide

## Testing Strategy Checklist

### Unit Tests

- [ ] Event store read/write
- [ ] Event parsing and serialization
- [ ] State projection for each event type
- [ ] WebSocket client connection/reconnection
- [ ] Store optimistic updates
- [ ] Sorting and filtering logic

### Integration Tests

- [ ] WebSocket server accepts connections
- [ ] Events persist to store
- [ ] Events broadcast to all clients
- [ ] State rollup on connect
- [ ] Client disconnect handling

### E2E Tests

- [ ] Create todo flow
- [ ] Complete todo flow
- [ ] Star todo flow
- [ ] Reorder todos via drag & drop
- [ ] Multi-tab sync
- [ ] Persistence across reload
- [ ] Offline behavior
- [ ] Reconnection recovery

## Best Practices

### Concurrency Patterns

**Go:** Use channels, not locks

```go
// Good: Single writer goroutine
go func() {
    for req := range writeCh {
        file.Write(req.data)
        req.resultCh <- nil
    }
}()

// Bad: Mutex locks everywhere
mutex.Lock()
file.Write(data)
mutex.Unlock()
```

**JavaScript:** Use async/await properly

```javascript
// Good: Handle errors
try {
  await ws.send(event)
} catch (error) {
  queueForRetry(event)
}
```

### Event Design

```typescript
// Good: Events describe what happened
{ type: "TodoCreated", id: "...", name: "..." }
{ type: "TodoCompleted", id: "...", completedAt: "..." }

// Bad: Events are commands
{ type: "CreateTodo", name: "..." }
{ type: "CompleteTodo", id: "..." }
```

### Optimistic Updates

```typescript
// 1. Apply locally immediately
store.update(optimisticEvent)

// 2. Send to server
ws.send(optimisticEvent)

// 3. Server confirms
ws.onMessage((serverEvent) => {
  // Reconcile if needed
  if (serverEvent.id !== optimisticEvent.id) {
    store.reconcile(optimisticEvent, serverEvent)
  }
})
```

### Error Handling

- Network errors: Queue and retry
- Validation errors: Show to user
- Server errors: Log and alert
- State conflicts: Last write wins or CRDT

## Common Pitfalls to Avoid

1. **Not testing first:** Always TDD, resist the urge to code first
2. **Using locks instead of channels:** In Go, prefer channels for concurrency
3. **Forgetting to close connections:** Always defer close() or use cleanup
4. **Not handling reconnection:** WebSocket will disconnect, plan for it
5. **Ignoring optimistic update conflicts:** Have a reconciliation strategy
6. **Skipping E2E tests:** They catch integration issues unit tests miss
7. **Not testing multi-client scenarios:** Real-time sync is core functionality
8. **Hardcoding URLs/ports:** Use environment variables
9. **Not cleaning up test data:** Each test should start clean
10. **Skipping animation testing:** Verify transitions don't break UX

## Performance Considerations

### Backend

- Event store: Use buffered writes for throughput
- WebSocket: Use connection pooling
- State: Cache projected state, don't rebuild on every read
- Broadcasting: Use pub/sub pattern for many clients

### Frontend

- Virtualize long todo lists (react-window, svelte-virtual)
- Debounce rapid updates
- Memoize expensive computations
- Use CSS transforms for animations (GPU-accelerated)

### Network

- Compress WebSocket messages
- Batch rapid events
- Delta updates instead of full state
- Implement pagination for large datasets

## Success Criteria

A successfully completed implementation should have:

- [ ] **All backend tests passing** (unit + integration)
- [ ] **All frontend tests passing** (unit + integration)
- [ ] **All E2E tests passing**
- [ ] **No console errors** in browser
- [ ] **No memory leaks** (check with browser devtools)
- [ ] **Handles 100+ concurrent clients**
- [ ] **Handles 1000+ todos without lag**
- [ ] **Survives network interruptions**
- [ ] **State stays consistent** across clients
- [ ] **Events persist** correctly
- [ ] **Animations are smooth** (60fps)
- [ ] **Code is documented**
- [ ] **Setup instructions work** for new developers

## Example Development Timeline

**Day 1: Foundation (4-5 hours)**

- Schema definition (30 min)
- Backend event store + tests (1 hour)
- Backend state projection + tests (1 hour)
- Backend WebSocket server + tests (1.5 hours)
- Frontend WebSocket client + tests (1 hour)

**Day 2: Features (4-5 hours)**

- Frontend state store + tests (1 hour)
- Create todo feature + tests (1 hour)
- Complete todo feature + tests (1 hour)
- Star todo feature + tests (1 hour)
- Drag & drop + tests (1 hour)

**Day 3: Polish & Testing (3-4 hours)**

- UI styling and animations (1.5 hours)
- E2E test suite (1.5 hours)
- Bug fixes and refinements (1 hour)
- Documentation (30 min)

**Total: 11-14 hours for complete implementation**

## Debugging Tips

### WebSocket Issues

```javascript
// Add verbose logging
ws.addEventListener("open", () => console.log("WS opened"))
ws.addEventListener("close", (e) => console.log("WS closed", e))
ws.addEventListener("error", (e) => console.error("WS error", e))
ws.addEventListener("message", (e) => console.log("WS message", e.data))
```

### Event Store Issues

```bash
# Inspect the JSONL file
cat events.jsonl | jq '.'

# Watch events being written
tail -f events.jsonl | jq '.'

# Count events by type
cat events.jsonl | jq -r '.type' | sort | uniq -c
```

### State Sync Issues

- Open browser devtools on multiple tabs
- Log state changes in each tab
- Compare timestamps
- Check event ordering

## Maintenance & Evolution

### Adding New Event Types

1. Update schema
2. Regenerate types
3. Add state projection handler
4. Write tests for new event
5. Update frontend UI
6. Add E2E test

### Performance Optimization

- Profile before optimizing
- Use performance.mark() in browser
- Use pprof in Go
- Optimize the slow parts, not everything

### Scaling Considerations

- Sharding event store by user/project
- Adding Redis for pub/sub
- Load balancing WebSocket connections
- CDN for static assets

---

## Quick Reference Commands

```bash
# Backend
cd backend
go test ./...                    # Run all tests
go test -v ./...                 # Verbose output
go test -cover ./...             # With coverage
go build -o server               # Build binary

# Frontend
cd frontend
npm test                         # Run tests once
npm test -- --watch              # Watch mode
npm run dev                      # Dev server
npm run build                    # Production build
npm run check                    # Type checking

# E2E
cd e2e
npm test                         # Run Cypress headless
npm run test:open                # Open Cypress GUI

# Full test suite
./run-all-tests.sh               # Custom script to run everything
```

## Additional Resources

- Event Sourcing: Martin Fowler's articles
- WebSocket Protocol: RFC 6455
- TDD: Kent Beck's "Test Driven Development by Example"
- Real-time Sync: CRDT papers and implementations
- Go Concurrency: "Concurrency in Go" by Katherine Cox-Buday

---

**Remember:** The goal is not just working code, but **tested, maintainable, and scalable** code. Take the time to do it right with TDD, and you'll save time debugging later.
