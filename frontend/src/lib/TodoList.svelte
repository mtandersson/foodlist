<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { flip } from 'svelte/animate';
  import { fade, slide } from 'svelte/transition';
  import { createTodoStore } from './store';
  import TodoItem from './TodoItem.svelte';
  import ModeSwitch from './ModeSwitch.svelte';
  import CategoriesView from './CategoriesView.svelte';
  import CollapsibleSection from './CollapsibleSection.svelte';
  import type { Todo, AutocompleteSuggestion } from './types';

  // Determine WebSocket URL
  const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const wsUrl = import.meta.env.DEV 
    ? 'ws://localhost:8080/ws'
    : `${wsProtocol}//${window.location.host}/ws`;

  const store = createTodoStore(wsUrl);
  const { activeTodos, completedTodos, categories, activeTodosByCategory, categoryLookup, connectionState, userCount, listTitle, autocompleteSuggestions } = store;

  let newTodoName = $state('');
  let completedExpanded = $state(true);
  let viewMode: 'normal' | 'categories' = $state((typeof localStorage !== 'undefined' && (localStorage.getItem('viewMode') as 'normal' | 'categories')) || 'normal');
  let pendingCategoryId: string | null = $state(null);
  let menuOpen = $state(false);
  
  // List title state
  let editingTitle = $state(false);
  let titleInputValue = $state('');
  
  // New category state
  let creatingCategory = $state(false);
  let newCategoryName = $state('');

  // Autocomplete state
  let showAutocomplete = $state(false);
  let selectedAutocompleteIndex = $state(-1);
  let inputFocused = $state(false);

  function handleAddTodo() {
    const name = newTodoName.trim();
    if (name) {
      store.createTodo(name, pendingCategoryId);
      newTodoName = '';
      pendingCategoryId = null;
      hideAutocomplete();
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    const suggestions = $autocompleteSuggestions;
    
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (suggestions.length > 0) {
        selectedAutocompleteIndex = Math.min(selectedAutocompleteIndex + 1, suggestions.length - 1);
      }
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (suggestions.length > 0) {
        selectedAutocompleteIndex = Math.max(selectedAutocompleteIndex - 1, -1);
      }
    } else if (e.key === 'Enter') {
      if (selectedAutocompleteIndex >= 0 && selectedAutocompleteIndex < suggestions.length) {
        e.preventDefault();
        selectSuggestion(suggestions[selectedAutocompleteIndex]);
      } else {
        handleAddTodo();
      }
    } else if (e.key === 'Escape') {
      hideAutocomplete();
    }
  }

  function handleInput() {
    // Request autocomplete on every keystroke
    store.requestAutocomplete(newTodoName);
    showAutocomplete = true;
    selectedAutocompleteIndex = -1;
    pendingCategoryId = null;
  }

  function handleInputFocus() {
    inputFocused = true;
    // Request autocomplete when focusing even if empty
    store.requestAutocomplete(newTodoName);
    showAutocomplete = true;
  }

  function handleInputBlur() {
    inputFocused = false;
    // Delay hiding to allow click on suggestion
    setTimeout(() => {
      if (!inputFocused) {
        hideAutocomplete();
      }
    }, 150);
  }

  function selectSuggestion(suggestion: AutocompleteSuggestion) {
    newTodoName = suggestion.name;
    pendingCategoryId = suggestion.categoryId ?? null;
    hideAutocomplete();
    // Immediately add the todo
    handleAddTodo();
  }

  function hideAutocomplete() {
    showAutocomplete = false;
    selectedAutocompleteIndex = -1;
    store.clearAutocomplete();
  }

  function toggleCompletedSection() {
    completedExpanded = !completedExpanded;
  }

  function handleModeChange(mode: 'normal' | 'categories') {
    viewMode = mode;
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem('viewMode', mode);
    }
  }

  function getCategoryName(categoryId: string | null | undefined): string | null {
    if (!categoryId) return null;
    const cat = $categoryLookup.get(categoryId);
    return cat ? cat.name : null;
  }

  function handleCategorize(id: string, categoryId: string | null) {
    store.categorizeTodo(id, categoryId);
  }

  function handleCreateCategory(name: string) {
    store.createCategory(name);
  }

  function handleRenameCategory(id: string, name: string) {
    store.renameCategory(id, name);
  }

  function handleDeleteCategory(id: string) {
    store.deleteCategory(id);
  }

  function handleReorderCategory(id: string, sortOrder: number) {
    store.reorderCategory(id, sortOrder);
  }

  // Title editing
  function startEditingTitle() {
    editingTitle = true;
    titleInputValue = $listTitle;
  }

  function finishEditingTitle() {
    const newTitle = titleInputValue.trim();
    if (newTitle && newTitle !== $listTitle) {
      store.setListTitle(newTitle);
    }
    editingTitle = false;
  }

  function handleTitleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      finishEditingTitle();
    } else if (e.key === 'Escape') {
      editingTitle = false;
    }
  }

  // Menu state
  function toggleMenu() {
    menuOpen = !menuOpen;
  }

  function closeMenu() {
    menuOpen = false;
  }

  function handleNewCategory() {
    creatingCategory = true;
    newCategoryName = '';
    closeMenu();
    // Focus the input after it renders
    setTimeout(() => {
      const input = document.querySelector('.new-category-input') as HTMLInputElement;
      input?.focus();
    }, 0);
  }

  function finishCreatingCategory() {
    const name = newCategoryName.trim();
    if (name) {
      handleCreateCategory(name);
    }
    creatingCategory = false;
    newCategoryName = '';
  }

  function cancelCreatingCategory() {
    creatingCategory = false;
    newCategoryName = '';
  }

  function handleNewCategoryKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      finishCreatingCategory();
    } else if (e.key === 'Escape') {
      cancelCreatingCategory();
    }
  }

  // Drag and drop state
  let draggedId: string | null = $state(null);
  let dropTargetId: string | null = $state(null);
  let dropPosition: 'above' | 'below' | null = $state(null);
  let isDragging = $state(false);

  function computeSortOrder(todos: Todo[], newIndex: number): number {
    if (todos.length === 0) return 0;
    if (newIndex <= 0) return todos[0].sortOrder + 1000;
    if (newIndex >= todos.length) return todos[todos.length - 1].sortOrder - 1000;
    const above = todos[newIndex - 1].sortOrder;
    const below = todos[newIndex].sortOrder;
    return Math.floor((above + below) / 2);
  }

  function handleDragStart(e: DragEvent, todo: Todo) {
    draggedId = todo.id;
    dropTargetId = null;
    dropPosition = null;
    isDragging = true;
    if (e.dataTransfer) {
      e.dataTransfer.effectAllowed = 'move';
      e.dataTransfer.setData('text/plain', todo.id);
    }
  }

  function wouldMove(targetId: string, position: 'above' | 'below'): boolean {
    const todos = $activeTodos;
    const draggedIndex = todos.findIndex((t) => t.id === draggedId);
    const targetIndex = todos.findIndex((t) => t.id === targetId);
    if (draggedIndex === -1 || targetIndex === -1) return false;
    const newIndex = position === 'below' ? targetIndex + 1 : targetIndex;
    // No-op if dropping in same or adjacent slot that keeps order
    return !(newIndex === draggedIndex || newIndex === draggedIndex + 1);
  }

  function handleDragOverItem(e: DragEvent, todo: Todo) {
    e.preventDefault();
    if (!draggedId || draggedId === todo.id) return;

    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const position: 'above' | 'below' = e.clientY < rect.top + rect.height / 2 ? 'above' : 'below';

    if (!wouldMove(todo.id, position)) {
      dropTargetId = null;
      dropPosition = null;
      return;
    }

    dropTargetId = todo.id;
    dropPosition = position;
  }

  function handleDragOverList(e: DragEvent) {
    e.preventDefault();
    if (!draggedId) return;

    const listEl = e.currentTarget as HTMLElement;
    const wrappers = Array.from(listEl.querySelectorAll('.todo-wrapper')) as HTMLElement[];
    const todos = $activeTodos;

    if (!wrappers.length || !todos.length) {
      dropTargetId = null;
      dropPosition = null;
      return;
    }

    const firstRect = wrappers[0].getBoundingClientRect();
    const lastRect = wrappers[wrappers.length - 1].getBoundingClientRect();
    const { clientY } = e;

    // If near the very top, target before first
    if (clientY < firstRect.top + firstRect.height / 3 && wouldMove(todos[0].id, 'above')) {
      dropTargetId = todos[0].id;
      dropPosition = 'above';
      return;
    }

    // If near the very bottom, target after last
    if (clientY > lastRect.bottom - lastRect.height / 3 && wouldMove(todos[todos.length - 1].id, 'below')) {
      dropTargetId = todos[todos.length - 1].id;
      dropPosition = 'below';
      return;
    }
    // Otherwise let item-level handlers set precise position
  }

  function handleDragLeaveItem(e: DragEvent, todoId: string) {
    const relatedTarget = e.relatedTarget as Node | null;
    if (!relatedTarget || !(e.currentTarget as Node).contains(relatedTarget)) {
      if (dropTargetId === todoId) {
        dropTargetId = null;
        dropPosition = null;
      }
    }
  }

  function finishDrop() {
    if (!draggedId) return;

    // Only reorder if we have a valid drop target
    if (!dropTargetId || !dropPosition) {
      clearDragState();
      return;
    }

    const todos = $activeTodos;
    const targetIndex = todos.findIndex((t) => t.id === dropTargetId);
    
    if (targetIndex === -1) {
      clearDragState();
      return;
    }

    const newIndex = dropPosition === 'below' ? targetIndex + 1 : targetIndex;
    const newSortOrder = computeSortOrder(todos, newIndex);
    store.reorder(draggedId, newSortOrder);
    clearDragState();
  }

  function handleDrop(e: DragEvent) {
    e.preventDefault();
    finishDrop();
  }

  function handleDragEnd() {
    clearDragState();
  }

  function clearDragState() {
    draggedId = null;
    dropTargetId = null;
    dropPosition = null;
    isDragging = false;
  }

  onDestroy(() => {
    store.destroy();
  });
