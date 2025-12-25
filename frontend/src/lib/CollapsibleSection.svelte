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
    children?: import('svelte').Snippet;
  }

  let { title, count, expanded = true, onToggle, onDelete, onMoveUp, onMoveDown, children }: Props = $props();

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
      <span class="section-title">{title}</span>
      {#if count !== undefined}
        <span class="section-count">{count}</span>
      {/if}
    </button>
    <div class="action-buttons">
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

