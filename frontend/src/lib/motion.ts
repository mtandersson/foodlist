import { backOut, cubicOut } from 'svelte/easing';
import type { TransitionConfig } from 'svelte/transition';

/** True when the user has asked the OS to minimise motion. */
export function prefersReducedMotion(): boolean {
  return typeof window !== 'undefined' &&
    typeof window.matchMedia === 'function' &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches;
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
      return `overflow: hidden; opacity: ${o}; height: ${k * h}px; padding-top: ${k * pt}px; padding-bottom: ${k * pb}px; margin-top: ${k * mt}px; margin-bottom: ${k * mb}px;`;
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
