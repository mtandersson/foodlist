<script lang="ts">
  import type { Category } from './types';

  interface Props {
    categories: Category[];
    onSelect: (categoryId: string) => void;
    onCancel: () => void;
    todoName: string;
  }

  let { categories, onSelect, onCancel, todoName }: Props = $props();

  function handleBackdropClick(e: MouseEvent) {
    if (e.target === e.currentTarget) {
      onCancel();
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      onCancel();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<div 
  class="modal-backdrop" 
  onclick={handleBackdropClick}
  role="dialog"
  aria-modal="true"
  aria-labelledby="modal-title"
>
  <div class="modal-content">
    <div class="modal-header">
      <h2 id="modal-title">Välj kategori</h2>
      <button 
        class="close-btn" 
        onclick={onCancel}
        aria-label="Stäng"
      >
        ✕
      </button>
    </div>
    
    <div class="modal-subheader">
      {todoName}
    </div>
    
    <div class="category-list">
      {#each categories as category (category.id)}
        <button
          class="category-option"
          onclick={() => onSelect(category.id)}
        >
          <span class="category-name">{category.name}</span>
        </button>
      {/each}
    </div>
    
    <div class="modal-footer">
      <button class="cancel-btn" onclick={onCancel}>
        Avbryt
      </button>
    </div>
  </div>
</div>

<style>
  .modal-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    padding: var(--spacing-lg);
    animation: fadeIn 0.2s ease-out;
  }

  @keyframes fadeIn {
    from {
      opacity: 0;
    }
    to {
      opacity: 1;
    }
  }

  .modal-content {
    background: var(--card-bg);
    border-radius: var(--radius-lg);
    max-width: 400px;
    width: 100%;
    max-height: 80vh;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
    animation: slideUp 0.3s ease-out;
  }

  @keyframes slideUp {
    from {
      transform: translateY(20px);
      opacity: 0;
    }
    to {
      transform: translateY(0);
      opacity: 1;
    }
  }

  .modal-header {
    padding: var(--spacing-xl);
    border-bottom: 1px solid var(--surface-light);
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--spacing-md);
  }

  .modal-header h2 {
    margin: 0;
    font-size: var(--font-size-lg);
    color: var(--text-primary);
    flex: 1;
  }

  .modal-subheader {
    padding: 0 var(--spacing-xl) var(--spacing-lg);
    font-size: var(--font-size-base);
    color: var(--text-secondary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .close-btn {
    background: transparent;
    border: none;
    font-size: var(--font-size-xl);
    color: var(--text-muted);
    cursor: pointer;
    padding: var(--spacing-xs);
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all var(--transition-normal);
    border-radius: var(--radius-sm);
    width: 32px;
    height: 32px;
    flex-shrink: 0;
  }

  .close-btn:hover {
    background: var(--surface-light);
    color: var(--text-primary);
  }

  .category-list {
    padding: var(--spacing-md);
    overflow-y: auto;
    flex: 1;
  }

  .category-option {
    width: 100%;
    padding: var(--spacing-lg);
    background: var(--surface-light);
    border: 2px solid transparent;
    border-radius: var(--radius-md);
    margin-bottom: var(--spacing-sm);
    cursor: pointer;
    text-align: left;
    transition: all var(--transition-normal);
    font-size: var(--font-size-base);
    color: var(--text-secondary);
  }

  .category-option:hover {
    background: var(--primary-color);
    color: white;
    transform: translateX(4px);
  }

  .category-option:active {
    transform: translateX(4px) scale(0.98);
  }

  .category-name {
    font-weight: var(--font-weight-medium);
  }

  .modal-footer {
    padding: var(--spacing-lg);
    border-top: 1px solid var(--surface-light);
  }

  .cancel-btn {
    width: 100%;
    padding: var(--spacing-lg);
    background: var(--surface-light);
    border: none;
    border-radius: var(--radius-md);
    cursor: pointer;
    font-size: var(--font-size-base);
    color: var(--text-secondary);
    transition: all var(--transition-normal);
    font-weight: var(--font-weight-medium);
  }

  .cancel-btn:hover {
    background: var(--surface-muted);
  }

  .cancel-btn:active {
    transform: scale(0.98);
  }

  /* Mobile optimizations */
  @media (max-width: 768px) {
    .modal-backdrop {
      padding: 0;
    }

    .modal-content {
      max-width: 100%;
      max-height: 100%;
      border-radius: 0;
    }

    .category-option {
      padding: var(--spacing-xl);
      font-size: var(--font-size-lg);
    }
  }
</style>

