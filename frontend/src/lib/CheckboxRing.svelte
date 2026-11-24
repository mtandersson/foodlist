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
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
    <polyline class="tick" points="20 6 9 17 4 12"></polyline>
  </svg>
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
    transition: background var(--transition-normal), border-color var(--transition-normal);
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
    color: var(--on-primary);
    animation: fl-pop 0.3s var(--ease-spring);
  }

  .tick {
    stroke-dasharray: 24;
    stroke-dashoffset: 24;
    transition: stroke-dashoffset 0.2s ease-out 0.08s;
  }

  .checked .tick {
    stroke-dashoffset: 0;
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