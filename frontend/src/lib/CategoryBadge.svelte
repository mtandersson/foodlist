<script lang="ts">
  interface Props {
    name: string;
    showIcon?: boolean;
    size?: 'small' | 'normal';
  }

  let { name, showIcon = false, size = 'normal' }: Props = $props();
</script>

<span class="category-badge" class:small={size === 'small'}>
  {#if showIcon}
    <svg class="category-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
    </svg>
  {/if}
  <span class="category-name">{name}</span>
</span>

<style>
  .category-badge {
    display: inline-flex;
    align-items: center;
    gap: var(--spacing-xs);
    padding: var(--spacing-xs) var(--spacing-md);
    background: var(--primary-color);
    color: white;
    border-radius: var(--radius-full);
    font-size: var(--font-size-xs);
    font-weight: var(--font-weight-medium);
    line-height: var(--line-height-normal);
    white-space: nowrap;
    flex-shrink: 0;
  }

  .category-badge.small {
    padding: 2px var(--spacing-sm);
    gap: 3px;
    font-size: var(--font-size-xs);
  }

  .category-icon {
    width: 12px;
    height: 12px;
    opacity: 0.9;
    flex-shrink: 0;
  }

  .category-badge.small .category-icon {
    width: 11px;
    height: 11px;
  }

  .category-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
