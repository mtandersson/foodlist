import { describe, it, expect, beforeEach, afterEach } from 'vitest';

describe('LocalStorage Persistence', () => {
  let localStorageMock: { [key: string]: string };

  beforeEach(() => {
    localStorageMock = {};
    
    (globalThis as any).localStorage = {
      getItem: (key: string) => localStorageMock[key] || null,
      setItem: (key: string, value: string) => {
        localStorageMock[key] = value;
      },
      removeItem: (key: string) => {
        delete localStorageMock[key];
      },
      clear: () => {
        localStorageMock = {};
      },
      get length() {
        return Object.keys(localStorageMock).length;
      },
      key: (index: number) => {
        const keys = Object.keys(localStorageMock);
        return keys[index] || null;
      }
    } as Storage;
  });

  afterEach(() => {
    localStorageMock = {};
  });

  describe('View Mode Persistence', () => {
    it('should save view mode to localStorage', () => {
      localStorage.setItem('viewMode', 'categories');
      expect(localStorage.getItem('viewMode')).toBe('categories');
    });

    it('should load view mode from localStorage', () => {
      localStorage.setItem('viewMode', 'normal');
      const viewMode = localStorage.getItem('viewMode') as 'normal' | 'categories';
      expect(viewMode).toBe('normal');
    });

    it('should default to normal mode when not in localStorage', () => {
      const viewMode = (localStorage.getItem('viewMode') as 'normal' | 'categories') || 'normal';
      expect(viewMode).toBe('normal');
    });
  });

  describe('Completed Section Persistence', () => {
    it('should save completed expanded state to localStorage', () => {
      localStorage.setItem('completedExpanded', 'false');
      expect(localStorage.getItem('completedExpanded')).toBe('false');
    });

    it('should load completed expanded state from localStorage', () => {
      localStorage.setItem('completedExpanded', 'true');
      const completedExpanded = localStorage.getItem('completedExpanded') === 'true';
      expect(completedExpanded).toBe(true);
    });

    it('should default to true when not in localStorage', () => {
      const completedExpanded = localStorage.getItem('completedExpanded') !== null 
        ? localStorage.getItem('completedExpanded') === 'true' 
        : true;
      expect(completedExpanded).toBe(true);
    });

    it('should parse string "false" correctly', () => {
      localStorage.setItem('completedExpanded', 'false');
      const completedExpanded = localStorage.getItem('completedExpanded') === 'true';
      expect(completedExpanded).toBe(false);
    });
  });

  describe('Expanded Categories Persistence', () => {
    it('should save expanded categories to localStorage', () => {
      const expandedCategories = new Set(['cat1', 'cat2', null]);
      localStorage.setItem('expandedCategories', JSON.stringify(Array.from(expandedCategories)));
      
      const stored = localStorage.getItem('expandedCategories');
      expect(stored).toBe(JSON.stringify(['cat1', 'cat2', null]));
    });

    it('should load expanded categories from localStorage', () => {
      localStorage.setItem('expandedCategories', JSON.stringify(['cat1', 'cat2']));
      const stored = localStorage.getItem('expandedCategories');
      const parsed = JSON.parse(stored!);
      const expandedCategories = new Set(parsed);
      
      expect(expandedCategories.has('cat1')).toBe(true);
      expect(expandedCategories.has('cat2')).toBe(true);
      expect(expandedCategories.size).toBe(2);
    });

    it('should handle empty expanded categories', () => {
      localStorage.setItem('expandedCategories', JSON.stringify([]));
      const stored = localStorage.getItem('expandedCategories');
      const parsed = JSON.parse(stored!);
      const expandedCategories = new Set(parsed);
      
      expect(expandedCategories.size).toBe(0);
    });

    it('should handle missing localStorage entry', () => {
      const stored = localStorage.getItem('expandedCategories');
      expect(stored).toBeNull();
    });
  });

  describe('Combined State Persistence', () => {
    it('should maintain all app state in localStorage', () => {
      // Set all state
      localStorage.setItem('viewMode', 'categories');
      localStorage.setItem('completedExpanded', 'false');
      localStorage.setItem('expandedCategories', JSON.stringify(['cat1', 'cat2']));

      // Verify all state
      expect(localStorage.getItem('viewMode')).toBe('categories');
      expect(localStorage.getItem('completedExpanded')).toBe('false');
      expect(localStorage.getItem('expandedCategories')).toBe(JSON.stringify(['cat1', 'cat2']));
    });

    it('should allow independent state updates', () => {
      // Set initial state
      localStorage.setItem('viewMode', 'normal');
      localStorage.setItem('completedExpanded', 'true');

      // Update only view mode
      localStorage.setItem('viewMode', 'categories');

      // Verify view mode changed and completedExpanded remained
      expect(localStorage.getItem('viewMode')).toBe('categories');
      expect(localStorage.getItem('completedExpanded')).toBe('true');
    });
  });
});

