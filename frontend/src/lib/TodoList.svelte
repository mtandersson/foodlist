<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { flip } from 'svelte/animate';
  import { fade, slide } from 'svelte/transition';
  import { createTodoStore } from './store';
  import TodoItem from './TodoItem.svelte';
  import type { Todo } from './types';

  // Determine WebSocket URL
  const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  const wsUrl = import.meta.env.DEV 
    ? 'ws://localhost:8080/ws'
    : `${wsProtocol}//${window.location.host}/ws`;

  const store = createTodoStore(wsUrl);
  const { activeTodos, completedTodos, connectionState, userCount, listTitle, autocompleteSuggestions } = store;

  let newTodoName = $state('');
  let completedExpanded = $state(true);
  
  // List title state
  let editingTitle = $state(false);
  let titleInputValue = $state('');

  // Autocomplete state
  let showAutocomplete = $state(false);
  let selectedAutocompleteIndex = $state(-1);
  let inputFocused = $state(false);

  function handleAddTodo() {
    const name = newTodoName.trim();
    if (name) {
      store.createTodo(name);
      newTodoName = '';
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

  function selectSuggestion(suggestion: string) {
    newTodoName = suggestion;
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

  // Drag and drop state
  let draggedId: string | null = $state(null);
  let dragOverId: string | null = $state(null);

  function handleDragStart(e: DragEvent, todo: Todo) {
    draggedId = todo.id;
    if (e.dataTransfer) {
      e.dataTransfer.effectAllowed = 'move';
      e.dataTransfer.setData('text/plain', todo.id);
    }
  }

  function handleDragOver(e: DragEvent, todo: Todo) {
    e.preventDefault();
    if (draggedId && draggedId !== todo.id) {
      dragOverId = todo.id;
    }
  }

  function handleDragLeave() {
    dragOverId = null;
  }

  function handleDrop(e: DragEvent, targetTodo: Todo) {
    e.preventDefault();
    dragOverId = null;

    if (!draggedId || draggedId === targetTodo.id) {
      draggedId = null;
      return;
    }

    // Calculate new sort order
    const todos = $activeTodos;
    const targetIndex = todos.findIndex(t => t.id === targetTodo.id);
    const draggedIndex = todos.findIndex(t => t.id === draggedId);

    if (targetIndex === -1 || draggedIndex === -1) {
      draggedId = null;
      return;
    }

    let newSortOrder: number;

    if (draggedIndex > targetIndex) {
      // Moving up - place above target
      const above = targetIndex > 0 ? todos[targetIndex - 1].sortOrder : targetTodo.sortOrder + 1000;
      newSortOrder = Math.floor((above + targetTodo.sortOrder) / 2);
    } else {
      // Moving down - place below target
      const below = targetIndex < todos.length - 1 ? todos[targetIndex + 1].sortOrder : targetTodo.sortOrder - 1000;
      newSortOrder = Math.floor((targetTodo.sortOrder + below) / 2);
    }

    store.reorder(draggedId, newSortOrder);
    draggedId = null;
  }

  function handleDragEnd() {
    draggedId = null;
    dragOverId = null;
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
      <span class="member-count" aria-label="Connected users">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path>
          <circle cx="9" cy="7" r="4"></circle>
          <path d="M23 21v-2a4 4 0 0 0-3-3.87"></path>
          <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
        </svg>
        {$userCount}
      </span>
      <button class="menu-btn" aria-label="Menu">
        <svg viewBox="0 0 24 24" fill="currentColor">
          <circle cx="12" cy="5" r="2"></circle>
          <circle cx="12" cy="12" r="2"></circle>
          <circle cx="12" cy="19" r="2"></circle>
        </svg>
      </button>
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

    <!-- Active todos -->
    <div class="todos-section">
      {#each $activeTodos as todo (todo.id)}
        <div
          animate:flip={{ duration: 300 }}
          transition:fade={{ duration: 200 }}
          draggable="true"
          ondragstart={(e) => handleDragStart(e, todo)}
          ondragover={(e) => handleDragOver(e, todo)}
          ondragleave={handleDragLeave}
          ondrop={(e) => handleDrop(e, todo)}
          ondragend={handleDragEnd}
          class:dragging={draggedId === todo.id}
          class:drag-over={dragOverId === todo.id}
        >
          <TodoItem
            {todo}
            onToggleComplete={store.toggleComplete}
            onToggleStar={store.toggleStar}
            onRename={store.rename}
          />
        </div>
      {/each}
    </div>

    <!-- Completed section -->
    {#if $completedTodos.length > 0}
      <button class="completed-header" onclick={toggleCompletedSection}>
        <svg 
          class="chevron" 
          class:expanded={completedExpanded}
          viewBox="0 0 24 24" 
          fill="none" 
          stroke="currentColor" 
          stroke-width="2"
        >
          <polyline points="9 18 15 12 9 6"></polyline>
        </svg>
        Slutfört
      </button>

      {#if completedExpanded}
        <div class="completed-section" transition:slide={{ duration: 300 }}>
          {#each $completedTodos as todo (todo.id)}
            <div
              animate:flip={{ duration: 300 }}
              transition:fade={{ duration: 200 }}
            >
              <TodoItem
                {todo}
                onToggleComplete={store.toggleComplete}
                onToggleStar={store.toggleStar}
                onRename={store.rename}
              />
            </div>
          {/each}
        </div>
      {/if}
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
            {suggestion}
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

  .back-btn {
    display: flex;
    align-items: center;
    gap: 4px;
    background: none;
    border: none;
    color: var(--text-on-primary);
    font-size: 16px;
    cursor: pointer;
    padding: 0;
  }

  .back-btn svg {
    width: 20px;
    height: 20px;
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

  .dragging {
    opacity: 0.5;
  }

  .drag-over {
    transform: translateY(4px);
  }

  .completed-header {
    display: flex;
    align-items: center;
    gap: 8px;
    background: var(--surface-muted);
    border: none;
    color: var(--text-on-primary);
    font-size: 14px;
    font-weight: 500;
    padding: 8px 12px;
    border-radius: 8px;
    cursor: pointer;
    margin-bottom: 12px;
    transition: background 0.2s ease;
  }

  .completed-header:hover {
    background: var(--surface-muted-strong);
  }

  .chevron {
    width: 16px;
    height: 16px;
    transition: transform 0.2s ease;
    transform: rotate(0deg);
  }

  .chevron.expanded {
    transform: rotate(90deg);
  }

  .completed-section {
    margin-bottom: 16px;
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

