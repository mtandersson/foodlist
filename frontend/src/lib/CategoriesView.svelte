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
    completedExpanded = true,
    onToggleCompletedSection,
  }: Props = $props();

  let expandedCategories = $state<Set<string | null>>(new Set());

  // Initialize new categories as expanded
  const categoryIds = $derived(categories.map((c) => c.id));

  $effect(() => {
    // Add any new categories to the expanded set
    categoryIds.forEach(id => {
      if (!expandedCategories.has(id)) {
        expandedCategories.add(id);
      }
    });
    // Force reactivity
    expandedCategories = expandedCategories;
  });

  function toggleCategory(categoryId: string | null) {
    const newSet = new Set(expandedCategories);
    if (newSet.has(categoryId)) {
      newSet.delete(categoryId);
    } else {
      newSet.add(categoryId);
    }
    expandedCategories = newSet;
  }

  function todosForCategory(categoryId: string | null): Todo[] {
    return activeTodosByCategory.get(categoryId) ?? [];
  }
</script>

<!-- Uncategorized todos (no header, just the items) -->
<div class="todos-section">
  {#each todosForCategory(null) as todo (todo.id)}
    <div
      animate:flip={{ duration: 300 }}
      transition:fade={{ duration: 200 }}
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
  {#if todosForCategory(category.id).length > 0}
    <CollapsibleSection
      title={category.name}
      count={todosForCategory(category.id).length}
      expanded={expandedCategories.has(category.id)}
      onToggle={() => toggleCategory(category.id)}
    >
      {#each todosForCategory(category.id) as todo (todo.id)}
        <div
          animate:flip={{ duration: 300 }}
          transition:fade={{ duration: 200 }}
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
    </CollapsibleSection>
  {/if}
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
</style>
