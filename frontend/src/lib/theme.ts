/**
 * Theme management for the application
 * Supports: 'light', 'dark', and 'auto' (follows system preference)
 */

export type ThemeMode = 'light' | 'dark' | 'auto';

const THEME_STORAGE_KEY = 'theme-mode';

/**
 * Get the stored theme preference, or 'auto' if none is set
 */
export function getStoredTheme(): ThemeMode {
  if (typeof localStorage === 'undefined') return 'auto';
  const stored = localStorage.getItem(THEME_STORAGE_KEY);
  if (stored === 'light' || stored === 'dark' || stored === 'auto') {
    return stored;
  }
  return 'auto';
}

/**
 * Store the theme preference
 */
export function storeTheme(theme: ThemeMode): void {
  if (typeof localStorage === 'undefined') return;
  localStorage.setItem(THEME_STORAGE_KEY, theme);
}

/**
 * Get the effective theme based on mode and system preference
 */
export function getEffectiveTheme(mode: ThemeMode): 'light' | 'dark' {
  if (mode === 'light' || mode === 'dark') return mode;
  
  // Auto mode: check system preference
  if (typeof window === 'undefined') return 'light';
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

/**
 * Apply the theme to the document
 */
export function applyTheme(mode: ThemeMode): void {
  const effectiveTheme = getEffectiveTheme(mode);
  
  if (typeof document === 'undefined') return;
  
  // Set data attribute for theme
  document.documentElement.setAttribute('data-theme', effectiveTheme);
  
  // Also set data-theme-mode to track the user's preference (light/dark/auto)
  document.documentElement.setAttribute('data-theme-mode', mode);

  // Keep the browser / OS chrome color in sync with the app's background.
  // Note: Installed PWAs on Android primarily use manifest.json's theme_color,
  // but many browsers still respect the meta tag (especially in-tab).
  requestAnimationFrame(() => {
    try {
      syncThemeColorMetaTag();
    } catch {
      // Ignore – cosmetic only
    }
  });
}

function syncThemeColorMetaTag(): void {
  if (typeof document === 'undefined' || typeof window === 'undefined') return;

  // Reuse existing tag or create one (some browsers only read it if present).
  let meta = document.querySelector('meta[name="theme-color"]') as HTMLMetaElement | null;
  if (!meta) {
    meta = document.createElement('meta');
    meta.name = 'theme-color';
    document.head.appendChild(meta);
  }

  const rootStyle = window.getComputedStyle(document.documentElement);
  const primaryGradient = rootStyle.getPropertyValue('--primary-gradient').trim();
  const primaryColor = rootStyle.getPropertyValue('--primary-color').trim();

  const color = pickTopColor(primaryGradient) ?? pickTopColor(primaryColor);
  if (color) meta.setAttribute('content', color);
}

function pickTopColor(value: string): string | null {
  if (!value) return null;

  // If it's already a single color (e.g. "#000000" in dark mode), use it.
  if (value.startsWith('#')) return value;
  if (value.startsWith('rgb(') || value.startsWith('rgba(')) return value;
  if (value.startsWith('hsl(') || value.startsWith('hsla(')) return value;

  // For gradients, use the first color stop to "fuse" with the status bar.
  const hex = value.match(/#[0-9a-fA-F]{3,8}/);
  if (hex?.[0]) return hex[0];

  const rgb = value.match(/rgba?\([^)]+\)/);
  if (rgb?.[0]) return rgb[0];

  return null;
}

/**
 * Initialize theme on app load
 */
export function initTheme(): ThemeMode {
  const mode = getStoredTheme();
  applyTheme(mode);
  
  // Listen for system theme changes when in auto mode
  if (typeof window !== 'undefined') {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    mediaQuery.addEventListener('change', () => {
      const currentMode = getStoredTheme();
      if (currentMode === 'auto') {
        applyTheme('auto');
      }
    });
  }
  
  return mode;
}

/**
 * Set theme and persist to localStorage
 */
export function setTheme(mode: ThemeMode): void {
  storeTheme(mode);
  applyTheme(mode);
}