</script>

<div class="todo-list-container">
  <header class="header">
    {#if editingTitle}
      <input
        type="text"
        class="title-input"
        bind:value={titleInputValue}
        onkeydown={handleTitleKeydown}
        onblur={finishEditingTitle}
        autofocus
      />
    {:else}
      <h1 class="title" onclick={startEditingTitle} role="button" tabindex="0">
        {$listTitle}
      </h1>
    {/if}
    <div class="header-right">
      <ModeSwitch value={viewMode} on:change={(e) => handleModeChange(e.detail)} />
      <span class="member-count" aria-label="Connected users">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path>
          <circle cx="9" cy="7" r="4"></circle>
          <path d="M23 21v-2a4 4 0 0 0-3-3.87"></path>
          <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
        </svg>
        {$userCount}
      </span>
      <div class="menu-wrapper">
        <button class="menu-btn" aria-label="Menu" onclick={toggleMenu}>
          <svg viewBox="0 0 24 24" fill="currentColor">
            <circle cx="12" cy="5" r="2"></circle>
            <circle cx="12" cy="12" r="2"></circle>
            <circle cx="12" cy="19" r="2"></circle>
          </svg>
        </button>
        {#if menuOpen}
          <div class="menu-dropdown">
            <button class="menu-item" onclick={handleNewCategory}>
              <span class="menu-icon">➕</span>
              Ny kategori
            </button>
          </div>
          <button class="menu-backdrop" onclick={closeMenu} aria-label="Close menu"></button>
        {/if}
      </div>
    </div>
  </header>

  <div class="scrollable-content">
    <!-- Connection status indicator -->
    {#if $connectionState !== 'CONNECTED'}
      <div class="connection-status" transition:slide>
        {#if $connectionState === 'CONNECTING'}
          Ansluter...
        {:else if $connectionState === 'RECONNECTING'}
          Återansluter...
        {:else}
          Frånkopplad
        {/if}
      </div>
    {/if}

    <!-- New Category Input -->
    {#if creatingCategory}
      <div class="new-category-wrapper" transition:slide>
        <div class="new-category-input-container">
          <input
            type="text"
            class="new-category-input"
            bind:value={newCategoryName}
            onkeydown={handleNewCategoryKeydown}
            onblur={finishCreatingCategory}
            placeholder="Namn på ny kategori..."
          />
          <div class="new-category-actions">
            <button
              type="button"
              class="new-category-btn primary"
              onclick={finishCreatingCategory}
              disabled={!newCategoryName.trim()}
            >
              Skapa
            </button>
            <button
              type="button"
              class="new-category-btn"
              onclick={cancelCreatingCategory}
            >
              Avbryt
            </button>
          </div>
        </div>
      </div>
    {/if}

    {#if viewMode === 'normal'}
      <!-- Active todos -->
      <div
        class="todos-section"
        ondragover={handleDragOverList}
        ondrop={handleDrop}
        role="list"
      >
        {#each $activeTodos as todo (todo.id)}
          <div
            class="todo-container"
            class:spacer-top={dropTargetId === todo.id && dropPosition === 'above'}
            class:spacer-bottom={dropTargetId === todo.id && dropPosition === 'below'}
            animate:flip={{ duration: isDragging ? 0 : 300 }}
            transition:fade={{ duration: 200 }}
          >
            <div
              class="todo-wrapper"
              role="listitem"
              aria-grabbed={draggedId === todo.id}
              class:drop-above={dropTargetId === todo.id && dropPosition === 'above'}
              class:drop-below={dropTargetId === todo.id && dropPosition === 'below'}
              draggable="true"
              ondragstart={(e) => handleDragStart(e, todo)}
              ondragover={(e) => handleDragOverItem(e, todo)}
              ondragleave={(e) => handleDragLeaveItem(e, todo.id)}
              ondrop={handleDrop}
              ondragend={handleDragEnd}
              class:dragging={draggedId === todo.id}
            >
              <TodoItem
                {todo}
                categoryName={getCategoryName(todo.categoryId ?? null)}
                onToggleComplete={store.toggleComplete}
                onToggleStar={store.toggleStar}
                onRename={store.rename}
              />
            </div>
          </div>
        {/each}
      </div>

      <!-- Completed section -->
      {#if $completedTodos.length > 0}
        <CollapsibleSection
          title="Slutfört"
          expanded={completedExpanded}
          onToggle={toggleCompletedSection}
        >
          {#each $completedTodos as todo (todo.id)}
            <div
              animate:flip={{ duration: 300 }}
              transition:fade={{ duration: 200 }}
            >
              <TodoItem
                {todo}
                categoryName={getCategoryName(todo.categoryId ?? null)}
                onToggleComplete={store.toggleComplete}
                onToggleStar={store.toggleStar}
                onRename={store.rename}
              />
            </div>
          {/each}
        </CollapsibleSection>
      {/if}
    {:else}
      <CategoriesView
        categories={$categories}
        activeTodosByCategory={$activeTodosByCategory}
        completedTodos={$completedTodos}
        getCategoryName={getCategoryName}
        onToggleComplete={store.toggleComplete}
        onToggleStar={store.toggleStar}
        onRename={store.rename}
        onDeleteCategory={handleDeleteCategory}
        onCategorizeTodo={handleCategorize}
        onReorderCategory={handleReorderCategory}
        onReorder={store.reorder}
        completedExpanded={completedExpanded}
        onToggleCompletedSection={toggleCompletedSection}
      />
    {/if}
  </div>

  <!-- Add todo input at bottom -->
  <div class="add-todo-wrapper">
    {#if showAutocomplete && $autocompleteSuggestions.length > 0}
      <div class="autocomplete-dropdown" transition:slide={{ duration: 150 }}>
        {#each $autocompleteSuggestions as suggestion, index}
          <button
            type="button"
            class="autocomplete-item"
            class:selected={index === selectedAutocompleteIndex}
            onmousedown={() => selectSuggestion(suggestion)}
            onmouseenter={() => selectedAutocompleteIndex = index}
          >
            <div class="autocomplete-item-main">
              <span>{suggestion.name}</span>
              {#if suggestion.categoryName || suggestion.categoryId}
                <span class="autocomplete-badge">
                  {suggestion.categoryName ?? getCategoryName(suggestion.categoryId)}
                </span>
              {/if}
            </div>
          </button>
        {/each}
      </div>
    {/if}
    <form class="add-todo-bottom" onsubmit={(e) => { e.preventDefault(); handleAddTodo(); }}>
      <div class="add-todo-icon">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <line x1="12" y1="5" x2="12" y2="19"></line>
          <line x1="5" y1="12" x2="19" y2="12"></line>
        </svg>
      </div>
      <input
        type="text"
        placeholder="Lägg till en uppgift"
        bind:value={newTodoName}
        onkeydown={handleKeydown}
        oninput={handleInput}
        onfocus={handleInputFocus}
        onblur={handleInputBlur}
        aria-label="Ny uppgift"
        autocomplete="off"
      />
    </form>
  </div>
</div>

<style>
  .todo-list-container {
    width: min(90vw, 820px);
    max-width: 820px;
    margin: 0 auto;
    display: flex;
    flex-direction: column;
    height: calc(100vh - 48px);
    padding: 0 24px;
  }

  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 24px 0 20px;
    flex-shrink: 0;
  }

  .scrollable-content {
    flex: 1;
    overflow-y: auto;
    overflow-x: hidden;
    margin-bottom: 16px;
    padding-right: 4px;
    min-height: 0;
    /* Force scrollbar to always be visible for consistent layout */
    scrollbar-gutter: stable;
  }

  .scrollable-content::-webkit-scrollbar {
    width: 8px;
  }

  .scrollable-content::-webkit-scrollbar-track {
    background: transparent;
  }

  .scrollable-content::-webkit-scrollbar-thumb {
    background: rgba(255, 255, 255, 0.2);
    border-radius: 4px;
  }

  .scrollable-content::-webkit-scrollbar-thumb:hover {
    background: rgba(255, 255, 255, 0.3);
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .member-count {
    display: flex;
    align-items: center;
    gap: 6px;
    color: var(--text-on-primary);
    font-size: 14px;
  }

  .member-count svg {
    width: 20px;
    height: 20px;
  }

  .menu-btn {
    background: none;
    border: none;
    color: var(--text-on-primary);
    cursor: pointer;
    padding: 4px;
  }

  .menu-btn svg {
    width: 24px;
    height: 24px;
  }

  .menu-wrapper {
    position: relative;
  }

  .menu-dropdown {
    position: absolute;
    top: calc(100% + 8px);
    right: 0;
    background: var(--card-bg);
    border-radius: 12px;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.2);
    min-width: 200px;
    z-index: 150;
    overflow: hidden;
  }

  .menu-item {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    padding: 12px 16px;
    border: none;
    background: transparent;
    color: var(--text-primary);
    font-size: 15px;
    text-align: left;
    cursor: pointer;
    transition: background 0.2s ease;
  }

  .menu-item:hover {
    background: var(--surface-muted);
  }

  .menu-icon {
    font-size: 18px;
  }

  .menu-backdrop {
    position: fixed;
    inset: 0;
    background: transparent;
    border: none;
    cursor: default;
    z-index: 100;
  }

  .title {
    color: var(--text-on-primary);
    font-size: 32px;
    font-weight: 600;
    margin: 0;
    cursor: pointer;
    transition: opacity 0.2s;
    padding: 8px;
    border-radius: 8px;
  }

  .title:hover {
    opacity: 0.9;
    background: var(--surface-light);
  }

  .title:focus {
    outline: 2px solid rgba(var(--card-bg-rgb), 0.5);
    outline-offset: 2px;
  }

  .title-input {
    color: var(--text-on-primary);
    font-size: 32px;
    font-weight: 600;
    margin: 0;
    padding: 8px;
    border: 2px solid rgba(var(--card-bg-rgb), 0.5);
    border-radius: 8px;
    background: var(--surface-muted);
    width: 100%;
    outline: none;
    transition: all 0.2s;
  }

  .title-input:focus {
    border-color: var(--text-on-primary);
    background: var(--surface-muted-strong);
  }

  .connection-status {
    background: var(--surface-muted);
    color: var(--text-on-primary);
    padding: 8px 16px;
    border-radius: 8px;
    text-align: center;
    margin-bottom: 16px;
    font-size: 14px;
  }

  .new-category-wrapper {
    margin-bottom: 16px;
  }

  .new-category-input-container {
    background: var(--card-bg);
    border-radius: 12px;
    padding: 16px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  }

  .new-category-input {
    width: 100%;
    padding: 12px 16px;
    font-size: 16px;
    border: 2px solid var(--surface-muted);
    border-radius: 8px;
    background: var(--surface-muted);
    color: var(--text-primary);
    outline: none;
    font-family: inherit;
    transition: all 0.2s ease;
    margin-bottom: 12px;
  }

  .new-category-input:focus {
    border-color: var(--text-primary);
    background: var(--surface-muted-strong);
  }

  .new-category-input::placeholder {
    color: var(--text-muted);
  }

  .new-category-actions {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
  }

  .new-category-btn {
    padding: 8px 16px;
    font-size: 14px;
    border: none;
    border-radius: 8px;
    cursor: pointer;
    font-family: inherit;
    font-weight: 500;
    transition: all 0.2s ease;
    background: var(--surface-muted);
    color: var(--text-primary);
  }

  .new-category-btn:hover:not(:disabled) {
    background: var(--surface-muted-strong);
    transform: translateY(-1px);
  }

  .new-category-btn.primary {
    background: var(--text-on-primary);
    color: var(--primary-bg);
  }

  .new-category-btn.primary:hover:not(:disabled) {
    opacity: 0.9;
  }

  .new-category-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .add-todo-bottom {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 18px 20px;
    background: var(--surface-muted);
    border-radius: 16px;
    box-shadow: 0 2px 6px rgba(0, 0, 0, 0.05);
    cursor: text;
    transition: all 0.2s ease;
  }

  .add-todo-bottom:hover {
    background: var(--surface-muted-strong);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  }

  .add-todo-bottom:focus-within {
    background: var(--card-bg);
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
  }

  .add-todo-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    flex-shrink: 0;
    color: var(--text-muted);
  }

  .add-todo-icon svg {
    width: 28px;
    height: 28px;
  }

  .add-todo-bottom input {
    flex: 1;
    border: none;
    outline: none;
    font-size: 17px;
    color: var(--text-muted);
    background: transparent;
    font-family: inherit;
    transition: color 0.2s ease;
  }

  .add-todo-bottom input:focus {
    color: var(--text-primary);
  }

  .add-todo-bottom input::placeholder {
    color: var(--text-muted);
  }

  .todos-section {
    margin-bottom: 16px;
  }

  .todo-container {
    position: relative;
  }

  .todo-container.spacer-top {
    margin-top: 64px;
  }

  .todo-container.spacer-bottom {
    margin-bottom: 64px;
  }

  .todo-wrapper {
    position: relative;
    margin-bottom: 8px;
    transition: transform 0.1s ease;
  }

  .todo-wrapper.dragging {
    opacity: 0.5;
  }

  .todo-wrapper.drop-above,
  .todo-wrapper.drop-below {
    z-index: 10;
  }

  .todo-wrapper.drop-above::before,
  .todo-wrapper.drop-below::after {
    content: '';
    display: block;
    height: 56px;
    border: 2px dashed var(--text-on-primary);
    border-radius: 12px;
    opacity: 0.9;
    animation: pulse 1s ease-in-out infinite;
    position: absolute;
    width: 100%;
    z-index: 11;
  }

  .todo-wrapper.drop-above::before {
    top: -64px;
  }

  .todo-wrapper.drop-below::after {
    top: auto;
    bottom: -64px;
  }

  /* 
    Remove the translateY on the item itself to avoid layout thrashing/jumping
    The pseudo-element provides the visual cue.
  */
  
  @keyframes pulse {
    0%, 100% {
      opacity: 0.8;
      border-width: 2px;
    }
    50% {
      opacity: 1;
      border-width: 3px;
    }
  }

  .add-todo-wrapper {
    position: relative;
    flex-shrink: 0;
    margin-bottom: 24px;
  }

  .autocomplete-dropdown {
    position: absolute;
    bottom: 100%;
    left: 0;
    right: 0;
    background: var(--card-bg);
    border-radius: 12px;
    box-shadow: 0 -4px 16px rgba(0, 0, 0, 0.15);
    margin-bottom: 8px;
    overflow: hidden;
    z-index: 100;
  }

  .autocomplete-item {
    display: block;
    width: 100%;
    padding: 14px 20px;
    text-align: left;
    background: transparent;
    border: none;
    font-size: 16px;
    color: var(--text-primary);
    cursor: pointer;
    transition: background 0.15s ease;
    font-family: inherit;
  }

  .autocomplete-item:hover,
  .autocomplete-item.selected {
    background: var(--surface-muted);
  }

  .autocomplete-item-main {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  .autocomplete-badge {
    background: var(--surface-muted-strong);
    color: var(--text-primary);
    padding: 4px 8px;
    border-radius: 999px;
    font-size: 12px;
    white-space: nowrap;
  }

  .autocomplete-item:first-child {
    border-radius: 12px 12px 0 0;
  }

  .autocomplete-item:last-child {
    border-radius: 0 0 12px 12px;
  }

  .autocomplete-item:only-child {
    border-radius: 12px;
  }
</style>
