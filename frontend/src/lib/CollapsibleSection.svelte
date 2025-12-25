<script lang="ts">
  import { slide } from 'svelte/transition';

  interface Props {
    title: string;
    count?: number;
    expanded?: boolean;
    onToggle: () => void;
    onDelete?: () => void;
    onMoveUp?: () => void;
    onMoveDown?: () => void;
    onRename?: (newName: string) => void;
    children?: import('svelte').Snippet;
  }

  let { title, count, expanded = true, onToggle, onDelete, onMoveUp, onMoveDown, onRename, children }: Props = $props();

  let isEditing = $state(false);
  let editValue = $state('');
  let inputElement: HTMLInputElement | undefined = $state();

  function handleDelete(e: MouseEvent) {
    e.stopPropagation();
    if (onDelete) {
      onDelete();
    }
  }

  function handleMoveUp(e: MouseEvent) {
    e.stopPropagation();
    if (onMoveUp) {
      onMoveUp();
    }
  }

  function handleMoveDown(e: MouseEvent) {
    e.stopPropagation();
    if (onMoveDown) {
      onMoveDown();
    }
  }

  function handleStartEdit(e: MouseEvent) {
    e.stopPropagation();
    isEditing = true;
    editValue = title;
    // Focus the input after it's rendered
    setTimeout(() => {
      inputElement?.focus();
      inputElement?.select();
    }, 0);
  }

  function handleFinishEdit() {
    const newName = editValue.trim();
    if (newName && newName !== title && onRename) {
      onRename(newName);
    }
    isEditing = false;
  }

  function handleEditKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleFinishEdit();
    } else if (e.key === 'Escape') {
      e.preventDefault();
      isEditing = false;
    }
  }

  function handleEditClick(e: MouseEvent) {
    e.stopPropagation();
  }

  function handleTitleDoubleClick(e: MouseEvent) {
    if (onRename) {
      e.stopPropagation();
      handleStartEdit(e);
    }
  }

  function handleTitleKeydown(e: KeyboardEvent) {
    if (onRename && (e.key === 'Enter' || e.key === ' ')) {
      e.preventDefault();
      e.stopPropagation();
      handleStartEdit(e as any);
    }
  }
</script>

