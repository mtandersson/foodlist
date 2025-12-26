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

