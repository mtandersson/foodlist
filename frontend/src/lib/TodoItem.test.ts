import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { cleanup, fireEvent, render, screen } from '@testing-library/svelte';
import type { Todo } from './types';
import TodoItem from './TodoItem.svelte';

describe('TodoItem', () => {
  let mockTodo: Todo;

  beforeEach(() => {
    mockTodo = {
      id: '1',
      name: 'Test Todo',
      createdAt: '2024-01-01',
      completedAt: null,
      sortOrder: 1000,
      starred: false,
      categoryId: null, // Uncategorized
    };
  });

  describe('Duplicate badge rendering', () => {
    it('does not show a badge when duplicateCount is 1 (default)', () => {
      render(TodoItem, {
        props: {
          todo: mockTodo,
          categoryName: null,
          onToggleComplete: vi.fn(),
          onToggleStar: vi.fn(),
          onRename: vi.fn(),
        },
      });

      expect(screen.queryByText('1x')).toBeNull();
      expect(screen.queryByText('2x')).toBeNull();
    });

    it('shows a 2x badge directly after the name when duplicateCount is 2', () => {
      render(TodoItem, {
        props: {
          todo: mockTodo,
          duplicateCount: 2,
          categoryName: null,
          onToggleComplete: vi.fn(),
          onToggleStar: vi.fn(),
          onRename: vi.fn(),
        },
      });

      expect(screen.getByText('Test Todo')).toBeInTheDocument();
      expect(screen.getByText('2x')).toBeInTheDocument();
    });

    it('shows a 3x badge when duplicateCount is 3', () => {
      render(TodoItem, {
        props: {
          todo: mockTodo,
          duplicateCount: 3,
          categoryName: null,
          onToggleComplete: vi.fn(),
          onToggleStar: vi.fn(),
          onRename: vi.fn(),
        },
      });

      expect(screen.getByText('3x')).toBeInTheDocument();
    });
  });

  describe('touch gestures', () => {
    beforeEach(() => vi.useFakeTimers());
    afterEach(() => {
      cleanup();
      vi.useRealTimers();
    });

    function renderItem(categoryId: string | null = null) {
      const todo = { ...mockTodo, categoryId };
      const onRequestCategorize = vi.fn();
      const onRename = vi.fn();
      const view = render(TodoItem, {
        props: {
          todo,
          onToggleComplete: vi.fn(),
          onToggleStar: vi.fn(),
          onRename,
          onRequestCategorize,
        },
      });
      const button = screen.getByRole('button', { name: 'Double-click or long-press to edit' });
      return { ...view, todo, button, onRequestCategorize, onRename };
    }

    it.each([null, 'fruit'])('categorizes a quick tap for category %s without entering edit mode', async (categoryId) => {
      const { button, todo, onRequestCategorize } = renderItem(categoryId);
      await fireEvent.touchStart(button);
      await vi.advanceTimersByTimeAsync(100);
      await fireEvent.touchEnd(button);
      await vi.advanceTimersByTimeAsync(500);
      expect(onRequestCategorize).toHaveBeenCalledExactlyOnceWith(todo);
      expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
      expect(button).not.toHaveClass('long-pressing');
    });

    it('edits after a stationary long press and saves the new name', async () => {
      const { button, onRequestCategorize, onRename } = renderItem();
      await fireEvent.touchStart(button);
      await vi.advanceTimersByTimeAsync(499);
      expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
      await vi.advanceTimersByTimeAsync(1);
      const input = screen.getByRole('textbox');
      expect(input).toHaveValue('Test Todo');
      await fireEvent.touchEnd(button);
      expect(onRequestCategorize).not.toHaveBeenCalled();
      await fireEvent.input(input, { target: { value: 'Fresh fruit' } });
      await fireEvent.keyDown(input, { key: 'Enter' });
      expect(onRename).toHaveBeenCalledExactlyOnceWith('1', 'Fresh fruit');
    });

    it('does not edit or categorize when a touch moves to scroll', async () => {
      const { button, onRequestCategorize } = renderItem();
      await fireEvent.touchStart(button);
      await vi.advanceTimersByTimeAsync(100);
      await fireEvent.touchMove(button);
      await fireEvent.touchEnd(button);
      await vi.advanceTimersByTimeAsync(500);
      expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
      expect(onRequestCategorize).not.toHaveBeenCalled();
      expect(button).not.toHaveClass('long-pressing');
    });

    it.each([0, 100, 499])('discards a touch canceled after %sms', async (delay) => {
      const { button, onRequestCategorize, onRename } = renderItem();
      await fireEvent.touchStart(button);
      await vi.advanceTimersByTimeAsync(delay);
      await fireEvent.touchCancel(button);
      expect(button).not.toHaveClass('long-pressing');
      await vi.advanceTimersByTimeAsync(600);
      expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
      expect(onRequestCategorize).not.toHaveBeenCalled();
      expect(onRename).not.toHaveBeenCalled();
    });

    it('allows a fresh quick tap after canceling a previous touch', async () => {
      const { button, todo, onRequestCategorize } = renderItem();
      await fireEvent.touchStart(button);
      await vi.advanceTimersByTimeAsync(400);
      await fireEvent.touchCancel(button);
      await fireEvent.touchStart(button);
      await vi.advanceTimersByTimeAsync(100);
      await fireEvent.touchEnd(button);
      await vi.advanceTimersByTimeAsync(600);
      expect(onRequestCategorize).toHaveBeenCalledExactlyOnceWith(todo);
      expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
    });

    it('ignores a touchend from a canceled gesture', async () => {
      const { button, onRequestCategorize } = renderItem();
      await fireEvent.touchStart(button);
      await vi.advanceTimersByTimeAsync(100);
      await fireEvent.touchCancel(button);
      await fireEvent.touchEnd(button);
      await vi.advanceTimersByTimeAsync(600);
      expect(onRequestCategorize).not.toHaveBeenCalled();
      expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
    });

    it('releases the pending gesture timer when its row is removed', async () => {
      const { button, unmount, onRequestCategorize, onRename } = renderItem();
      await fireEvent.touchStart(button);
      await unmount();
      expect(vi.getTimerCount()).toBe(0);
      await vi.advanceTimersByTimeAsync(600);
      expect(onRequestCategorize).not.toHaveBeenCalled();
      expect(onRename).not.toHaveBeenCalled();
    });

    it('still supports desktop double-click editing', async () => {
      const { button } = renderItem();
      await fireEvent.doubleClick(button);
      expect(screen.getByRole('textbox')).toHaveValue('Test Todo');
    });
  });
});

