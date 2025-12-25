<script lang="ts">
  import { createEventDispatcher } from 'svelte';

  interface Props {
    value?: 'normal' | 'categories';
  }

  let { value = 'normal' }: Props = $props();

  const dispatch = createEventDispatcher<{ change: 'normal' | 'categories' }>();

  function select(mode: 'normal' | 'categories') {
    if (mode !== value) {
      dispatch('change', mode);
    }
  }
</script>

<div class="mode-switch" role="group" aria-label="Visa-läge">
  <button
    type="button"
    class:selected={value === 'normal'}
    aria-pressed={value === 'normal'}
    onclick={() => select('normal')}
  >
    <span class="icon">☰</span>
    Normal
  </button>
  <button
    type="button"
    class:selected={value === 'categories'}
    aria-pressed={value === 'categories'}
    onclick={() => select('categories')}
  >
    <span class="icon">📁</span>
    Kategorier
  </button>
</div>

<style>
  .mode-switch {
    display: inline-flex;
    background: var(--surface-muted);
    border-radius: var(--radius-full);
    padding: var(--spacing-xs);
    gap: var(--spacing-xs);
    box-shadow: var(--shadow-inset);
  }

  .mode-switch button {
    display: inline-flex;
    align-items: center;
    gap: var(--spacing-sm);
    border: none;
    background: transparent;
    padding: var(--spacing-sm) var(--spacing-md);
    border-radius: var(--radius-full);
    color: var(--primary-color);
    cursor: pointer;
    transition: all var(--transition-normal);
    font-weight: var(--font-weight-semibold);
  }

  .mode-switch button.selected {
    background: var(--card-bg);
    color: var(--text-primary);
    box-shadow: var(--shadow-md);
  }

  .mode-switch .icon {
    font-size: var(--font-size-sm);
    line-height: var(--line-height-tight);
  }
</style>

