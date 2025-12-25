<script lang="ts">
  import type { Todo } from './types';

  interface Props {
    todo: Todo;
    categoryName?: string | null;
    onToggleComplete: (id: string) => void;
    onToggleStar: (id: string) => void;
    onRename: (id: string, name: string) => void;
  }

  let { todo, categoryName = null, onToggleComplete, onToggleStar, onRename }: Props = $props();

  let isEditing = $state(false);
  let editName = $state('');
  let longPressTimer: number | null = $state(null);
  let isLongPressing = $state(false);

  function handleCheckClick() {
    onToggleComplete(todo.id);
  }

  function handleStarClick() {
    onToggleStar(todo.id);
  }

  function startEditing() {
    editName = todo.name;
    isEditing = true;
    isLongPressing = false;
  }

  function finishEditing() {
    if (editName.trim() && editName !== todo.name) {
      onRename(todo.id, editName.trim());
    }
    isEditing = false;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      finishEditing();
    } else if (e.key === 'Escape') {
      editName = todo.name;
      isEditing = false;
    }
  }

  // Mobile long-press support
  function handleTouchStart(e: TouchEvent) {
    // Clear any existing timer
    if (longPressTimer) {
      clearTimeout(longPressTimer);
    }
    
    isLongPressing = true;
    
    // Start long-press timer (500ms)
    longPressTimer = window.setTimeout(() => {
      startEditing();
      longPressTimer = null;
    }, 500);
  }

  function handleTouchEnd(e: TouchEvent) {
    // Clear timer if touch ends before long-press threshold
    if (longPressTimer) {
      clearTimeout(longPressTimer);
      longPressTimer = null;
    }
    isLongPressing = false;
  }

  function handleTouchMove(e: TouchEvent) {
    // Cancel long-press if user moves finger
    if (longPressTimer) {
      clearTimeout(longPressTimer);
      longPressTimer = null;
    }
    isLongPressing = false;
  }
</script>

<div class="todo-item" class:completed={todo.completedAt !== null}>
  <button 
    class="checkbox" 
    class:checked={todo.completedAt !== null}
    onclick={handleCheckClick}
    aria-label={todo.completedAt ? 'Mark as incomplete' : 'Mark as complete'}
  >
    {#if todo.completedAt !== null}
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
        <polyline points="20 6 9 17 4 12"></polyline>
      </svg>
    {/if}
  </button>

  {#if isEditing}
    <input
      type="text"
      class="edit-input"
      bind:value={editName}
      onblur={finishEditing}
      onkeydown={handleKeydown}
      autofocus
    />
  {:else}
    <span 
      role="button"
      tabindex="0"
      class="todo-name" 
      class:strikethrough={todo.completedAt !== null}
      class:long-pressing={isLongPressing}
      ondblclick={startEditing}
      ontouchstart={handleTouchStart}
      ontouchend={handleTouchEnd}
      ontouchmove={handleTouchMove}
      aria-label="Double-click or long-press to edit"
    >
      {todo.name}
    </span>
  {/if}

  {#if categoryName}
    <span class="category-badge">
      {categoryName}
    </span>
  {/if}

  <button 
    class="star-btn" 
    class:starred={todo.starred}
    onclick={handleStarClick}
    aria-label={todo.starred ? 'Unstar' : 'Star'}
  >
    <svg viewBox="0 0 24 24" fill={todo.starred ? 'currentColor' : 'none'} stroke="currentColor" stroke-width="2">
      <polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"></polygon>
    </svg>
  </button>
</div>

<style>
  .todo-item {
    display: flex;
    align-items: center;
    gap: var(--spacing-md);
    padding: var(--spacing-lg) var(--spacing-xl);
    background: var(--card-bg);
    border-radius: var(--radius-md);
    transition: all var(--transition-slow);
    box-shadow: var(--shadow-sm);
  }

  .todo-item.completed {
    opacity: var(--opacity-completed);
  }

  .checkbox {
    width: var(--checkbox-size);
    height: var(--checkbox-size);
    border-radius: var(--position-half);
    border: var(--stroke-thin) solid var(--checkbox-border);
    background: transparent;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    transition: all var(--transition-normal);
    padding: 0;
  }

  .checkbox:hover {
    border-color: var(--primary-color);
  }

  .checkbox.checked {
    background: var(--primary-color);
    border-color: var(--primary-color);
    color: var(--text-on-primary);
  }

  .checkbox svg {
    width: var(--icon-xs);
    height: var(--icon-xs);
  }

  .todo-name {
    flex: 1;
    font-size: var(--font-size-base);
    color: var(--text-primary);
    cursor: default;
    transition: all var(--transition-normal);
    user-select: none;
    -webkit-user-select: none;
    -webkit-touch-callout: none;
  }

  .todo-name.long-pressing {
    opacity: var(--opacity-subtle);
    transform: scale(0.98);
  }

  .todo-name.strikethrough {
    text-decoration: line-through;
    color: var(--text-muted);
  }

  .category-badge {
    padding: var(--spacing-xs) var(--spacing-md);
    background: var(--primary-color);
    color: white;
    border-radius: var(--radius-full);
    font-size: var(--font-size-xs);
    font-weight: var(--font-weight-medium);
    line-height: var(--line-height-normal);
    white-space: nowrap;
    flex-shrink: 0;
  }

  .edit-input {
    flex: 1;
    font-size: var(--font-size-base);
    color: var(--text-primary);
    border: none;
    outline: none;
    background: transparent;
    padding: 0;
    font-family: inherit;
  }

  .star-btn {
    width: var(--icon-xl);
    height: var(--icon-xl);
    border: none;
    background: transparent;
    cursor: pointer;
    color: var(--checkbox-border);
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all var(--transition-normal);
    padding: 0;
  }

  .star-btn:hover {
    color: var(--star-color);
    transform: scale(1.1);
  }

  .star-btn.starred {
    color: var(--star-color);
  }

  .star-btn svg {
    width: var(--icon-sm);
    height: var(--icon-sm);
  }
</style>

