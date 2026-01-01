import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';
import type { Todo, Category, StateRollup, TodoCreated, TodoCompleted, Event, ServerMessage, ListTitleChanged, AutocompleteResponse, AutocompleteSuggestion } from './types';

// Mock handlers storage
let messageHandler: ((msg: ServerMessage) => void) | null = null;
let autocompleteHandler: ((response: AutocompleteResponse) => void) | null = null;
const mockSend = vi.fn();
const mockSendAutocomplete = vi.fn();

// Mock the websocket module
vi.mock('./websocket', () => {
  return {
    TodoWebSocket: class MockTodoWebSocket {
      constructor(_url: string) {}
      
      send(event: Event) {
        mockSend(JSON.stringify(event));
      }
      
      sendAutocompleteRequest(query: string, requestId: string) {
        mockSendAutocomplete({ query, requestId });
      }
      
      onMessage(handler: (msg: ServerMessage) => void) {
        messageHandler = handler;
        return () => { messageHandler = null; };
      }
      
      onAutocomplete(handler: (response: AutocompleteResponse) => void) {
        autocompleteHandler = handler;
        return () => { autocompleteHandler = null; };
      }
      
      onConnectionChange(_handler: (state: string) => void) {
        return () => {};
      }
      
      close() {}
      
      getConnectionState() {
        return 'CONNECTED';
      }
    },
    ConnectionState: {
      CONNECTING: 'CONNECTING',
      CONNECTED: 'CONNECTED',
      RECONNECTING: 'RECONNECTING',
      DISCONNECTED: 'DISCONNECTED',
    },
  };
});

// Import after mock
import { createTodoStore } from './store';

