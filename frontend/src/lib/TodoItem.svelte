<script lang="ts">
  import type { Todo } from './types';
  import CheckboxRing from './CheckboxRing.svelte';
  import CategoryBadge from './CategoryBadge.svelte';

  interface Props {
    todo: Todo;
    categoryName?: string | null;
    duplicateCount?: number;
    onToggleComplete: (id: string) => void;
    onToggleStar: (id: string) => void;
    onRename: (id: string, name: string) => void;
    onRequestCategorize?: (todo: Todo) => void;
  }

  let { todo, categoryName = null, duplicateCount = 1, onToggleComplete, onToggleStar, onRename, onRequestCategorize }: Props = $props();

  let isEditing = $state(false);
  let editName = $state('');
  let longPressTimer: number | null = $state(null);
  let isLongPressing = $state(false);
  let touchStartTime = $state(0);
  let touchMoved = $state(false);
  let checkboxButton: HTMLButtonElement | null = $state(null);
  let starButton: HTMLButtonElement | null = $state(null);

  function handleCheckClick() {
    onToggleComplete(todo.id);
    // Blur the button after click to prevent focus from moving to next item
    // This fixes the issue where the item below gets a focus ring on mobile
    // Use queueMicrotask to ensure blur happens after browser's default focus behavior
    queueMicrotask(() => {
      if (checkboxButton) {
        checkboxButton.blur();
      }
    });
  }

  function handleStarClick() {
    onToggleStar(todo.id);
    // Blur the button after click to prevent focus from moving to next item
    queueMicrotask(() => {
      if (starButton) {
        starButton.blur();
      }
    });
  }

  function startEditing() {
    editName = todo.originalInput || todo.name;
    isEditing = true;
    isLongPressing = false;
  }

  function finishEditing() {
    const displayName = todo.originalInput || todo.name;
    if (editName.trim() && editName !== displayName) {
      onRename(todo.id, editName.trim());
    }
    isEditing = false;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      finishEditing();
    } else if (e.key === 'Escape') {
      editName = todo.originalInput || todo.name;
      isEditing = false;
    }
  }

  // Mobile long-press support
  function handleTouchStart(e: TouchEvent) {
    touchStartTime = Date.now();
    touchMoved = false;
    
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
    const touchDuration = Date.now() - touchStartTime;
    const wasQuickTap = touchDuration < 500 && !touchMoved;
    
    // Clear timer if touch ends before long-press threshold
    if (longPressTimer) {
      clearTimeout(longPressTimer);
      longPressTimer = null;
    }
    isLongPressing = false;
    
    // If it's a quick tap, show category selector (works for both categorized and uncategorized)
    if (wasQuickTap && onRequestCategorize) {
      e.preventDefault(); // Prevent any default behavior
      onRequestCategorize(todo);
    }
  }

  function handleTouchMove(e: TouchEvent) {
    touchMoved = true;
    // Cancel long-press if user moves finger
    if (longPressTimer) {
      clearTimeout(longPressTimer);
      longPressTimer = null;
    }
    isLongPressing = false;
  }
</script>

