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
    border-radius: 999px;
    padding: 4px;
    gap: 4px;
    box-shadow: inset 0 1px 2px rgba(0, 0, 0, 0.05);
  }

  .mode-switch button {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    border: none;
    background: transparent;
    padding: 8px 12px;
    border-radius: 999px;
    color: var(--text-on-primary);
    cursor: pointer;
    transition: all 0.2s ease;
    font-weight: 600;
  }

  .mode-switch button.selected {
    background: var(--card-bg);
    color: var(--text-primary);
    box-shadow: 0 2px 6px rgba(0, 0, 0, 0.1);
  }

  .mode-switch .icon {
    font-size: 14px;
    line-height: 1;
  }
</style>

