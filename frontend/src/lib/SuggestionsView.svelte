<script lang="ts">
  import { flip } from 'svelte/animate';
  import { fade } from 'svelte/transition';
  import CategoryBadge from './CategoryBadge.svelte';
  import { formatShortRelative } from './relativeDate';
  import type { Suggestion } from './types';

  interface Props {
    suggestions: Suggestion[];
    onAdd: (suggestion: Suggestion) => void;
  }

  let { suggestions, onAdd }: Props = $props();
</script>

<div class="suggestions-view" role="list" aria-label="Förslag att handla">
  {#if suggestions.length === 0}
    <div class="empty-state">
      <p>Inga förslag just nu.</p>
      <p class="hint">När du börjar handla samma sak regelbundet dyker den upp här.</p>
    </div>
  {:else}
    {#each suggestions as suggestion (suggestion.id)}
      <div
        class="suggestion-row"
        role="listitem"
        animate:flip={{ duration: 300 }}
        transition:fade={{ duration: 200 }}
      >
        <div class="suggestion-main">
          <span class="suggestion-name">{suggestion.name}</span>
          <span class="suggestion-meta">
            <span class="relative-date" title={`Senast handlat ${new Date(suggestion.lastPurchasedAt).toLocaleString('sv-SE')}`}>
              {formatShortRelative(suggestion.lastPurchasedAt)}
            </span>
            {#if suggestion.categoryName}
              <CategoryBadge name={suggestion.categoryName} size="small" />
            {/if}
          </span>
        </div>
        <button
          type="button"
          class="add-btn"
          aria-label={`Lägg till ${suggestion.name} i listan`}
          onclick={() => onAdd(suggestion)}
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <line x1="12" y1="5" x2="12" y2="19"></line>
            <line x1="5" y1="12" x2="19" y2="12"></line>
          </svg>
        </button>
      </div>
    {/each}
  {/if}
</div>

<style>
  .suggestions-view {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-xs);
  }

  .empty-state {
    text-align: center;
    padding: var(--spacing-2xl) var(--spacing-lg);
    color: var(--text-on-primary);
    opacity: 0.85;
  }

  .empty-state .hint {
    font-size: var(--font-size-sm);
    opacity: 0.7;
    margin-top: var(--spacing-sm);
  }

  .suggestion-row {
    display: flex;
    align-items: center;
    gap: var(--spacing-md);
    padding: var(--spacing-md) var(--spacing-lg);
    background: var(--card-bg);
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-sm, var(--shadow-md));
    transition: background var(--transition-normal), transform var(--transition-fast);
  }

  .suggestion-row:hover {
    background: var(--surface-muted-strong, var(--card-bg));
  }

  .suggestion-main {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: var(--spacing-xs);
    min-width: 0;
  }

  .suggestion-name {
    font-size: var(--font-size-base);
    color: var(--text-primary);
    font-weight: var(--font-weight-medium);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .suggestion-meta {
    display: flex;
    align-items: center;
    gap: var(--spacing-sm);
    font-size: var(--font-size-xs);
    color: var(--text-secondary);
  }

  .relative-date {
    color: var(--text-muted);
    font-variant-numeric: tabular-nums;
  }

  .add-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: var(--icon-xl);
    height: var(--icon-xl);
    border-radius: var(--radius-full);
    border: none;
    background: var(--primary-color);
    color: white;
    cursor: pointer;
    flex-shrink: 0;
    transition: transform var(--transition-fast), background var(--transition-normal);
  }

  .add-btn svg {
    width: var(--icon-md);
    height: var(--icon-md);
  }

  .add-btn:hover {
    transform: scale(1.05);
  }

  .add-btn:active {
    transform: scale(0.95);
  }
</style>