<div class="todo-item" class:completed={todo.completedAt !== null} class:editing={isEditing}>
  <button 
    class="checkbox" 
    bind:this={checkboxButton}
    onclick={handleCheckClick}
    aria-label={todo.completedAt ? 'Mark as incomplete' : 'Mark as complete'}
  >
    <CheckboxRing checked={todo.completedAt !== null} />
  </button>

  {#if isEditing}
    <input
      type="text"
      class="edit-input"
      bind:value={editName}
      onblur={finishEditing}
      onkeydown={handleKeydown}
    />
  {:else}
    <button
      type="button"
      class="todo-name-button" 
      class:strikethrough={todo.completedAt !== null}
      class:long-pressing={isLongPressing}
      ondblclick={startEditing}
      ontouchstart={handleTouchStart}
      ontouchend={handleTouchEnd}
      ontouchmove={handleTouchMove}
      onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); } }}
      aria-label="Double-click or long-press to edit"
    >
      <span class="todo-name">
        {todo.originalInput || todo.name}
      </span>
    </button>
  {/if}

  {#if duplicateCount > 1}
    <span class="duplicate-badge" aria-label={`${duplicateCount}x`}>
      {duplicateCount}x
    </span>
  {/if}

  {#if categoryName}
    <CategoryBadge name={categoryName!} />
  {/if}

  <button 
    class="star-btn" 
    class:starred={todo.starred}
    bind:this={starButton}
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
    /* Ensure proper alignment and prevent text overlap */
    min-height: 44px;
    /* Ensure proper stacking context */
    position: relative;
    isolation: isolate;
  }

  /* When editing, ensure item is above others and has solid background */
  /* Updated: 2026-02-17 - Fixed iOS text input layout issues */
  .todo-item.editing {
    z-index: 5;
    background: var(--card-bg);
    box-shadow: var(--shadow-md);
  }

  @media (max-width: 768px) {
    .todo-item {
      gap: var(--spacing-md);
      padding: var(--spacing-md) var(--spacing-lg);
      /* Ensure proper alignment on iOS */
      align-items: center;
      min-height: 44px; /* iOS touch target minimum */
    }

    :global(.category-badge) {
      display: none;
    }

    /* Ensure proper spacing for edit input on mobile */
    .edit-input {
      margin: 0;
      padding: 0;
      /* Ensure text doesn't overlap with checkbox */
      min-width: 0;
      /* Ensure proper vertical alignment - center with checkbox */
      height: auto;
      line-height: 1.4;
      /* Solid background to prevent content bleeding through */
      background: var(--card-bg);
      /* Ensure it's above other content */
      position: relative;
      z-index: 1;
    }

    /* Ensure checkbox maintains proper size and spacing */
    .checkbox {
      flex-shrink: 0;
      width: var(--checkbox-size);
      height: var(--checkbox-size);
      min-width: var(--checkbox-size);
      min-height: var(--checkbox-size);
    }

    /* When editing on mobile, ensure item is elevated */
    .todo-item.editing {
      z-index: 10;
      box-shadow: var(--shadow-lg);
    }
  }

  .todo-item.completed {
    opacity: var(--opacity-completed);
  }

  .checkbox {
    background: transparent;
    border: none;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 0;
    outline: none;
    /* Ensure checkbox doesn't shrink and maintains proper spacing */
    flex-shrink: 0;
    min-width: var(--checkbox-size);
    height: var(--checkbox-size);
  }

  /* Prevent focus rings on mobile for checkbox buttons */
  @media (max-width: 768px) {
    .checkbox:focus {
      outline: none;
    }
    
    .checkbox:focus-visible {
      outline: none;
    }
  }

  .todo-name-button {
    flex: 1;
    font-size: var(--font-size-base);
    color: var(--text-secondary);
    cursor: default;
    transition: all var(--transition-normal);
    user-select: none;
    -webkit-user-select: none;
    -webkit-touch-callout: none;
    background: transparent;
    border: none;
    padding: 0;
    text-align: left;
    font-family: inherit;
    width: 100%;
    /* Ensure proper alignment */
    display: flex;
    align-items: center;
    min-width: 0;
  }

  .todo-name {
    min-width: 0;
    /* Ensure proper text display */
    display: block;
    line-height: 1.4;
    word-wrap: break-word;
    overflow-wrap: break-word;
  }

  .duplicate-badge {
    padding: 2px var(--spacing-sm);
    border-radius: var(--radius-full);
    background: #e9d5ff; /* Light purple for light mode */
    color: #6b21a8; /* Dark purple text for light mode */
    font-size: var(--font-size-xs);
    font-weight: var(--font-weight-medium);
    line-height: var(--line-height-normal);
    flex-shrink: 0;
  }

  /* Dark mode styling */
  @media (prefers-color-scheme: dark) {
    .duplicate-badge {
      background: #c4b5fd; /* Lighter purple for dark mode */
      color: white;
    }
  }

  :global(:root[data-theme="dark"]) .duplicate-badge {
    background: #c4b5fd; /* Lighter purple for dark mode */
    color: white;
  }

  .todo-name-button.long-pressing {
    opacity: var(--opacity-subtle);
    transform: scale(0.98);
  }

  .todo-name-button.strikethrough {
    text-decoration: line-through;
    color: var(--text-muted);
  }


  .edit-input {
    flex: 1;
    font-size: var(--font-size-base);
    color: var(--text-secondary);
    border: none;
    outline: none;
    /* Use solid background to prevent content bleeding through */
    background: var(--card-bg);
    padding: 0;
    font-family: inherit;
    /* Ensure proper alignment and prevent text overlap */
    line-height: 1.4;
    /* Prevent iOS zoom on focus */
    font-size: max(var(--font-size-base), 16px);
    /* Ensure proper appearance on iOS */
    -webkit-appearance: none;
    appearance: none;
    /* Proper text alignment */
    text-align: left;
    width: 100%;
    min-width: 0;
    /* Ensure proper vertical alignment - inherit from parent flex container */
    height: auto;
    /* Ensure input is above other content */
    position: relative;
    z-index: 1;
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
    outline: none;
  }

  /* Prevent focus rings on mobile for star buttons */
  @media (max-width: 768px) {
    .star-btn:focus {
      outline: none;
    }
    
    .star-btn:focus-visible {
      outline: none;
    }
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

