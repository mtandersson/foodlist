<script lang="ts">
  import { slide } from 'svelte/transition';

  interface Props {
    title: string;
    count?: number;
    expanded?: boolean;
    onToggle: () => void;
    children?: import('svelte').Snippet;
  }

  let { title, count, expanded = true, onToggle, children }: Props = $props();
</script>

<div class="collapsible-section">
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

  .section-header {
    display: inline-flex;
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
    text-align: left;
  }

  .section-header:hover {
    background: var(--surface-muted-strong);
  }

  .section-title {
    flex: 1;
  }

  .section-count {
    color: var(--text-muted);
    font-size: 12px;
  }

  .chevron {
    width: 16px;
    height: 16px;
    transition: transform 0.2s ease;
    transform: rotate(0deg);
    flex-shrink: 0;
  }

  .chevron.expanded {
    transform: rotate(90deg);
  }

  .section-content {
    margin-bottom: 16px;
    width: 100%;
  }
</style>

