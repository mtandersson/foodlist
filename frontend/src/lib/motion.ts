import { slide } from 'svelte/transition';
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

/** A row collapsing out of a list after its check animation has played. */
export function collapseOut(node: Element, { delay = 160, duration = 280 } = {}): TransitionConfig {
  return slide(node, { delay: motionMs(delay), duration: motionMs(duration), easing: cubicOut });
}
