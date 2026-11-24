<script lang="ts">
  import { createEventDispatcher } from 'svelte';

  export type ViewMode = 'normal' | 'categories' | 'suggestions' | 'recipes';

  interface Props {
    value?: ViewMode;
    showSuggestions?: boolean;
    showRecipes?: boolean;
  }

  let { value = 'normal', showSuggestions = false, showRecipes = false }: Props = $props();

  const dispatch = createEventDispatcher<{ change: ViewMode }>();

  const ALL: { mode: ViewMode; label: string; title?: string }[] = [
    { mode: 'normal', label: 'Normal' },
    { mode: 'categories', label: 'Kategorier' },
    { mode: 'suggestions', label: 'Förslag', title: 'Förslag på saker att handla' },
    { mode: 'recipes', label: 'Recept', title: 'Recept' },
  ];

  let modes = $derived(
    ALL.filter(
      (m) =>
        (m.mode !== 'suggestions' || showSuggestions) && (m.mode !== 'recipes' || showRecipes)
    )
  );
  let selectedIndex = $derived(Math.max(0, modes.findIndex((m) => m.mode === value)));

  function select(mode: ViewMode) {
    if (mode !== value) {
      dispatch('change', mode);
    }
  }
</script>

<div
  class="mode-switch"
  role="group"
  aria-label="Visa-läge"
  style="--count: {modes.length}; --index: {selectedIndex}"
>
  <span class="indicator" aria-hidden="true"></span>
  {#each modes as m (m.mode)}
    <button
      type="button"
      class:selected={value === m.mode}
      aria-pressed={value === m.mode}
      aria-label={m.label}
      title={m.title}
      onclick={() => select(m.mode)}
    >
      <svg class="icon" viewBox="0 0 24 24" aria-hidden="true">
        {#if m.mode === 'normal'}<path d="M4 6h16M4 12h16M4 18h10"/>
        {:else if m.mode === 'categories'}<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>
        {:else if m.mode === 'suggestions'}<path d="M9 18h6M10 21h4M12 3a6 6 0 0 0-3.5 10.9c.6.4 1 1.1 1 1.8V16h5v-.3c0-.7.4-1.4 1-1.8A6 6 0 0 0 12 3z"/>
        {:else if m.mode === 'recipes'}<path d="M4 5a2 2 0 0 1 2-2h13v16H6a2 2 0 0 0-2 2zM4 21V5M8 7h7"/>
        {/if}
      </svg>
      <span class="label">{m.label}</span>
    </button>
  {/each}
</div>

<style>
  .mode-switch {
    position: relative;
    display: inline-grid;
    grid-template-columns: repeat(var(--count), minmax(0, 1fr));
    background: var(--surface-muted);
    border-radius: var(--radius-full);
    padding: var(--spacing-xs);
    box-shadow: var(--shadow-inset);
  }

  .indicator {
    position: absolute;
    top: var(--spacing-xs);
    bottom: var(--spacing-xs);
    left: var(--spacing-xs);
    width: calc((100% - 2 * var(--spacing-xs)) / var(--count));
    background: var(--card-bg);
    border-radius: var(--radius-full);
    box-shadow: var(--shadow-sm);
    transform: translateX(calc(var(--index) * 100%));
    transition: transform 0.35s var(--ease-spring);
  }

  .mode-switch button {
    position: relative;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--spacing-sm);
    border: none;
    background: transparent;
    min-height: 36px;
    padding: var(--spacing-sm) var(--spacing-md);
    border-radius: var(--radius-full);
    color: var(--text-secondary);
    cursor: pointer;
    transition: color var(--transition-normal);
    font-weight: var(--font-weight-semibold);
    white-space: nowrap;
  }

  .mode-switch button.selected {
    color: var(--text-primary);
  }

  .mode-switch button:focus-visible {
    outline: 2px solid var(--primary-color);
    outline-offset: 2px;
  }

  .mode-switch .icon {
    width: 18px;
    height: 18px;
    fill: none;
    stroke: currentColor;
    stroke-width: 2;
    stroke-linecap: round;
    stroke-linejoin: round;
    flex: none;
  }

  /* Hide text labels on mobile, show only icons */
  @media (max-width: 768px) {
    .mode-switch .label {
      display: none;
    }

    .mode-switch button {
      gap: 0;
      padding: var(--spacing-sm);
      min-width: 40px;
    }
  }
</style>
