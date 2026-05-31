import '@testing-library/jest-dom';

// jsdom doesn't implement Element.animate; Svelte transitions
// (fade/slide/etc.) call it during mount. Stub a no-op so component
// tests can render without crashing.
if (typeof Element !== 'undefined' && !Element.prototype.animate) {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  (Element.prototype as any).animate = function () {
    return {
      cancel() {},
      finish() {},
      addEventListener() {},
      removeEventListener() {},
      finished: Promise.resolve(),
    };
  };
}