describe('TodoStore', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    messageHandler = null;
    autocompleteHandler = null;
  });

  it('should apply StateRollup to initialize todos', () => {
    const store = createTodoStore('ws://localhost:8080/ws');
    
    const rollup: StateRollup = {
      type: 'StateRollup',
      todos: [
        { id: '1', name: 'Task 1', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 2000, starred: false },
        { id: '2', name: 'Task 2', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 1000, starred: false },
      ],
      categories: [],
      listTitle: 'My Todo List',
    };
    
    messageHandler!(rollup);
    
    const todos = get(store.todos);
    expect(todos).toHaveLength(2);
    // Should be sorted by sortOrder descending
    expect(todos[0].id).toBe('1');
    expect(todos[1].id).toBe('2');
    
    store.destroy();
  });

  describe('Version handling in StateRollup', () => {
    it('should extract and store version from StateRollup when present', () => {
      const store = createTodoStore('ws://localhost:8080/ws');
      const rollup: StateRollup = {
        type: 'StateRollup',
        todos: [],
        categories: [],
        listTitle: 'Test',
        version: '1.6.0'
      };
      
      messageHandler!(rollup);
      
      const serverVersion = get(store.serverVersion);
      expect(serverVersion).toBe('1.6.0');
      
      store.destroy();
    });

    it('should handle StateRollup without version field (backward compatibility)', () => {
      const store = createTodoStore('ws://localhost:8080/ws');
      const rollup: StateRollup = {
        type: 'StateRollup',
        todos: [],
        categories: [],
        listTitle: 'Test'
        // version field omitted
      };
      
      messageHandler!(rollup);
      
      const serverVersion = get(store.serverVersion);
      expect(serverVersion).toBeNull();
      
      store.destroy();
    });

    it('should handle StateRollup with undefined version', () => {
      const store = createTodoStore('ws://localhost:8080/ws');
      const rollup: StateRollup = {
        type: 'StateRollup',
        todos: [],
        categories: [],
        listTitle: 'Test',
        version: undefined
      };
      
      messageHandler!(rollup);
      
      const serverVersion = get(store.serverVersion);
      expect(serverVersion).toBeNull();
      
      store.destroy();
    });

    it('should update serverVersion when new StateRollup received', () => {
      const store = createTodoStore('ws://localhost:8080/ws');
      
      // First rollup with version
      messageHandler!({
        type: 'StateRollup',
        todos: [],
        categories: [],
        listTitle: 'Test',
        version: '1.6.0'
      });
      
      expect(get(store.serverVersion)).toBe('1.6.0');
      
      // Second rollup with different version
      messageHandler!({
        type: 'StateRollup',
        todos: [],
        categories: [],
        listTitle: 'Test',
        version: '1.7.0'
      });
      
      expect(get(store.serverVersion)).toBe('1.7.0');
      
      store.destroy();
    });
  });

  it('should apply TodoCreated event to add new todo', () => {
    const store = createTodoStore('ws://localhost:8080/ws');
    
    // Initialize with empty rollup
    messageHandler!({ type: 'StateRollup', todos: [], categories: [], listTitle: 'Title' });
    
    const event: TodoCreated = {
      type: 'TodoCreated',
      id: 'new-id',
      name: 'New Task',
      createdAt: '2024-01-01T00:00:00Z',
      sortOrder: 1000,
    };
    
    messageHandler!(event);
    
    const todos = get(store.todos);
    expect(todos).toHaveLength(1);
    expect(todos[0].name).toBe('New Task');
    
    store.destroy();
  });

  it('should load categories from rollup and expose categoryLookup', () => {
    const store = createTodoStore('ws://localhost:8080/ws');

    messageHandler!({
      type: 'StateRollup',
      todos: [{ id: '1', name: 'Task', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 1000, starred: false, categoryId: 'cat-1' }],
      categories: [{ id: 'cat-1', name: 'Work', createdAt: '2024-01-01T00:00:00Z', sortOrder: 1000 }],
      listTitle: 'My Todo List',
    });

    const categories = get(store.categories);
    expect(categories).toHaveLength(1);
    expect(categories[0].name).toBe('Work');
    expect(get(store.categoryLookup).get('cat-1')?.name).toBe('Work');

    store.destroy();
  });

  it('should send categorize todo command', () => {
    const store = createTodoStore('ws://localhost:8080/ws');
    messageHandler!({
      type: 'StateRollup',
      todos: [{ id: '1', name: 'Task', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 1000, starred: false, categoryId: null }],
      categories: [],
      listTitle: 'My Todo List',
    });

    store.categorizeTodo('1', 'cat-2');

    const sentEvent = JSON.parse(mockSend.mock.calls[0][0]);
    expect(sentEvent.type).toBe('CategorizeTodo');
    expect(sentEvent.id).toBe('1');
    expect(sentEvent.categoryId).toBe('cat-2');

    store.destroy();
  });

  it('should apply TodoCompleted event', () => {
    const store = createTodoStore('ws://localhost:8080/ws');
    
    messageHandler!({
      type: 'StateRollup',
      todos: [{ id: '1', name: 'Task', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 1000, starred: false }],
      categories: [],
      listTitle: 'My Todo List',
    });
    
    const completed: TodoCompleted = {
      type: 'TodoCompleted',
      id: '1',
      completedAt: '2024-01-02T00:00:00Z',
    };
    
    messageHandler!(completed);
    
    const todos = get(store.todos);
    expect(todos[0].completedAt).toBe('2024-01-02T00:00:00Z');
    
    store.destroy();
  });

  it('should send optimistic create and update on confirmation', () => {
    const store = createTodoStore('ws://localhost:8080/ws');
    
    messageHandler!({ type: 'StateRollup', todos: [], categories: [], listTitle: 'My Todo List' });
    
    // Create todo optimistically
    store.createTodo('Optimistic task');
    
    // Should immediately appear in store
    let todos = get(store.todos);
    expect(todos).toHaveLength(1);
    expect(todos[0].name).toBe('Optimistic task');
    
    // Should have sent to server
    expect(mockSend).toHaveBeenCalled();
    const sentEvent = JSON.parse(mockSend.mock.calls[0][0]);
    expect(sentEvent.type).toBe('CreateTodo');
    expect(sentEvent.name).toBe('Optimistic task');
    
    // Simulate server confirmation (same event back)
    messageHandler!(sentEvent);
    
    // Should still have one todo
    todos = get(store.todos);
    expect(todos).toHaveLength(1);
    
    store.destroy();
  });

  it('should sort todos by sortOrder descending', () => {
    const store = createTodoStore('ws://localhost:8080/ws');
    
    messageHandler!({
      type: 'StateRollup',
      todos: [
        { id: '1', name: 'Low', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 1000, starred: false },
        { id: '2', name: 'High', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 3000, starred: false },
        { id: '3', name: 'Mid', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 2000, starred: false },
      ],
      categories: [],
      listTitle: 'My Todo List',
    });
    
    const todos = get(store.todos);
    expect(todos[0].name).toBe('High');
    expect(todos[1].name).toBe('Mid');
    expect(todos[2].name).toBe('Low');
    
    store.destroy();
  });

  it('should get highest sortOrder for new todo', () => {
    const store = createTodoStore('ws://localhost:8080/ws');
    
    messageHandler!({
      type: 'StateRollup',
      todos: [
        { id: '1', name: 'Task 1', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 5000, starred: false },
      ],
      categories: [],
      listTitle: 'My Todo List',
    });
    
    store.createTodo('New task');
    
    const sentEvent = JSON.parse(mockSend.mock.calls[0][0]);
    // New task should have sortOrder = highest + 1000
    expect(sentEvent.sortOrder).toBe(6000);
    
    store.destroy();
  });

  it('should toggle todo completion', () => {
    const store = createTodoStore('ws://localhost:8080/ws');
    
    messageHandler!({
      type: 'StateRollup',
      todos: [{ id: '1', name: 'Task', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 1000, starred: false }],
      categories: [],
      listTitle: 'My Todo List'
    });
    
    // Complete the task
    store.toggleComplete('1');
    
    let todos = get(store.todos);
    expect(todos[0].completedAt).not.toBeNull();
    
    const sentEvent = JSON.parse(mockSend.mock.calls[0][0]);
    expect(sentEvent.type).toBe('CompleteTodo');
    
    store.destroy();
  });

  it('should toggle todo star', () => {
    const store = createTodoStore('ws://localhost:8080/ws');
    
    messageHandler!({
      type: 'StateRollup',
      todos: [
        { id: '1', name: 'Task 1', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 1000, starred: false },
        { id: '2', name: 'Task 2', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 2000, starred: false },
      ],
      categories: [],
      listTitle: 'My Todo List'
    });
    
    // Star task 1 - should move to top
    store.toggleStar('1');
    
    let todos = get(store.todos);
    expect(todos[0].id).toBe('1');
    expect(todos[0].starred).toBe(true);
    expect(todos[0].sortOrder).toBe(3000); // highest + 1000
    
    const sentEvent = JSON.parse(mockSend.mock.calls[0][0]);
    expect(sentEvent.type).toBe('StarTodo');
    
    store.destroy();
  });

  it('should reorder todo', () => {
    const store = createTodoStore('ws://localhost:8080/ws');
    
    messageHandler!({
      type: 'StateRollup',
      todos: [
        { id: '1', name: 'Task 1', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 1000, starred: false },
        { id: '2', name: 'Task 2', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 2000, starred: false },
        { id: '3', name: 'Task 3', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 3000, starred: false },
      ],
      categories: [],
      listTitle: 'My Todo List',
    });
    
    // Move task 1 to position between 2 and 3 (sortOrder 2500)
    store.reorder('1', 2500);
    
    const todos = get(store.todos);
    const task1 = todos.find(t => t.id === '1');
    expect(task1?.sortOrder).toBe(2500);
    
    const sentEvent = JSON.parse(mockSend.mock.calls[0][0]);
    expect(sentEvent.type).toBe('ReorderTodo');
    expect(sentEvent.sortOrder).toBe(2500);
    
    store.destroy();
  });

  it('should toggle star (unstarred -> starred)', () => {
    const store = createTodoStore('ws://localhost:8080/ws');
    
    messageHandler!({
      type: 'StateRollup',
      todos: [{
        id: '1',
        name: 'Test',
        createdAt: '2024-01-01T00:00:00Z',
        completedAt: null,
        sortOrder: 1000,
        starred: false,
      }],
      categories: [],
      listTitle: 'My Todo List',
    });

    store.toggleStar('1');

    const sentEvent = JSON.parse(mockSend.mock.calls[0][0]);
    expect(sentEvent.type).toBe('StarTodo');
    expect(sentEvent.id).toBe('1');
    
    store.destroy();
  });

  it('should toggle star (starred -> unstarred)', () => {
    const store = createTodoStore('ws://localhost:8080/ws');
    
    messageHandler!({
      type: 'StateRollup',
      todos: [{
        id: '1',
        name: 'Test',
        createdAt: '2024-01-01T00:00:00Z',
        completedAt: null,
        sortOrder: 1000,
        starred: true,
      }],
      categories: [],
      listTitle: 'My Todo List',
    });

    store.toggleStar('1');

    const sentEvent = JSON.parse(mockSend.mock.calls[0][0]);
    expect(sentEvent.type).toBe('UnstarTodo');
    expect(sentEvent.id).toBe('1');
    
    store.destroy();
  });

  it('should toggle completion (uncompleted -> completed)', () => {
    const store = createTodoStore('ws://localhost:8080/ws');
    
    messageHandler!({
      type: 'StateRollup',
      todos: [{
        id: '1',
        name: 'Test',
        createdAt: '2024-01-01T00:00:00Z',
        completedAt: null,
        sortOrder: 1000,
        starred: false,
      }],
      categories: [],
      listTitle: 'My Todo List',
    });

    store.toggleComplete('1');

    const sentEvent = JSON.parse(mockSend.mock.calls[0][0]);
    expect(sentEvent.type).toBe('CompleteTodo');
    expect(sentEvent.id).toBe('1');
    
    store.destroy();
  });

  it('should toggle completion (completed -> uncompleted)', () => {
    const store = createTodoStore('ws://localhost:8080/ws');
    
    messageHandler!({
      type: 'StateRollup',
      todos: [{
        id: '1',
        name: 'Test',
        createdAt: '2024-01-01T00:00:00Z',
        completedAt: '2024-01-02T00:00:00Z',
        sortOrder: 1000,
        starred: false,
      }],
      categories: [],
      listTitle: 'My Todo List',
    });

    store.toggleComplete('1');

    const sentEvent = JSON.parse(mockSend.mock.calls[0][0]);
    expect(sentEvent.type).toBe('UncompleteTodo');
    expect(sentEvent.id).toBe('1');
    
    store.destroy();
  });

  it('should not toggle star for non-existent todo', () => {
    const store = createTodoStore('ws://localhost:8080/ws');
    
    store.toggleStar('nonexistent');
    
    expect(mockSend).not.toHaveBeenCalled();
    
    store.destroy();
  });

  it('should not toggle completion for non-existent todo', () => {
    const store = createTodoStore('ws://localhost:8080/ws');
    
    store.toggleComplete('nonexistent');
    
    expect(mockSend).not.toHaveBeenCalled();
    
    store.destroy();
  });

  it('should rename todo', () => {
    const store = createTodoStore('ws://localhost:8080/ws');
    
    messageHandler!({
      type: 'StateRollup',
      todos: [{
        id: '1',
        name: 'Old Name',
        createdAt: '2024-01-01T00:00:00Z',
        completedAt: null,
        sortOrder: 1000,
        starred: false,
      }],
      categories: [],
      listTitle: 'My Todo List',
    });

    store.rename('1', 'New Name');

    const sentEvent = JSON.parse(mockSend.mock.calls[0][0]);
    expect(sentEvent.type).toBe('RenameTodo');
    expect(sentEvent.id).toBe('1');
    expect(sentEvent.name).toBe('New Name');
    
    store.destroy();
  });

  it('should separate active and completed todos', () => {
    const store = createTodoStore('ws://localhost:8080/ws');
    
    messageHandler!({
      type: 'StateRollup',
      todos: [
        { id: '1', name: 'Active', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 1000, starred: false },
        { id: '2', name: 'Done', createdAt: '2024-01-01T00:00:00Z', completedAt: '2024-01-02T00:00:00Z', sortOrder: 2000, starred: false },
      ],
      categories: [],
      listTitle: 'My Todo List',
    });
    
    const active = get(store.activeTodos);
    const completed = get(store.completedTodos);
    
    expect(active).toHaveLength(1);
    expect(active[0].name).toBe('Active');
    expect(completed).toHaveLength(1);
    expect(completed[0].name).toBe('Done');
    
    store.destroy();
  });

  describe('Collapsed duplicates (same exact name + category + starred)', () => {
    it('collapses duplicates into a single display item with count', () => {
      const store = createTodoStore('ws://localhost:8080/ws');

      messageHandler!({
        type: 'StateRollup',
        todos: [
          { id: 'a', name: 'Mjölk', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 3000, starred: false, categoryId: 'dairy' },
          { id: 'b', name: 'Mjölk', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 1000, starred: false, categoryId: 'dairy' },
        ],
        categories: [],
        listTitle: 'My Todo List',
      });

      const collapsed = get(store.activeTodosCollapsed);
      expect(collapsed).toHaveLength(1);
      expect(collapsed[0].todo.id).toBe('a'); // highest sortOrder representative
      expect(collapsed[0].count).toBe(2);

      store.destroy();
    });

    it('is case sensitive (does not collapse different casing)', () => {
      const store = createTodoStore('ws://localhost:8080/ws');

      messageHandler!({
        type: 'StateRollup',
        todos: [
          { id: 'a', name: 'Milk', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 2000, starred: false, categoryId: null },
          { id: 'b', name: 'milk', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 1000, starred: false, categoryId: null },
        ],
        categories: [],
        listTitle: 'My Todo List',
      });

      const collapsed = get(store.activeTodosCollapsed);
      expect(collapsed).toHaveLength(2);

      store.destroy();
    });

    it('does not collapse across different categories', () => {
      const store = createTodoStore('ws://localhost:8080/ws');

      messageHandler!({
        type: 'StateRollup',
        todos: [
          { id: 'a', name: 'Bröd', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 2000, starred: false, categoryId: 'cat1' },
          { id: 'b', name: 'Bröd', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 1000, starred: false, categoryId: 'cat2' },
        ],
        categories: [],
        listTitle: 'My Todo List',
      });

      const collapsed = get(store.activeTodosCollapsed);
      expect(collapsed).toHaveLength(2);

      store.destroy();
    });

    it('does not collapse across different star status', () => {
      const store = createTodoStore('ws://localhost:8080/ws');

      messageHandler!({
        type: 'StateRollup',
        todos: [
          { id: 'a', name: 'Pasta', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 2000, starred: true, categoryId: null },
          { id: 'b', name: 'Pasta', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 1000, starred: false, categoryId: null },
        ],
        categories: [],
        listTitle: 'My Todo List',
      });

      const collapsed = get(store.activeTodosCollapsed);
      expect(collapsed).toHaveLength(2);

      store.destroy();
    });

    it('completing a collapsed item completes the lowest sortOrder item (one only)', () => {
      const store = createTodoStore('ws://localhost:8080/ws');

      messageHandler!({
        type: 'StateRollup',
        todos: [
          { id: 'hi', name: 'Mjölk', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 3000, starred: false, categoryId: 'dairy' },
          { id: 'lo', name: 'Mjölk', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 1000, starred: false, categoryId: 'dairy' },
        ],
        categories: [],
        listTitle: 'My Todo List',
      });

      // User clicks the representative (highest), but we must complete the lowest.
      store.toggleComplete('hi');

      const sent = JSON.parse(mockSend.mock.calls[0][0]);
      expect(sent.type).toBe('CompleteTodo');
      expect(sent.id).toBe('lo');

      // Only one todo is completed optimistically.
      const all = get(store.todos);
      const hi = all.find((t) => t.id === 'hi')!;
      const lo = all.find((t) => t.id === 'lo')!;
      expect(hi.completedAt).toBeNull();
      expect(lo.completedAt).not.toBeNull();

      store.destroy();
    });

    it('reordering a collapsed item upwards only reorders the highest sortOrder item', () => {
      const store = createTodoStore('ws://localhost:8080/ws');

      messageHandler!({
        type: 'StateRollup',
        todos: [
          { id: 'hi', name: 'Bröd', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 3000, starred: false, categoryId: null },
          { id: 'lo', name: 'Bröd', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 1000, starred: false, categoryId: null },
        ],
        categories: [],
        listTitle: 'My Todo List',
      });

      store.reorder('hi', 6000); // delta >= 0

      expect(mockSend).toHaveBeenCalledTimes(1);
      const sent = JSON.parse(mockSend.mock.calls[0][0]);
      expect(sent.type).toBe('ReorderTodo');
      expect(sent.id).toBe('hi');
      expect(sent.sortOrder).toBe(6000);

      store.destroy();
    });

    it('reordering a collapsed item downwards shifts the whole duplicate group', () => {
      const store = createTodoStore('ws://localhost:8080/ws');

      messageHandler!({
        type: 'StateRollup',
        todos: [
          { id: 'hi', name: 'Bröd', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 3000, starred: false, categoryId: null },
          { id: 'mid', name: 'Bröd', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 2000, starred: false, categoryId: null },
          { id: 'lo', name: 'Bröd', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 1000, starred: false, categoryId: null },
        ],
        categories: [],
        listTitle: 'My Todo List',
      });

      // Move the group down by 5000 (delta = -5000)
      store.reorder('hi', -2000);

      expect(mockSend).toHaveBeenCalledTimes(3);
      const sentCommands = mockSend.mock.calls.map((c) => JSON.parse(c[0]));
      expect(sentCommands.every((c) => c.type === 'ReorderTodo')).toBe(true);

      const byId = new Map(sentCommands.map((c) => [c.id, c.sortOrder]));
      expect(byId.get('hi')).toBe(-2000);
      expect(byId.get('mid')).toBe(-3000);
      expect(byId.get('lo')).toBe(-4000);

      store.destroy();
    });

    it('collapses per-category in activeTodosByCategoryCollapsed', () => {
      const store = createTodoStore('ws://localhost:8080/ws');

      messageHandler!({
        type: 'StateRollup',
        todos: [
          { id: 'a1', name: 'Mjölk', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 2000, starred: false, categoryId: 'dairy' },
          { id: 'a2', name: 'Mjölk', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 1000, starred: false, categoryId: 'dairy' },
          { id: 'b1', name: 'Mjölk', createdAt: '2024-01-01T00:00:00Z', completedAt: null, sortOrder: 3000, starred: false, categoryId: 'other' },
        ],
        categories: [],
        listTitle: 'My Todo List',
      });

      const map = get(store.activeTodosByCategoryCollapsed);
      const dairy = map.get('dairy')!;
      const other = map.get('other')!;

      expect(dairy).toHaveLength(1);
      expect(dairy[0].count).toBe(2);
      expect(other).toHaveLength(1);
      expect(other[0].count).toBe(1);

      store.destroy();
    });
  });

  it('should sort completed todos by completedAt descending (most recent first)', () => {
    const store = createTodoStore('ws://localhost:8080/ws');
    
    messageHandler!({
      type: 'StateRollup',
      todos: [
        { id: '1', name: 'Completed First', createdAt: '2024-01-01T00:00:00Z', completedAt: '2024-01-02T10:00:00Z', sortOrder: 3000, starred: false },
        { id: '2', name: 'Completed Last', createdAt: '2024-01-01T00:00:00Z', completedAt: '2024-01-02T14:00:00Z', sortOrder: 1000, starred: false },
        { id: '3', name: 'Completed Middle', createdAt: '2024-01-01T00:00:00Z', completedAt: '2024-01-02T12:00:00Z', sortOrder: 2000, starred: false },
      ],
      categories: [],
      listTitle: 'My Todo List',
    });
    
    const completed = get(store.completedTodos);
    
    expect(completed).toHaveLength(3);
    // Should be sorted by completedAt descending (most recent first)
    expect(completed[0].name).toBe('Completed Last'); // 14:00
    expect(completed[1].name).toBe('Completed Middle'); // 12:00
    expect(completed[2].name).toBe('Completed First'); // 10:00
    
    store.destroy();
  });

  describe('Collapsed duplicates in completed todos', () => {
    it('collapses duplicates into a single display item with count', () => {
      const store = createTodoStore('ws://localhost:8080/ws');

      messageHandler!({
        type: 'StateRollup',
        todos: [
          { id: 'a', name: 'Mjölk', createdAt: '2024-01-01T00:00:00Z', completedAt: '2024-01-02T10:00:00Z', sortOrder: 3000, starred: false, categoryId: 'dairy' },
          { id: 'b', name: 'Mjölk', createdAt: '2024-01-01T00:00:00Z', completedAt: '2024-01-02T12:00:00Z', sortOrder: 1000, starred: false, categoryId: 'dairy' },
        ],
        categories: [],
        listTitle: 'My Todo List',
      });

      const collapsed = get(store.completedTodosCollapsed);
      expect(collapsed).toHaveLength(1);
      expect(collapsed[0].count).toBe(2);
      // Should use the most recently completed as representative
      expect(collapsed[0].todo.id).toBe('b'); // completedAt: 12:00 (most recent)

      store.destroy();
    });

    it('maintains completedAt sorting after collapse (most recent first)', () => {
      const store = createTodoStore('ws://localhost:8080/ws');

      messageHandler!({
        type: 'StateRollup',
        todos: [
          { id: 'a', name: 'Mjölk', createdAt: '2024-01-01T00:00:00Z', completedAt: '2024-01-02T10:00:00Z', sortOrder: 3000, starred: false, categoryId: 'dairy' },
          { id: 'b', name: 'Mjölk', createdAt: '2024-01-01T00:00:00Z', completedAt: '2024-01-02T12:00:00Z', sortOrder: 1000, starred: false, categoryId: 'dairy' },
          { id: 'c', name: 'Bröd', createdAt: '2024-01-01T00:00:00Z', completedAt: '2024-01-02T14:00:00Z', sortOrder: 2000, starred: false, categoryId: null },
        ],
        categories: [],
        listTitle: 'My Todo List',
      });

      const collapsed = get(store.completedTodosCollapsed);
      expect(collapsed).toHaveLength(2);
      // Should be sorted by completedAt descending (most recent first)
      expect(collapsed[0].todo.id).toBe('c'); // 14:00 - most recent
      expect(collapsed[1].todo.id).toBe('b'); // 12:00 - representative of Mjölk group

      store.destroy();
    });

    it('is case sensitive (does not collapse different casing)', () => {
      const store = createTodoStore('ws://localhost:8080/ws');

      messageHandler!({
        type: 'StateRollup',
        todos: [
          { id: 'a', name: 'Milk', createdAt: '2024-01-01T00:00:00Z', completedAt: '2024-01-02T10:00:00Z', sortOrder: 2000, starred: false, categoryId: null },
          { id: 'b', name: 'milk', createdAt: '2024-01-01T00:00:00Z', completedAt: '2024-01-02T12:00:00Z', sortOrder: 1000, starred: false, categoryId: null },
        ],
        categories: [],
        listTitle: 'My Todo List',
      });

      const collapsed = get(store.completedTodosCollapsed);
      expect(collapsed).toHaveLength(2);

      store.destroy();
    });

    it('does not collapse across different categories', () => {
      const store = createTodoStore('ws://localhost:8080/ws');

      messageHandler!({
        type: 'StateRollup',
        todos: [
          { id: 'a', name: 'Bröd', createdAt: '2024-01-01T00:00:00Z', completedAt: '2024-01-02T10:00:00Z', sortOrder: 2000, starred: false, categoryId: 'cat1' },
          { id: 'b', name: 'Bröd', createdAt: '2024-01-01T00:00:00Z', completedAt: '2024-01-02T12:00:00Z', sortOrder: 1000, starred: false, categoryId: 'cat2' },
        ],
        categories: [],
        listTitle: 'My Todo List',
      });

      const collapsed = get(store.completedTodosCollapsed);
      expect(collapsed).toHaveLength(2);

      store.destroy();
    });

    it('does not collapse across different star status', () => {
      const store = createTodoStore('ws://localhost:8080/ws');

      messageHandler!({
        type: 'StateRollup',
        todos: [
          { id: 'a', name: 'Pasta', createdAt: '2024-01-01T00:00:00Z', completedAt: '2024-01-02T10:00:00Z', sortOrder: 2000, starred: true, categoryId: null },
          { id: 'b', name: 'Pasta', createdAt: '2024-01-01T00:00:00Z', completedAt: '2024-01-02T12:00:00Z', sortOrder: 1000, starred: false, categoryId: null },
        ],
        categories: [],
        listTitle: 'My Todo List',
      });

      const collapsed = get(store.completedTodosCollapsed);
      expect(collapsed).toHaveLength(2);

      store.destroy();
    });

    it('uncompleting a collapsed item uncompletes the most recently completed item (one only)', () => {
      const store = createTodoStore('ws://localhost:8080/ws');

      messageHandler!({
        type: 'StateRollup',
        todos: [
          { id: 'old', name: 'Mjölk', createdAt: '2024-01-01T00:00:00Z', completedAt: '2024-01-02T10:00:00Z', sortOrder: 1000, starred: false, categoryId: 'dairy' },
          { id: 'recent', name: 'Mjölk', createdAt: '2024-01-01T00:00:00Z', completedAt: '2024-01-02T14:00:00Z', sortOrder: 3000, starred: false, categoryId: 'dairy' },
        ],
        categories: [],
        listTitle: 'My Todo List',
      });

      // The collapsed item shows 'recent' as representative (most recent completedAt)
      // When uncompleting, we should uncomplete 'recent' (the most recently completed)
      store.toggleComplete('recent');

      const sent = JSON.parse(mockSend.mock.calls[0][0]);
      expect(sent.type).toBe('UncompleteTodo');
      expect(sent.id).toBe('recent');

      // Only one todo is uncompleted optimistically
      const all = get(store.todos);
      const old = all.find((t) => t.id === 'old')!;
      const recent = all.find((t) => t.id === 'recent')!;
      expect(old.completedAt).not.toBeNull(); // Still completed
      expect(recent.completedAt).toBeNull(); // Uncompleted

      store.destroy();
    });

    it('handles 3x duplicates correctly', () => {
      const store = createTodoStore('ws://localhost:8080/ws');

      messageHandler!({
        type: 'StateRollup',
        todos: [
          { id: 'a', name: 'Mjölk', createdAt: '2024-01-01T00:00:00Z', completedAt: '2024-01-02T10:00:00Z', sortOrder: 1000, starred: false, categoryId: 'dairy' },
          { id: 'b', name: 'Mjölk', createdAt: '2024-01-01T00:00:00Z', completedAt: '2024-01-02T12:00:00Z', sortOrder: 2000, starred: false, categoryId: 'dairy' },
          { id: 'c', name: 'Mjölk', createdAt: '2024-01-01T00:00:00Z', completedAt: '2024-01-02T14:00:00Z', sortOrder: 3000, starred: false, categoryId: 'dairy' },
        ],
        categories: [],
        listTitle: 'My Todo List',
      });

      const collapsed = get(store.completedTodosCollapsed);
      expect(collapsed).toHaveLength(1);
      expect(collapsed[0].count).toBe(3);
      expect(collapsed[0].todo.id).toBe('c'); // Most recently completed

      store.destroy();
    });

    it('uncompleting from a 3x group uncompletes only the most recent', () => {
      const store = createTodoStore('ws://localhost:8080/ws');

      messageHandler!({
        type: 'StateRollup',
        todos: [
          { id: 'old', name: 'Mjölk', createdAt: '2024-01-01T00:00:00Z', completedAt: '2024-01-02T10:00:00Z', sortOrder: 1000, starred: false, categoryId: 'dairy' },
          { id: 'mid', name: 'Mjölk', createdAt: '2024-01-01T00:00:00Z', completedAt: '2024-01-02T12:00:00Z', sortOrder: 2000, starred: false, categoryId: 'dairy' },
          { id: 'recent', name: 'Mjölk', createdAt: '2024-01-01T00:00:00Z', completedAt: '2024-01-02T14:00:00Z', sortOrder: 3000, starred: false, categoryId: 'dairy' },
        ],
        categories: [],
        listTitle: 'My Todo List',
      });

      // Uncomplete the collapsed item (represented by 'recent')
      store.toggleComplete('recent');

      const sent = JSON.parse(mockSend.mock.calls[0][0]);
      expect(sent.type).toBe('UncompleteTodo');
      expect(sent.id).toBe('recent');

      // Only 'recent' should be uncompleted
      const all = get(store.todos);
      expect(all.find((t) => t.id === 'old')!.completedAt).not.toBeNull();
      expect(all.find((t) => t.id === 'mid')!.completedAt).not.toBeNull();
      expect(all.find((t) => t.id === 'recent')!.completedAt).toBeNull();

      store.destroy();
    });
  });

  it('should handle list title changes', () => {
    const store = createTodoStore('ws://localhost:8080/ws');
    
    // Send initial rollup with title
    messageHandler!({
      type: 'StateRollup',
      todos: [],
      categories: [],
      listTitle: 'Initial Title',
    } as StateRollup);
    
    expect(get(store.listTitle)).toBe('Initial Title');
    
    // Send ListTitleChanged event
    messageHandler!({
      type: 'ListTitleChanged',
      title: 'New Title',
    });
    
    expect(get(store.listTitle)).toBe('New Title');
    
    store.destroy();
  });

  it('should send SetListTitle command', () => {
    const store = createTodoStore('ws://localhost:8080/ws');
    
    store.setListTitle('Shopping List');
    
    expect(mockSend).toHaveBeenCalledTimes(1);
    const sentCommand = JSON.parse(mockSend.mock.calls[0][0]);
    expect(sentCommand.type).toBe('SetListTitle');
    expect(sentCommand.title).toBe('Shopping List');
    
    store.destroy();
  });

  it('should handle list title with optimistic updates', () => {
    const store = createTodoStore('ws://localhost:8080/ws');
    
    // Send initial rollup
    messageHandler!({
      type: 'StateRollup',
      todos: [],
      categories: [],
      listTitle: 'Old Title',
    } as StateRollup);
    
    expect(get(store.listTitle)).toBe('Old Title');
    
    // Change title (optimistic update happens)
    store.setListTitle('New Title');
    
    // Should be updated immediately (optimistic)
    expect(get(store.listTitle)).toBe('New Title');
    
    store.destroy();
  });

  // Autocomplete tests
  describe('Autocomplete', () => {
    it('should send autocomplete request', () => {
      const store = createTodoStore('ws://localhost:8080/ws');
      
      store.requestAutocomplete('mil');
      
      expect(mockSendAutocomplete).toHaveBeenCalledTimes(1);
      const call = mockSendAutocomplete.mock.calls[0][0];
      expect(call.query).toBe('mil');
      expect(call.requestId).toBeDefined();
      
      store.destroy();
    });

    it('should update autocompleteSuggestions on response', () => {
      const store = createTodoStore('ws://localhost:8080/ws');
      
      // Request autocomplete
      store.requestAutocomplete('mil');
      const requestId = mockSendAutocomplete.mock.calls[0][0].requestId;
      
      // Simulate response
      autocompleteHandler!({
        type: 'AutocompleteResponse',
        suggestions: [
          { name: 'Milk', categoryId: null, categoryName: null },
          { name: 'Milo', categoryId: null, categoryName: null },
        ] as AutocompleteSuggestion[],
        requestId: requestId,
      });
      
      const suggestions = get(store.autocompleteSuggestions);
      expect(suggestions.map((s) => s.name)).toEqual(['Milk', 'Milo']);
      
      store.destroy();
    });

    it('should ignore response with wrong requestId', () => {
      const store = createTodoStore('ws://localhost:8080/ws');
      
      // Request autocomplete
      store.requestAutocomplete('mil');
      
      // Simulate response with wrong requestId
      autocompleteHandler!({
        type: 'AutocompleteResponse',
        suggestions: [
          { name: 'Wrong', categoryId: null, categoryName: null },
          { name: 'Response', categoryId: null, categoryName: null },
        ] as AutocompleteSuggestion[],
        requestId: 'wrong-id',
      });
      
      // Should not update
      const suggestions = get(store.autocompleteSuggestions);
      expect(suggestions).toEqual([]);
      
      store.destroy();
    });

    it('should clear autocomplete suggestions', () => {
      const store = createTodoStore('ws://localhost:8080/ws');
      
      // Request and receive autocomplete
      store.requestAutocomplete('mil');
      const requestId = mockSendAutocomplete.mock.calls[0][0].requestId;
      autocompleteHandler!({
        type: 'AutocompleteResponse',
        suggestions: [{ name: 'Milk', categoryId: null, categoryName: null }],
        requestId: requestId,
      });
      
      expect(get(store.autocompleteSuggestions).map((s) => s.name)).toEqual(['Milk']);
      
      // Clear autocomplete
      store.clearAutocomplete();
      
      expect(get(store.autocompleteSuggestions)).toEqual([]);
      
      store.destroy();
    });

    it('should handle multiple rapid requests (only latest matters)', () => {
      const store = createTodoStore('ws://localhost:8080/ws');
      
      // Send multiple requests rapidly
      store.requestAutocomplete('m');
      const firstRequestId = mockSendAutocomplete.mock.calls[0][0].requestId;
      
      store.requestAutocomplete('mi');
      const secondRequestId = mockSendAutocomplete.mock.calls[1][0].requestId;
      
      store.requestAutocomplete('mil');
      const thirdRequestId = mockSendAutocomplete.mock.calls[2][0].requestId;
      
      // Response from first request arrives (stale)
      autocompleteHandler!({
        type: 'AutocompleteResponse',
        suggestions: [{ name: 'Meat', categoryId: null, categoryName: null }],
        requestId: firstRequestId,
      });
      
      // Should not update (stale response)
      expect(get(store.autocompleteSuggestions)).toEqual([]);
      
      // Response from third request arrives
      autocompleteHandler!({
        type: 'AutocompleteResponse',
        suggestions: [{ name: 'Milk', categoryId: null, categoryName: null }],
        requestId: thirdRequestId,
      });
      
      // Should update with latest
      expect(get(store.autocompleteSuggestions).map((s) => s.name)).toEqual(['Milk']);
      
      store.destroy();
    });

    it('should send autocomplete request with empty string', () => {
      const store = createTodoStore('ws://localhost:8080/ws');
      
      store.requestAutocomplete('');
      
      expect(mockSendAutocomplete).toHaveBeenCalledTimes(1);
      const call = mockSendAutocomplete.mock.calls[0][0];
      expect(call.query).toBe('');
      
      store.destroy();
    });
  });

  describe('Trimming whitespace', () => {
    it('should trim whitespace from createTodo', () => {
      const store = createTodoStore('ws://localhost:8080/ws');
      
      messageHandler!({ type: 'StateRollup', todos: [], categories: [], listTitle: 'My Todo List' });
      
      // Test trimming leading and trailing spaces
      store.createTodo('  Test task  ');
      
      const sentEvent = JSON.parse(mockSend.mock.calls[0][0]);
      expect(sentEvent.type).toBe('CreateTodo');
      expect(sentEvent.name).toBe('Test task');
      
      // Test trimming only leading spaces
      store.createTodo('  Leading spaces');
      const sentEvent2 = JSON.parse(mockSend.mock.calls[1][0]);
      expect(sentEvent2.name).toBe('Leading spaces');
      
      // Test trimming only trailing spaces
      store.createTodo('Trailing spaces  ');
      const sentEvent3 = JSON.parse(mockSend.mock.calls[2][0]);
      expect(sentEvent3.name).toBe('Trailing spaces');
      
      store.destroy();
    });

    it('should not create todo with only whitespace', () => {
      const store = createTodoStore('ws://localhost:8080/ws');
      
      messageHandler!({ type: 'StateRollup', todos: [], categories: [], listTitle: 'My Todo List' });
      
      const initialCallCount = mockSend.mock.calls.length;
      
      // Should not send command for whitespace-only strings
      store.createTodo('   ');
      store.createTodo('\t\t');
      store.createTodo('\n\n');
      
      expect(mockSend.mock.calls.length).toBe(initialCallCount);
      
      store.destroy();
    });

    it('should trim whitespace from createCategory', async () => {
      const store = createTodoStore('ws://localhost:8080/ws');
      
      messageHandler!({ type: 'StateRollup', todos: [], categories: [], listTitle: 'My Todo List' });
      
      // Test trimming leading and trailing spaces
      const promise = store.createCategory('  Test category  ');
      
      const sentEvent = JSON.parse(mockSend.mock.calls[0][0]);
      expect(sentEvent.type).toBe('CreateCategory');
      expect(sentEvent.name).toBe('Test category');
      
      // Simulate server success
      messageHandler!({
        type: 'CommandResponse',
        commandId: sentEvent.commandId,
        success: true,
      });
      
      await promise;
      
      store.destroy();
    });

    it('should reject createCategory with only whitespace', async () => {
      const store = createTodoStore('ws://localhost:8080/ws');
      
      messageHandler!({ type: 'StateRollup', todos: [], categories: [], listTitle: 'My Todo List' });
      
      await expect(store.createCategory('   ')).rejects.toBe('Category name cannot be empty');
      await expect(store.createCategory('\t\t')).rejects.toBe('Category name cannot be empty');
      
      expect(mockSend).not.toHaveBeenCalled();
      
      store.destroy();
    });

    it('should trim whitespace from rename', () => {
      const store = createTodoStore('ws://localhost:8080/ws');
      
      messageHandler!({
        type: 'StateRollup',
        todos: [{
          id: '1',
          name: 'Old Name',
          createdAt: '2024-01-01T00:00:00Z',
          completedAt: null,
          sortOrder: 1000,
          starred: false,
        }],
        categories: [],
        listTitle: 'My Todo List',
      });

      // Test trimming leading and trailing spaces
      store.rename('1', '  New Name  ');
      
      const sentEvent = JSON.parse(mockSend.mock.calls[0][0]);
      expect(sentEvent.type).toBe('RenameTodo');
      expect(sentEvent.name).toBe('New Name');
      
      store.destroy();
    });

    it('should not rename todo with only whitespace', () => {
      const store = createTodoStore('ws://localhost:8080/ws');
      
      messageHandler!({
        type: 'StateRollup',
        todos: [{
          id: '1',
          name: 'Old Name',
          createdAt: '2024-01-01T00:00:00Z',
          completedAt: null,
          sortOrder: 1000,
          starred: false,
        }],
        categories: [],
        listTitle: 'My Todo List',
      });

      const initialCallCount = mockSend.mock.calls.length;
      
      // Should not send command for whitespace-only strings
      store.rename('1', '   ');
      store.rename('1', '\t\t');
      
      expect(mockSend.mock.calls.length).toBe(initialCallCount);
      
      store.destroy();
    });

    it('should trim whitespace from renameCategory', async () => {
      const store = createTodoStore('ws://localhost:8080/ws');
      
      messageHandler!({
        type: 'StateRollup',
        todos: [],
        categories: [{
          id: 'cat-1',
          name: 'Old Name',
          createdAt: '2024-01-01T00:00:00Z',
          sortOrder: 1000,
        }],
        listTitle: 'My Todo List',
      });

      // Test trimming leading and trailing spaces
      const promise = store.renameCategory('cat-1', '  New Name  ');
      
      const sentEvent = JSON.parse(mockSend.mock.calls[0][0]);
      expect(sentEvent.type).toBe('RenameCategory');
      expect(sentEvent.name).toBe('New Name');
      
      // Simulate server success
      messageHandler!({
        type: 'CommandResponse',
        commandId: sentEvent.commandId,
        success: true,
      });
      
      await promise;
      
      store.destroy();
    });

    it('should reject renameCategory with only whitespace', async () => {
      const store = createTodoStore('ws://localhost:8080/ws');
      
      messageHandler!({
        type: 'StateRollup',
        todos: [],
        categories: [{
          id: 'cat-1',
          name: 'Old Name',
          createdAt: '2024-01-01T00:00:00Z',
          sortOrder: 1000,
        }],
        listTitle: 'My Todo List',
      });

      await expect(store.renameCategory('cat-1', '   ')).rejects.toBe('Category name cannot be empty');
      
      expect(mockSend).not.toHaveBeenCalled();
      
      store.destroy();
    });

    it('should trim whitespace from setListTitle', () => {
      const store = createTodoStore('ws://localhost:8080/ws');
      
      messageHandler!({ type: 'StateRollup', todos: [], categories: [], listTitle: 'My Todo List' });

      // Test trimming leading and trailing spaces
      store.setListTitle('  New Title  ');
      
      const sentEvent = JSON.parse(mockSend.mock.calls[0][0]);
      expect(sentEvent.type).toBe('SetListTitle');
      expect(sentEvent.title).toBe('New Title');
      
      // Test trimming only leading spaces
      store.setListTitle('  Leading spaces');
      const sentEvent2 = JSON.parse(mockSend.mock.calls[1][0]);
      expect(sentEvent2.title).toBe('Leading spaces');
      
      // Test trimming only trailing spaces
      store.setListTitle('Trailing spaces  ');
      const sentEvent3 = JSON.parse(mockSend.mock.calls[2][0]);
      expect(sentEvent3.title).toBe('Trailing spaces');
      
      store.destroy();
    });
  });
});

