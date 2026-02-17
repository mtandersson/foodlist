<script lang="ts">
  interface Props {
    clientVersion: string;
    serverVersion: string | null;
    onCancel: () => void;
  }

  let { clientVersion, serverVersion, onCancel }: Props = $props();

  function handleBackdropClick(e: MouseEvent | KeyboardEvent) {
    if (e.target === e.currentTarget) {
      onCancel();
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      onCancel();
    }
  }

  // Determine what version text to display
  function getVersionText(): string {
    // If server version is null/undefined (old server or not yet received)
    if (!serverVersion) {
      return `Client Version: ${clientVersion}`;
    }
    
    // If versions match
    if (clientVersion === serverVersion) {
      return `Version ${clientVersion}`;
    }
    
    // Versions differ - show both
    return `Client: ${clientVersion}\nServer: ${serverVersion}`;
  }

  const versionText = $derived(getVersionText());
</script>

<svelte:window onkeydown={handleKeydown} />

<div 
  class="modal-backdrop" 
  onclick={handleBackdropClick}
  onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onCancel(); } }}
  role="dialog"
  aria-modal="true"
  aria-labelledby="modal-title"
  tabindex="-1"
>
  <div class="modal-content">
    <div class="modal-header">
      <h2 id="modal-title">FoodList</h2>
      <button 
        class="close-btn" 
        onclick={onCancel}
        aria-label="Stäng"
      >
        ✕
      </button>
    </div>
    
    <div class="modal-body">
      <div class="version-text">
        {#each versionText.split('\n') as line}
          <div>{line}</div>
        {/each}
      </div>
    </div>
    
    <div class="modal-footer">
      <button class="close-button" onclick={onCancel}>
        Stäng
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

  .modal-body {
    padding: var(--spacing-xl);
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 120px;
  }

  .version-text {
    font-size: var(--font-size-base);
    color: var(--text-primary);
    text-align: center;
    line-height: 1.8;
    width: 100%;
  }

  .version-text div {
    margin-bottom: var(--spacing-sm);
    word-break: break-word;
  }

  .version-text div:last-child {
    margin-bottom: 0;
  }

  .modal-footer {
    padding: var(--spacing-lg);
    border-top: 1px solid var(--surface-light);
  }

  .close-button {
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
    min-height: 44px; /* iOS touch target minimum */
  }

  .close-button:hover {
    background: var(--surface-muted);
  }

  .close-button:active {
    transform: scale(0.98);
    background: var(--surface-muted);
  }

  /* Mobile optimizations */
  @media (max-width: 768px) {
    .modal-backdrop {
      padding: 0;
      align-items: flex-end; /* Slide up from bottom on mobile */
    }

    .modal-content {
      max-width: 100%;
      width: 100%;
      max-height: 90vh;
      border-radius: var(--radius-lg) var(--radius-lg) 0 0;
      margin-top: auto;
      animation: slideUpMobile 0.3s cubic-bezier(0.32, 0.72, 0, 1);
      box-shadow: 0 -4px 20px rgba(0, 0, 0, 0.15);
    }

    @keyframes slideUpMobile {
      from {
        transform: translateY(100%);
      }
      to {
        transform: translateY(0);
      }
    }

    .modal-header {
      padding: var(--spacing-xl) var(--spacing-lg);
      padding-top: calc(var(--spacing-lg) + env(safe-area-inset-top, 0px));
      min-height: 60px;
      border-bottom: 1px solid var(--border-color);
    }

    .modal-header h2 {
      font-size: var(--font-size-xl);
      font-weight: var(--font-weight-semibold);
      letter-spacing: -0.01em;
    }

    .close-btn {
      width: 44px; /* iOS touch target minimum */
      height: 44px;
      font-size: 24px;
      color: var(--text-primary);
    }

    .close-btn:active {
      background: var(--surface-muted);
    }

    .modal-body {
      padding: var(--spacing-3xl) var(--spacing-lg);
      min-height: 160px;
      flex: 1;
    }

    .version-text {
      font-size: var(--font-size-lg);
      line-height: 1.8;
      color: var(--text-primary);
    }

    .version-text div {
      margin-bottom: var(--spacing-md);
      padding: var(--spacing-xs) 0;
    }

    .version-text div:last-child {
      margin-bottom: 0;
    }

    .modal-footer {
      padding: var(--spacing-lg);
      padding-bottom: var(--spacing-lg);
      border-top: 1px solid var(--border-color);
    }

    .close-button {
      padding: var(--spacing-xl) var(--spacing-lg);
      font-size: var(--font-size-lg);
      font-weight: var(--font-weight-semibold);
      min-height: 52px; /* Larger touch target on mobile */
      background: var(--primary-color);
      color: var(--text-on-primary);
    }

    .close-button:hover,
    .close-button:active {
      background: var(--primary-color);
      opacity: var(--opacity-hover);
    }

    .close-button:active {
      transform: scale(0.98);
    }
  }

  /* Very small screens */
  @media (max-width: 375px) {
    .modal-header {
      padding: var(--spacing-lg);
      padding-top: calc(var(--spacing-md) + env(safe-area-inset-top, 0px));
    }

    .modal-header h2 {
      font-size: var(--font-size-lg);
    }

    .modal-body {
      padding: var(--spacing-2xl) var(--spacing-lg);
    }

    .version-text {
      font-size: var(--font-size-base);
    }

    .close-button {
      padding: var(--spacing-lg);
      font-size: var(--font-size-base);
    }
  }
</style>

