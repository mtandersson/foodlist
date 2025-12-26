<script lang="ts">
  interface Props {
    checked?: boolean;
    size?: 'small' | 'normal';
  }

  let { checked = false, size = 'normal' }: Props = $props();
</script>

<div 
  class="checkbox-ring" 
  class:checked
  class:small={size === 'small'}
>
  {#if checked}
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
      <polyline points="20 6 9 17 4 12"></polyline>
    </svg>
  {/if}
</div>

<style>
  .checkbox-ring {
    width: var(--checkbox-size);
    height: var(--checkbox-size);
    border-radius: var(--position-half);
    border: var(--stroke-thin) solid var(--checkbox-border);
    background: transparent;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    transition: all var(--transition-normal);
  }

  .checkbox-ring.small {
    width: var(--icon-lg);
    height: var(--icon-lg);
  }

  .checkbox-ring:hover {
    border-color: var(--primary-color);
  }

  .checkbox-ring.checked {
    background: var(--primary-color);
    border-color: var(--primary-color);
    color: var(--text-on-primary);
  }

  .checkbox-ring svg {
    width: var(--icon-xs);
    height: var(--icon-xs);
  }

  .checkbox-ring.small svg {
    width: calc(var(--icon-xs) * 0.8);
    height: calc(var(--icon-xs) * 0.8);
  }
</style>

