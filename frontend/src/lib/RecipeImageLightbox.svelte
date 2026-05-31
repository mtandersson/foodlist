<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { fade } from 'svelte/transition';
  import { enablePinchZoom } from './viewportMeta';

  interface Props {
    src: string;
    alt?: string;
    onClose: () => void;
  }

  let { src, alt = '', onClose }: Props = $props();

  // The lightbox does NOT implement pinch-zoom in JavaScript. We
  // unlock the viewport meta for the lifetime of this component and
  // let iOS Safari / Android Chrome handle pinch, double-tap, and
  // panning natively. The fullscreen overlay covers everything else
  // so the visual viewport zoom effectively zooms the image alone.
  // See ./viewportMeta.ts for the rationale.
  let restoreViewport: () => void = () => {};

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault();
      onClose();
    }
  }

  let prevBodyOverflow = '';
  let prevBodyOverscroll = '';

  onMount(() => {
    restoreViewport = enablePinchZoom();
    document.addEventListener('keydown', onKeydown);
    prevBodyOverflow = document.body.style.overflow;
    prevBodyOverscroll = document.body.style.overscrollBehavior;
    document.body.style.overflow = 'hidden';
    document.body.style.overscrollBehavior = 'contain';
  });

  onDestroy(() => {
    restoreViewport();
    document.removeEventListener('keydown', onKeydown);
    document.body.style.overflow = prevBodyOverflow;
    document.body.style.overscrollBehavior = prevBodyOverscroll;
  });

  function onBackdropClick(e: MouseEvent) {
    // Only a tap on the backdrop itself dismisses; the close button
    // and the image stop propagation via their own positioning.
    // Tapping the image directly is the user's pinch surface, not
    // a dismiss target.
    if (e.target === e.currentTarget) onClose();
  }
</script>

<div
  class="lightbox"
  role="dialog"
  aria-modal="true"
  aria-label="Receptbild"
  onclick={onBackdropClick}
  transition:fade={{ duration: 150 }}
>
  <button
    type="button"
    class="close"
    onclick={onClose}
    aria-label="Stäng bild"
  >×</button>
  <img {src} {alt} draggable="false" />
</div>

<style>
  .lightbox {
    position: fixed;
    inset: 0;
    z-index: 9999;
    background: rgba(0, 0, 0, 0.95);
    display: flex;
    align-items: center;
    justify-content: center;
    overscroll-behavior: contain;
    /* No `touch-action: none` here: native pinch-zoom works through
       the default touch action. Suppressing it would also kill the
       browser's built-in double-tap-to-zoom and momentum panning. */
  }

  .lightbox img {
    max-width: 100vw;
    max-height: 100vh;
    width: auto;
    height: auto;
    object-fit: contain;
    /* Allow the user to pinch the image specifically; modern browsers
       interpret pinch-zoom on the visual viewport because the
       <meta viewport> we install on mount permits it. */
    -webkit-user-drag: none;
    user-select: none;
    -webkit-user-select: none;
  }

  .close {
    position: absolute;
    top: max(env(safe-area-inset-top), var(--spacing-md));
    right: max(env(safe-area-inset-right), var(--spacing-md));
    width: 44px;
    height: 44px;
    border-radius: 50%;
    border: none;
    background: rgba(255, 255, 255, 0.15);
    color: white;
    font-size: 28px;
    line-height: 1;
    cursor: pointer;
    /* Above the image even when the browser pans/scales it; the
       button stays in viewport-fixed coordinates because of
       `position: absolute` inside the fixed lightbox. */
    z-index: 1;
    backdrop-filter: blur(8px);
    -webkit-backdrop-filter: blur(8px);
  }

  .close:hover,
  .close:focus-visible {
    background: rgba(255, 255, 255, 0.28);
  }
</style>
