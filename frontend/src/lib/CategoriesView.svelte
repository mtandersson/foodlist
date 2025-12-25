<script lang="ts">
  import { flip } from 'svelte/animate';
  import { fade } from 'svelte/transition';
  import TodoItem from './TodoItem.svelte';
  import CollapsibleSection from './CollapsibleSection.svelte';
  import type { Category, Todo } from './types';

  interface Props {
    categories?: Category[];
    activeTodosByCategory?: Map<string | null, Todo[]>;
    completedTodos?: Todo[];
    getCategoryName: (categoryId: string | null | undefined) => string | null;
    onToggleComplete: (id: string) => void;
    onToggleStar: (id: string) => void;
    onRename: (id: string, name: string) => void;
    onDeleteCategory: (id: string) => void;
    onCategorizeTodo: (id: string, categoryId: string | null) => void;
    onReorderCategory: (id: string, sortOrder: number) => void;
    onReorder: (id: string, newSortOrder: number) => void;
    completedExpanded?: boolean;
    onToggleCompletedSection: () => void;
  }

  let {
    categories = [],
    activeTodosByCategory = new Map(),
    completedTodos = [],
    getCategoryName,
    onToggleComplete,
    onToggleStar,
    onRename,
    onDeleteCategory,
    onCategorizeTodo,
    onReorderCategory,
    onReorder,
    completedExpanded = true,
    onToggleCompletedSection,
  }: Props = $props();

  let expandedCategories = $state<Set<string | null>>(new Set());
  let draggedId: string | null = $state(null);
  let dropCategoryId: string | null = $state(null);
  let dropTargetId: string | null = $state(null);
  let dropPosition: 'above' | 'below' | null = $state(null);
  let isDragging = $state(false);
  let autoExpandTimer: number | null = null;
  let isInitialized = false;
  let lastCategoryIds: string[] = [];

  // Load expanded state from localStorage once on mount
  if (typeof localStorage !== 'undefined' && !isInitialized) {
    const stored = localStorage.getItem('expandedCategories');
    if (stored) {
      try {
        const parsed = JSON.parse(stored);
        expandedCategories = new Set(parsed);
      } catch (e) {
        console.error('Failed to parse expandedCategories from localStorage', e);
      }
    }
    isInitialized = true;
  }

  // Initialize new categories as expanded and clean up deleted ones
  const categoryIds = $derived(categories.map((c) => c.id));

  $effect(() => {
    if (!isInitialized) return;
    
    // Check if the set of category IDs actually changed (not just order)
    const currentIdsSet = new Set(categoryIds);
    const lastIdsSet = new Set(lastCategoryIds);
    const hasNewCategories = categoryIds.some(id => !lastIdsSet.has(id));
    const hasDeletedCategories = lastCategoryIds.some(id => !currentIdsSet.has(id));
    
    if (!hasNewCategories && !hasDeletedCategories) {
      lastCategoryIds = categoryIds;
      return;
    }
    
    let changed = false;
    const newExpandedCategories = new Set(expandedCategories);
    
    // Add any new categories to the expanded set
    categoryIds.forEach(id => {
      if (!newExpandedCategories.has(id)) {
        newExpandedCategories.add(id);
        changed = true;
      }
    });
    
    // Remove deleted categories from expanded set
    const categoryIdsSet = new Set(categoryIds);
    newExpandedCategories.forEach(id => {
      if (id !== null && !categoryIdsSet.has(id)) {
        newExpandedCategories.delete(id);
        changed = true;
      }
    });
    
    // Only update if something actually changed
    if (changed) {
      expandedCategories = newExpandedCategories;
      saveExpandedState();
    }
    
    lastCategoryIds = categoryIds;
  });

  function saveExpandedState() {
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem('expandedCategories', JSON.stringify(Array.from(expandedCategories)));
    }
  }

  function toggleCategory(categoryId: string | null) {
    const newSet = new Set(expandedCategories);
    if (newSet.has(categoryId)) {
      newSet.delete(categoryId);
    } else {
      newSet.add(categoryId);
    }
    expandedCategories = newSet;
    saveExpandedState();
  }

  function todosForCategory(categoryId: string | null): Todo[] {
    return activeTodosByCategory.get(categoryId) ?? [];
  }

  function computeSortOrder(todos: Todo[], newIndex: number): number {
    if (todos.length === 0) return 0;
    if (newIndex <= 0) return todos[0].sortOrder + 1000;
    if (newIndex >= todos.length) return todos[todos.length - 1].sortOrder - 1000;
    const above = todos[newIndex - 1].sortOrder;
    const below = todos[newIndex].sortOrder;
    return Math.floor((above + below) / 2);
  }

  function getWrappersForCategory(container: HTMLElement, categoryId: string | null): HTMLElement[] {
    if (categoryId === null) {
      return Array.from(container.querySelectorAll(':scope > .todo-container')) as HTMLElement[];
    }
    const zone = container.querySelector('.category-drop-zone');
    if (!zone) return [];
    return Array.from(zone.querySelectorAll(':scope > .todo-container')) as HTMLElement[];
  }

  function wouldMove(categoryId: string | null, targetId: string, position: 'above' | 'below'): boolean {
    const todos = todosForCategory(categoryId);
    const draggedIndex = todos.findIndex((t) => t.id === draggedId);
    const targetIndex = todos.findIndex((t) => t.id === targetId);
    if (draggedIndex === -1 || targetIndex === -1) return true; // moving into a new category
    const newIndex = position === 'below' ? targetIndex + 1 : targetIndex;
    return !(newIndex === draggedIndex || newIndex === draggedIndex + 1);
  }

  function handleDragStart(e: DragEvent, todo: Todo) {
    draggedId = todo.id;
    dropTargetId = null;
    dropPosition = null;
    dropCategoryId = null;
    isDragging = true;
    if (e.dataTransfer) {
      e.dataTransfer.effectAllowed = 'move';
      e.dataTransfer.setData('text/plain', todo.id);
    }
  }

  function handleDragOverCategory(e: DragEvent, categoryId: string | null) {
    e.preventDefault();
    if (!draggedId) return;

    // Don't override dropCategoryId if we are hovering a todo in this category
    if (dropCategoryId !== categoryId) {
        dropCategoryId = categoryId;
    }
    
    // Auto-expand category after hovering for 500ms
    if (categoryId !== null && !expandedCategories.has(categoryId)) {
      if (autoExpandTimer) {
        clearTimeout(autoExpandTimer);
      }
      autoExpandTimer = window.setTimeout(() => {
        const newSet = new Set(expandedCategories);
        newSet.add(categoryId);
        expandedCategories = newSet;
        saveExpandedState();
        autoExpandTimer = null;
      }, 500);
    }

    const todos = todosForCategory(categoryId);
    const listEl = e.currentTarget as HTMLElement;
    const wrappers = getWrappersForCategory(listEl, categoryId);

    if (!todos.length || !wrappers.length) {
      if (todos.length === 0) {
        // Empty category - valid drop target
        dropTargetId = null;
        dropPosition = null;
      }
      return;
    }

    const firstRect = wrappers[0].getBoundingClientRect();
    const lastRect = wrappers[wrappers.length - 1].getBoundingClientRect();
    const { clientY } = e;

    // If near top of category, target above first (only if it changes order)
    if (clientY < firstRect.top + firstRect.height / 3) {
      const firstId = todos[0].id;
      if (wouldMove(categoryId, firstId, 'above')) {
        dropTargetId = firstId;
        dropPosition = 'above';
        return;
      }
    }

    // If near bottom, target below last (only if it changes order)
    if (clientY > lastRect.bottom - lastRect.height / 3) {
      const lastId = todos[todos.length - 1].id;
      if (wouldMove(categoryId, lastId, 'below')) {
        dropTargetId = lastId;
        dropPosition = 'below';
        return;
      }
    }
  }

  function handleDragOverTodo(e: DragEvent, todo: Todo, categoryId: string | null) {
    e.preventDefault();
    e.stopPropagation();
    
    if (!draggedId || draggedId === todo.id) {
      return;
    }

    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const midpoint = rect.top + rect.height / 2;
    const position: 'above' | 'below' = e.clientY < midpoint ? 'above' : 'below';

    // Avoid showing a drop target if it would not change order
    if (!wouldMove(categoryId, todo.id, position)) {
      if (dropTargetId === todo.id) {
        dropTargetId = null;
        dropPosition = null;
      }
      dropCategoryId = categoryId;
      return;
    }
    
    dropTargetId = todo.id;
    dropCategoryId = categoryId;
    dropPosition = position;
  }

  function handleDragLeaveTodo(e: DragEvent, todoId: string) {
    const relatedTarget = e.relatedTarget as Node | null;
    if (!relatedTarget || !(e.currentTarget as Node).contains(relatedTarget)) {
      if (dropTargetId === todoId) {
        dropTargetId = null;
        dropPosition = null;
      }
    }
  }

  function handleDragLeaveCategory(e: DragEvent) {
    // Only clear if we're actually leaving the category area
    const relatedTarget = e.relatedTarget as Node | null;
    if (!relatedTarget || !(e.currentTarget as Node).contains(relatedTarget)) {
      if (autoExpandTimer) {
        clearTimeout(autoExpandTimer);
        autoExpandTimer = null;
      }
      dropCategoryId = null;
      dropTargetId = null;
      dropPosition = null;
    }
  }

  function finalizeDrop(targetCategoryId: string | null) {
    if (!draggedId) return;

    const todos = todosForCategory(targetCategoryId);

    // Check if we have a valid drop target
    const effectiveTargetId = dropTargetId && todos.some((t) => t.id === dropTargetId)
      ? dropTargetId
      : null;

    // Find current category of dragged todo
    let draggedTodo: Todo | undefined;
    let currentCategory: string | null = null;
    for (const [catId, list] of activeTodosByCategory) {
      const found = list.find((t) => t.id === draggedId);
      if (found) {
        draggedTodo = found;
        currentCategory = catId;
        break;
      }
    }

    // Moving to a different category
    const isChangingCategory = draggedTodo && currentCategory !== targetCategoryId;

    // Only proceed if we're changing category OR we have a valid drop target
    if (!isChangingCategory && !effectiveTargetId) {
      clearDragState();
      return;
    }

    let newIndex = todos.length;
    if (effectiveTargetId) {
      const targetIndex = todos.findIndex((t) => t.id === effectiveTargetId);
      if (targetIndex !== -1) {
        newIndex = (dropPosition ?? 'below') === 'below' ? targetIndex + 1 : targetIndex;
      }
    }

    const newSortOrder = computeSortOrder(todos, newIndex);

    if (isChangingCategory) {
      onCategorizeTodo(draggedId, targetCategoryId);
    }

    onReorder(draggedId, newSortOrder);
    clearDragState();
  }

  function handleDrop(e: DragEvent, categoryId: string | null) {
    e.preventDefault();
    if (autoExpandTimer) {
      clearTimeout(autoExpandTimer);
      autoExpandTimer = null;
    }
    finalizeDrop(categoryId);
  }

  function handleDragEnd() {
    clearDragState();
  }

  function clearDragState() {
    if (autoExpandTimer) {
      clearTimeout(autoExpandTimer);
      autoExpandTimer = null;
    }
    draggedId = null;
    dropCategoryId = null;
    dropTargetId = null;
    dropPosition = null;
    isDragging = false;
  }

  function handleMoveUp(categoryId: string, currentIndex: number) {
    if (currentIndex === 0) return; // Already at the top
    
    const category = categories[currentIndex];
    const categoryAbove = categories[currentIndex - 1];
    
    // Calculate new sortOrder to be between the one above and the current
    const above = currentIndex > 1 ? categories[currentIndex - 2].sortOrder : categoryAbove.sortOrder + 1000;
    const newSortOrder = Math.floor((above + categoryAbove.sortOrder) / 2);
    
    onReorderCategory(categoryId, newSortOrder);
  }

  function handleMoveDown(categoryId: string, currentIndex: number) {
    if (currentIndex === categories.length - 1) return; // Already at the bottom
    
    const category = categories[currentIndex];
    const categoryBelow = categories[currentIndex + 1];
    
    // Calculate new sortOrder to be between the current and the one below
    const below = currentIndex < categories.length - 2 ? categories[currentIndex + 2].sortOrder : categoryBelow.sortOrder - 1000;
    const newSortOrder = Math.floor((categoryBelow.sortOrder + below) / 2);
    
    onReorderCategory(categoryId, newSortOrder);
  }