<div class="collapsible-section">
  <div class="section-header-wrapper">
    <button class="section-header" onclick={onToggle}>
      <svg 
        class="chevron" 
        class:expanded={expanded}
        viewBox="0 0 24 24" 
        fill="none" 
        stroke="currentColor" 
        stroke-width="2"
      >
        <polyline points="9 18 15 12 9 6"></polyline>
      </svg>
      {#if isEditing}
        <input
          bind:this={inputElement}
          bind:value={editValue}
          class="title-input"
          onclick={handleEditClick}
          onkeydown={handleEditKeydown}
          onblur={handleFinishEdit}
          type="text"
        />
      {:else}
        <span 
          class="section-title" 
          ondblclick={handleTitleDoubleClick}
          onkeydown={handleTitleKeydown}
          role={onRename ? "button" : undefined}
          tabindex={onRename ? 0 : undefined}
        >{title}</span>
      {/if}
      {#if count !== undefined}
        <span class="section-count">{count}</span>
      {/if}
    </button>
    <div class="action-buttons">
      {#if onRename && !isEditing}
        <button 
          class="edit-btn" 
          onclick={handleStartEdit}
          aria-label="Rename category"
          title="Rename category"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path>
            <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path>
          </svg>
        </button>
      {/if}
      {#if onMoveUp}
        <button 
          class="move-btn" 
          onclick={handleMoveUp}
          aria-label="Move category up"
          title="Move up"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="18 15 12 9 6 15"></polyline>
          </svg>
        </button>
      {/if}
      {#if onMoveDown}
        <button 
          class="move-btn" 
          onclick={handleMoveDown}
          aria-label="Move category down"
          title="Move down"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="6 9 12 15 18 9"></polyline>
          </svg>
        </button>
      {/if}
      {#if onDelete}
        <button 
          class="delete-btn" 
          onclick={handleDelete}
          aria-label="Delete category"
          title="Delete category"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="3 6 5 6 21 6"></polyline>
            <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
            <line x1="10" y1="11" x2="10" y2="17"></line>
            <line x1="14" y1="11" x2="14" y2="17"></line>
          </svg>
        </button>
      {/if}
    </div>
  </div>

  {#if expanded}
    <div class="section-content" transition:slide={{ duration: 300 }}>
      {@render children?.()}
    </div>
  {/if}
</div>

<style>
  .collapsible-section {
    width: 100%;
  }

  .section-header-wrapper {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--spacing-md);
    width: 100%;
  }

  .section-header {
    display: inline-flex;
    align-items: center;
    gap: var(--spacing-sm);
    background: var(--surface-muted);
    border: none;
    color: var(--text-on-primary);
    font-size: var(--font-size-sm);
    font-weight: var(--font-weight-medium);
    padding: var(--spacing-sm) var(--spacing-md);
    border-radius: var(--radius-sm);
    cursor: pointer;
    transition: background var(--transition-normal);
    text-align: left;
    flex-shrink: 1;
    min-width: 0;
  }

  .section-header:hover {
    background: var(--surface-muted-strong);
  }

  .section-title {
    flex: 1;
    min-width: 0;
  }

  .section-title[ondblclick] {
    cursor: text;
  }

  @media (hover: hover) and (pointer: fine) {
    /* On desktop, show hover effect for double-click */
    .section-title[ondblclick]:hover {
      opacity: 0.8;
    }
  }

  .title-input {
    flex: 1;
    min-width: 0;
    background: var(--surface-light);
    border: var(--stroke-thin) solid var(--text-on-primary);
    border-radius: var(--radius-sm);
    padding: var(--spacing-xs) var(--spacing-sm);
    color: var(--text-on-primary);
    font-size: var(--font-size-sm);
    font-weight: var(--font-weight-medium);
    outline: none;
    transition: border-color var(--transition-normal);
  }

  .title-input:focus {
    border-color: var(--text-on-primary);
    box-shadow: 0 0 0 2px rgba(255, 255, 255, 0.1);
  }

  .section-count {
    color: var(--text-muted);
    font-size: var(--font-size-xs);
    flex-shrink: 0;
  }

  .chevron {
    width: var(--icon-xs);
    height: var(--icon-xs);
    transition: transform var(--transition-normal);
    transform: rotate(0deg);
    flex-shrink: 0;
  }

  .chevron.expanded {
    transform: rotate(90deg);
  }

  .section-content {
    margin-bottom: var(--spacing-lg);
    width: 100%;
  }

  .action-buttons {
    display: flex;
    align-items: center;
    gap: var(--spacing-xs);
    flex-shrink: 0;
  }

  .edit-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    background: transparent;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    padding: var(--spacing-sm);
    border-radius: var(--radius-sm);
    transition: all var(--transition-normal);
    flex-shrink: 0;
    height: var(--button-height-sm);
    width: var(--button-height-sm);
  }

  .edit-btn svg {
    width: var(--icon-xs);
    height: var(--icon-xs);
  }

  .edit-btn:hover {
    background: var(--surface-muted);
    color: var(--text-on-primary);
  }

  .move-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    background: transparent;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    padding: var(--spacing-sm);
    border-radius: var(--radius-sm);
    transition: all var(--transition-normal);
    flex-shrink: 0;
    height: var(--button-height-sm);
    width: var(--button-height-sm);
  }

  .move-btn svg {
    width: var(--icon-xs);
    height: var(--icon-xs);
  }

  .move-btn:hover {
    background: var(--surface-muted);
    color: var(--text-on-primary);
  }

  .delete-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    background: transparent;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    padding: var(--spacing-sm);
    border-radius: var(--radius-sm);
    transition: all var(--transition-normal);
    flex-shrink: 0;
    height: var(--button-height-sm);
    width: var(--button-height-sm);
  }

  .delete-btn svg {
    width: var(--icon-xs);
    height: var(--icon-xs);
  }

  .delete-btn:hover {
    background: var(--color-delete-bg);
    color: var(--color-delete);
  }
</style>