describe('TodoItem check-off', () => {
  afterEach(() => {
    vi.useRealTimers();
    cleanup();
  });

  function renderOpen() {
    const onToggleComplete = vi.fn();
    const todo: Todo = { id: 't1', name: 'Mjölk', createdAt: '2026-11-24', completedAt: null, sortOrder: 1, starred: false };
    render(TodoItem, { props: { todo, onToggleComplete, onToggleStar: vi.fn(), onRename: vi.fn() } });
    return { onToggleComplete };
  }

  it('shows the item as checked in place before completing it', async () => {
    vi.useFakeTimers();
    const { onToggleComplete } = renderOpen();
    await fireEvent.click(screen.getByRole('button', { name: 'Mark as complete' }));
    expect(onToggleComplete).not.toHaveBeenCalled();
    expect(document.querySelector('.checkbox-ring')).toHaveClass('checked');
    vi.advanceTimersByTime(400);
    expect(onToggleComplete).toHaveBeenCalledExactlyOnceWith('t1');
  });

  it('shows a completed item as unchecked in place before reopening it', async () => {
    vi.useFakeTimers();
    const onToggleComplete = vi.fn();
    const todo: Todo = { id: 't2', name: 'Lime', createdAt: '2026-11-24', completedAt: '2026-11-24', sortOrder: 1, starred: false };
    render(TodoItem, { props: { todo, onToggleComplete, onToggleStar: vi.fn(), onRename: vi.fn() } });
    await fireEvent.click(screen.getByRole('button', { name: 'Mark as incomplete' }));
    expect(onToggleComplete).not.toHaveBeenCalled();
    expect(document.querySelector('.checkbox-ring')).not.toHaveClass('checked');
    vi.advanceTimersByTime(400);
    expect(onToggleComplete).toHaveBeenCalledExactlyOnceWith('t2');
  });

  it('ignores repeat taps while the check is settling', async () => {
    vi.useFakeTimers();
    const { onToggleComplete } = renderOpen();
    const box = screen.getByRole('button', { name: 'Mark as complete' });
    await fireEvent.click(box);
    await fireEvent.click(box);
    vi.advanceTimersByTime(400);
    expect(onToggleComplete).toHaveBeenCalledOnce();
  });

  it('still completes the item if the row is removed mid-animation', async () => {
    vi.useFakeTimers();
    const { onToggleComplete } = renderOpen();
    await fireEvent.click(screen.getByRole('button', { name: 'Mark as complete' }));
    cleanup();
    expect(onToggleComplete).toHaveBeenCalledExactlyOnceWith('t1');
  });
});
