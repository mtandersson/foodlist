import { backOut, cubicOut } from 'svelte/easing';
import type { TransitionConfig } from 'svelte/transition';

/** True when the user has asked the OS to minimise motion. */
export function prefersReducedMotion(): boolean {
  return typeof window !== 'undefined' &&
    typeof window.matchMedia === 'function' &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}

/**
 * Glide keyed rows to their new places when a list re-sorts (FLIP), without
 * Svelte's animate:, which pulls leaving rows out of flow. Call before the
 * DOM updates; the returned function runs the glide after it.
 */
export function captureRowPositions(container: HTMLElement | null | undefined): () => void {
  if (!container || prefersReducedMotion()) return () => {};
  const before = new Map<string, number>();
  for (const el of Array.from(container.querySelectorAll<HTMLElement>(':scope > [data-row-id]'))) {
    before.set(el.dataset.rowId!, el.getBoundingClientRect().top);
  }
  return () => {
    for (const el of Array.from(container.querySelectorAll<HTMLElement>(':scope > [data-row-id]'))) {
      const prev = before.get(el.dataset.rowId!);
      if (prev === undefined) continue;
      const dy = prev - el.getBoundingClientRect().top;
      if (Math.abs(dy) < 1) continue;
      // Rows are see-through inside a section card; give moving rows the card
      // colour, and keep the one travelling furthest on top.
      el.style.position = 'relative';
      el.style.background = 'var(--card-bg)';
      el.style.zIndex = String(Math.round(Math.abs(dy)));
      const anim = el.animate([{ transform: `translateY(${dy}px)` }, { transform: 'none' }], {
        duration: 380,
        easing: 'cubic-bezier(0.22, 1, 0.36, 1)',
      });
      const reset = () => { el.style.background = ''; el.style.zIndex = ''; el.style.position = ''; };
      anim.onfinish = reset;
      anim.oncancel = reset;
    }
  };
}

/** Scale a duration to zero under reduced motion. */
export function motionMs(ms: number): number {
  return prefersReducedMotion() ? 0 : ms;
}

/** A row dropping in from above with a light spring. */
export function dropIn(_node: Element, { delay = 0, duration = 420 } = {}): TransitionConfig {
  return {
    delay,
    duration: motionMs(duration),
    easing: backOut,
    css: (t) => `opacity: ${Math.min(1, t * 1.6)}; transform: translateY(${(1 - t) * -12}px) scale(${0.97 + 0.03 * t});`,
  };
}

/**
 * Two-phase height animation: the row fades while it holds its height, then
 * the gap closes (or, in reverse, the gap opens and then the row fades in),
 * so a half-clipped row is never visible.
 */
function heightFade(node: Element, delay: number, duration: number, fadeShare: number): TransitionConfig {
  const style = getComputedStyle(node);
  const h = parseFloat(style.height);
  const pt = parseFloat(style.paddingTop);
  const pb = parseFloat(style.paddingBottom);
  const mt = parseFloat(style.marginTop);
  const mb = parseFloat(style.marginBottom);
  const gap = 1 - fadeShare;
  return {
    delay: motionMs(delay),
    duration: motionMs(duration),
    css: (t) => {
      // t: 0 = gone, 1 = fully present
      const k = cubicOut(Math.min(1, t / gap));
      const o = Math.max(0, (t - gap) / fadeShare);
      return `overflow: hidden; opacity: ${o}; visibility: ${o > 0 ? 'visible' : 'hidden'}; height: ${k * h}px; padding-top: ${k * pt}px; padding-bottom: ${k * pb}px; margin-top: ${k * mt}px; margin-bottom: ${k * mb}px;`;
    },
  };
}

/** A row leaving a list: fade in place, then close the gap. */
export function collapseOut(node: Element, { delay = 0, duration = 380 } = {}): TransitionConfig {
  return heightFade(node, delay, duration, 0.4);
}


/** A row arriving in a list: open the gap, then fade the row in. */
export function growIn(node: Element, { delay = 0, duration = 420 } = {}): TransitionConfig {
  return heightFade(node, delay, duration, 0.45);
}
