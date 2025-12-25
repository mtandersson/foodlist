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
    completedExpanded = true,
    onToggleCompletedSection,
  }: Props = $props();

  let expandedCategories = $state<Set<string | null>>(new Set());
  let draggedId: string | null = $state(null);
  let dragOverCategoryId: string | null = $state(null);
  let autoExpandTimer: number | null = null;

  // Load expanded state from localStorage
  $effect(() => {
    if (typeof localStorage !== 'undefined') {
      const stored = localStorage.getItem('expandedCategories');
      if (stored) {
        try {
          const parsed = JSON.parse(stored);
          expandedCategories = new Set(parsed);
        } catch (e) {
          console.error('Failed to parse expandedCategories from localStorage', e);
        }
      }
    }
  });

  // Initialize new categories as expanded
  const categoryIds = $derived(categories.map((c) => c.id));

  $effect(() => {
    let changed = false;
    // Add any new categories to the expanded set
    categoryIds.forEach(id => {
      if (!expandedCategories.has(id)) {
        expandedCategories.add(id);
        changed = true;
      }
    });
    
    // Remove deleted categories from expanded set
    const categoryIdsSet = new Set(categoryIds);
    expandedCategories.forEach(id => {
      if (id !== null && !categoryIdsSet.has(id)) {
        expandedCategories.delete(id);
        changed = true;
      }
    });
    
    // Force reactivity and save to localStorage
    if (changed) {
      expandedCategories = expandedCategories;
      saveExpandedState();
    }
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

  function handleDragStart(e: DragEvent, todo: Todo) {
    draggedId = todo.id;
    if (e.dataTransfer) {
      e.dataTransfer.effectAllowed = 'move';
      e.dataTransfer.setData('text/plain', todo.id);
    }
  }

  function handleDragOverCategory(e: DragEvent, categoryId: string | null) {
    e.preventDefault();
    if (draggedId) {
      dragOverCategoryId = categoryId;
      
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
    }
  }

  function handleDragLeaveCategory() {
    if (autoExpandTimer) {
      clearTimeout(autoExpandTimer);
      autoExpandTimer = null;
    }
    dragOverCategoryId = null;
  }

  function handleDropOnCategory(e: DragEvent, categoryId: string | null) {
    e.preventDefault();
    if (autoExpandTimer) {
      clearTimeout(autoExpandTimer);
      autoExpandTimer = null;
    }
    dragOverCategoryId = null;

    if (!draggedId) {
      return;
    }

    // Categorize the todo
    onCategorizeTodo(draggedId, categoryId);
    draggedId = null;
  }

  function handleDragEnd() {
    if (autoExpandTimer) {
      clearTimeout(autoExpandTimer);
      autoExpandTimer = null;
    }
    draggedId = null;
    dragOverCategoryId = null;
  }
</script>

<!-- Uncategorized todos (no header, just the items) -->
<div class="todos-section">
  {#each todosForCategory(null) as todo (todo.id)}
    <div
      animate:flip={{ duration: 300 }}
      transition:fade={{ duration: 200 }}
      draggable="true"
      ondragstart={(e) => handleDragStart(e, todo)}
      ondragend={handleDragEnd}
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
  {/each}
</div>

<!-- Categories -->
{#each categories as category (category.id)}
  <CollapsibleSection
    title={category.name}
    count={todosForCategory(category.id).length}
    expanded={expandedCategories.has(category.id)}
    onToggle={() => toggleCategory(category.id)}
    onDelete={todosForCategory(category.id).length === 0 ? () => onDeleteCategory(category.id) : undefined}
  >
    {#if todosForCategory(category.id).length > 0 || draggedId}
      <div
        class="category-drop-zone"
        class:drag-over={dragOverCategoryId === category.id}
        ondragover={(e) => handleDragOverCategory(e, category.id)}
        ondragleave={handleDragLeaveCategory}
        ondrop={(e) => handleDropOnCategory(e, category.id)}
      >
        {#each todosForCategory(category.id) as todo (todo.id)}
          <div
            animate:flip={{ duration: 300 }}
            transition:fade={{ duration: 200 }}
            draggable="true"
            ondragstart={(e) => handleDragStart(e, todo)}
            ondragend={handleDragEnd}
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
        {/each}
        {#if todosForCategory(category.id).length === 0}
          <div class="empty-drop-area">Drop items here</div>
        {/if}
      </div>
    {/if}
  </CollapsibleSection>
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
    margin-bottom: 16px;
  }

  .category-drop-zone {
    position: relative;
    transition: all 0.2s ease;
    min-height: 20px;
    pointer-events: none;
  }

  .category-drop-zone > * {
    pointer-events: auto;
  }

  .category-drop-zone.drag-over {
    pointer-events: auto;
    background: var(--surface-light);
    border-radius: 12px;
    padding: 8px;
    margin: -8px;
  }

  .dragging {
    opacity: 0.5;
  }

  .empty-drop-area {
    padding: 24px;
    text-align: center;
    color: var(--text-muted);
    font-size: 14px;
    border: 2px dashed var(--surface-muted);
    border-radius: 8px;
    opacity: 0;
    transition: opacity 0.2s ease;
    pointer-events: none;
  }

  .category-drop-zone.drag-over .empty-drop-area {
    opacity: 1;
  }
</style>
