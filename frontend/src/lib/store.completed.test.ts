import { describe, it, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';
import type { ServerMessage, Todo } from './types';

let messageHandler: ((msg: ServerMessage) => void) | null = null;
const requestStateRollup = vi.fn();

vi.mock('./websocket', () => ({
  TodoWebSocket: class {
    constructor(_url: string) {}
    send() {}
    sendAutocompleteRequest() {}
    requestStateRollup(includeAll: boolean) { requestStateRollup(includeAll); }
    onMessage(handler: (msg: ServerMessage) => void) { messageHandler = handler; return () => {}; }
    onAutocomplete() { return () => {}; }
    onConnectionChange() { return () => {}; }
    close() {}
  },
  ConnectionState: { CONNECTING: 'CONNECTING', CONNECTED: 'CONNECTED', RECONNECTING: 'RECONNECTING', DISCONNECTED: 'DISCONNECTED' },
}));

import { createTodoStore, COMPLETED_LIMIT } from './store';

function todo(id: string, completedMinute: number | null): Todo {
  return {
    id,
    name: id,
    createdAt: '2026-01-01T00:00:00Z',
    completedAt: completedMinute === null ? null : new Date(Date.UTC(2026, 0, 2, 0, completedMinute)).toISOString(),
    sortOrder: 1000,
    starred: false,
    categoryId: null,
  };
}

function completed(n: number, offset = 0): Todo[] {
  return Array.from({ length: n }, (_, i) => todo(`done-${i + offset}`, i + offset));
}

function rollup(todos: Todo[], completedTotal?: number) {
  messageHandler!({ type: 'StateRollup', todos, categories: [], listTitle: 'List', completedTotal });
}

describe('completed history', () => {
  beforeEach(() => {
    requestStateRollup.mockClear();
    messageHandler = null;
  });

  it('counts completed items the server did not send and offers to load them', () => {
    const store = createTodoStore('ws://test');
    rollup([todo('open', null), ...completed(COMPLETED_LIMIT, 10)], COMPLETED_LIMIT + 10);

    expect(get(store.completedTotal)).toBe(COMPLETED_LIMIT + 10);
    expect(get(store.completedTodosCollapsed)).toHaveLength(COMPLETED_LIMIT);
    expect(get(store.hasMoreCompleted)).toBe(true);

    store.loadAllCompleted();
    expect(requestStateRollup).toHaveBeenCalledWith(true);

    rollup([todo('open', null), ...completed(COMPLETED_LIMIT + 10)], COMPLETED_LIMIT + 10);
    expect(get(store.completedTodosCollapsed)).toHaveLength(COMPLETED_LIMIT + 10);
    expect(get(store.completedTotal)).toBe(COMPLETED_LIMIT + 10);
    expect(get(store.hasMoreCompleted)).toBe(false);
  });

  it('returns to the newest items when capped after showing all, without refetching', () => {
    const store = createTodoStore('ws://test');
    rollup([...completed(COMPLETED_LIMIT, 10)], COMPLETED_LIMIT + 10);
    store.loadAllCompleted();
    rollup(completed(COMPLETED_LIMIT + 10), COMPLETED_LIMIT + 10);
    requestStateRollup.mockClear();

    store.capCompleted();

    const shown = get(store.completedTodosCollapsed);
    expect(shown).toHaveLength(COMPLETED_LIMIT);
    expect(shown[0].todo.id).toBe(`done-${COMPLETED_LIMIT + 9}`);
    expect(get(store.completedTotal)).toBe(COMPLETED_LIMIT + 10);
    expect(get(store.hasMoreCompleted)).toBe(true);

    store.loadAllCompleted();
    expect(requestStateRollup).not.toHaveBeenCalled();
    expect(get(store.completedTodosCollapsed)).toHaveLength(COMPLETED_LIMIT + 10);
  });

  it('has nothing more to load when every completed item fits', () => {
    const store = createTodoStore('ws://test');
    rollup(completed(3), 3);
    expect(get(store.completedTotal)).toBe(3);
    expect(get(store.hasMoreCompleted)).toBe(false);
  });

  it('caps items completed during the session to the newest until all are shown', () => {
    const store = createTodoStore('ws://test');
    rollup([...completed(COMPLETED_LIMIT), todo('late', null)], COMPLETED_LIMIT);

    messageHandler!({ type: 'TodoCompleted', id: 'late', completedAt: '2026-02-01T00:00:00Z' } as ServerMessage);

    const shown = get(store.completedTodosCollapsed);
    expect(shown).toHaveLength(COMPLETED_LIMIT);
    expect(shown[0].todo.id).toBe('late');
    expect(get(store.completedTotal)).toBe(COMPLETED_LIMIT + 1);
    expect(get(store.hasMoreCompleted)).toBe(true);

    store.loadAllCompleted();
    expect(requestStateRollup).not.toHaveBeenCalled();
    expect(get(store.completedTodosCollapsed)).toHaveLength(COMPLETED_LIMIT + 1);
  });

  it('refetches when another client uncompletes an item this client has not loaded', () => {
    const store = createTodoStore('ws://test');
    rollup(completed(COMPLETED_LIMIT, 1), COMPLETED_LIMIT + 1);

    messageHandler!({ type: 'TodoUncompleted', id: 'done-0' } as ServerMessage);

    expect(requestStateRollup).toHaveBeenCalledWith(false);
    expect(get(store.completedTotal)).toBe(COMPLETED_LIMIT + 1);
  });

  it('treats a rollup without completedTotal as complete', () => {
    const store = createTodoStore('ws://test');
    rollup(completed(2));
    expect(get(store.completedTotal)).toBe(2);
    expect(get(store.hasMoreCompleted)).toBe(false);
  });
});