</script>

<!-- Uncategorized todos (no header, just the items) -->
<div 
  class="todos-section"
  class:drag-over-uncategorized={draggedId && dropCategoryId === null}
  ondragover={(e) => handleDragOverCategory(e, null)}
  ondragleave={(e) => handleDragLeaveCategory(e)}
  ondrop={(e) => handleDrop(e, null)}
  role="list"
>
  {#each todosForCategory(null) as todo (todo.id)}
    <div
      class="todo-container"
      class:spacer-top={dropCategoryId === null && dropTargetId === todo.id && dropPosition === 'above'}
      class:spacer-bottom={dropCategoryId === null && dropTargetId === todo.id && dropPosition === 'below'}
      animate:flip={{ duration: isDragging ? 0 : 300 }}
      transition:fade={{ duration: 200 }}
    >
      <div
        class="todo-wrapper"
        role="listitem"
        aria-grabbed={draggedId === todo.id}
        class:drop-above={dropCategoryId === null && dropTargetId === todo.id && dropPosition === 'above'}
        class:drop-below={dropCategoryId === null && dropTargetId === todo.id && dropPosition === 'below'}
        draggable="true"
        ondragstart={(e) => handleDragStart(e, todo)}
        ondragover={(e) => handleDragOverTodo(e, todo, null)}
        ondragleave={(e) => handleDragLeaveTodo(e, todo.id)}
        ondragend={handleDragEnd}
        ondrop={(e) => handleDrop(e, null)}
        class:dragging={draggedId === todo.id}
      >
        <TodoItem
          {todo}
          categoryName={null}
          onToggleComplete={onToggleComplete}
          onToggleStar={onToggleStar}
          onRename={onRename}
        />
      </div>
    </div>
  {/each}
  {#if todosForCategory(null).length === 0 && draggedId}
    <div class="empty-drop-area">Drop here to remove category</div>
  {/if}
</div>

<!-- Categories -->
{#each categories as category, index (category.id)}
  <div
    class="category-wrapper"
    class:drag-over-category={draggedId && dropCategoryId === category.id}
    ondragover={(e) => handleDragOverCategory(e, category.id)}
    ondragleave={(e) => handleDragLeaveCategory(e)}
    ondrop={(e) => handleDrop(e, category.id)}
    role="group"
  >
    <CollapsibleSection
      title={category.name}
      count={todosForCategory(category.id).length}
      expanded={expandedCategories.has(category.id)}
      onToggle={() => toggleCategory(category.id)}
      onDelete={todosForCategory(category.id).length === 0 ? () => onDeleteCategory(category.id) : undefined}
      onMoveUp={index > 0 ? () => handleMoveUp(category.id, index) : undefined}
      onMoveDown={index < categories.length - 1 ? () => handleMoveDown(category.id, index) : undefined}
    >
      {#if todosForCategory(category.id).length > 0 || draggedId}
        <div class="category-drop-zone" role="list">
          {#each todosForCategory(category.id) as todo (todo.id)}
            <div
              class="todo-container"
              class:spacer-top={dropCategoryId === category.id && dropTargetId === todo.id && dropPosition === 'above'}
              class:spacer-bottom={dropCategoryId === category.id && dropTargetId === todo.id && dropPosition === 'below'}
              animate:flip={{ duration: isDragging ? 0 : 300 }}
              transition:fade={{ duration: 200 }}
            >
              <div
                class="todo-wrapper"
                role="listitem"
                aria-grabbed={draggedId === todo.id}
                class:drop-above={dropCategoryId === category.id && dropTargetId === todo.id && dropPosition === 'above'}
                class:drop-below={dropCategoryId === category.id && dropTargetId === todo.id && dropPosition === 'below'}
                draggable="true"
                ondragstart={(e) => handleDragStart(e, todo)}
                ondragover={(e) => handleDragOverTodo(e, todo, category.id)}
                ondragleave={(e) => handleDragLeaveTodo(e, todo.id)}
                ondragend={handleDragEnd}
                ondrop={(e) => handleDrop(e, category.id)}
                class:dragging={draggedId === todo.id}
              >
                <TodoItem
                  {todo}
                  categoryName={null}
                  onToggleComplete={onToggleComplete}
                  onToggleStar={onToggleStar}
                  onRename={onRename}
                />
              </div>
            </div>
          {/each}
          {#if todosForCategory(category.id).length === 0 && draggedId}
            <div class="empty-drop-area">Drop items here</div>
          {/if}
        </div>
      {/if}
    </CollapsibleSection>
  </div>
{/each}

<!-- Completed section -->
{#if completedTodos.length > 0}
  <CollapsibleSection
    title="Slutfört"
    expanded={completedExpanded}
    onToggle={onToggleCompletedSection}
  >
    {#each completedTodos as todo (todo.id)}
      <div
        class="todo-wrapper"
        animate:flip={{ duration: 300 }}
        transition:fade={{ duration: 200 }}
      >
        <TodoItem
          {todo}
          categoryName={getCategoryName(todo.categoryId)}
          onToggleComplete={onToggleComplete}
          onToggleStar={onToggleStar}
          onRename={onRename}
        />
      </div>
    {/each}
  </CollapsibleSection>
{/if}

<style>
  .todos-section {
    margin-bottom: var(--spacing-lg);
    position: relative;
    min-height: var(--min-drop-zone-height);
    transition: all var(--transition-normal);
  }

  .todos-section.drag-over-uncategorized {
    background: var(--surface-light);
    border-radius: var(--radius-md);
    padding: var(--spacing-sm);
    margin: calc(-1 * var(--spacing-sm)) calc(-1 * var(--spacing-sm)) var(--spacing-sm) calc(-1 * var(--spacing-sm));
  }

  .category-wrapper {
    position: relative;
    transition: all var(--transition-normal);
    margin-bottom: var(--spacing-lg);
  }

  .category-wrapper.drag-over-category {
    background: var(--surface-light);
    border-radius: var(--radius-md);
    padding: var(--spacing-sm);
    margin: calc(-1 * var(--spacing-sm)) calc(-1 * var(--spacing-sm)) var(--spacing-sm) calc(-1 * var(--spacing-sm));
  }

  .category-drop-zone {
    position: relative;
    min-height: var(--min-drop-zone-height);
  }

  .todo-container {
    position: relative;
  }

  .todo-container.spacer-top {
    margin-top: var(--drop-spacing);
  }

  .todo-container.spacer-bottom {
    margin-bottom: var(--drop-spacing);
  }

  .todo-wrapper {
    position: relative;
    margin-bottom: var(--spacing-sm);
    transition: transform var(--duration-instant);
  }

  .dragging {
    opacity: var(--opacity-dragging);
  }

  /* svelte-ignore css-unused-selector */
  :global(.todo-wrapper.drop-above),
  :global(.todo-wrapper.drop-below) {
    z-index: var(--z-index-drop-target);
  }

  /* svelte-ignore css-unused-selector */
  :global(.todo-wrapper.drop-above)::before {
    content: '';
    display: block;
    height: var(--drop-indicator-height);
    border: var(--stroke-thin) dashed var(--text-on-primary);
    border-radius: var(--radius-md);
    opacity: var(--opacity-hover);
    animation: pulse var(--duration-pulse) ease-in-out infinite;
    position: absolute;
    width: 100%;
    top: calc(-1 * var(--drop-spacing));
    z-index: var(--z-index-drop-indicator);
  }

  /* svelte-ignore css-unused-selector */
  :global(.todo-wrapper.drop-below)::after {
    content: '';
    display: block;
    height: var(--drop-indicator-height);
    border: var(--stroke-thin) dashed var(--text-on-primary);
    border-radius: var(--radius-md);
    opacity: var(--opacity-hover);
    animation: pulse var(--duration-pulse) ease-in-out infinite;
    position: absolute;
    width: 100%;
    bottom: calc(-1 * var(--drop-spacing));
    z-index: var(--z-index-drop-indicator);
  }

  @keyframes pulse {
    0%, 100% {
      opacity: var(--opacity-pulse-min);
      border-width: var(--stroke-thin);
    }
    50% {
      opacity: var(--opacity-pulse-max);
      border-width: var(--stroke-bold);
    }
  }

  .empty-drop-area {
    padding: var(--spacing-2xl);
    text-align: center;
    color: var(--text-muted);
    font-size: var(--font-size-sm);
    border: var(--stroke-thin) dashed var(--surface-muted);
    border-radius: var(--radius-sm);
    transition: opacity var(--transition-normal);
  }
</style>
