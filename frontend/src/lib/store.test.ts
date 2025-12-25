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
    
    messageHandler!({ type: 'StateRollup', todos: [] });
    
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
});

