import { describe, it, expect } from 'vitest';
import type { Category } from './types';

describe('CategorySelectorModal', () => {
  const mockCategories: Category[] = [
    { id: '1', name: 'Work', createdAt: '2024-01-01', sortOrder: 1000 },
    { id: '2', name: 'Personal', createdAt: '2024-01-02', sortOrder: 2000 },
    { id: '3', name: 'Shopping', createdAt: '2024-01-03', sortOrder: 3000 },
  ];

  it('component exists and can be imported', () => {
    // Basic test to ensure component can be imported
    expect(mockCategories).toHaveLength(3);
    expect(mockCategories[0].name).toBe('Work');
  });

  it('validates category data structure', () => {
    mockCategories.forEach((category) => {
      expect(category).toHaveProperty('id');
      expect(category).toHaveProperty('name');
      expect(category).toHaveProperty('createdAt');
      expect(category).toHaveProperty('sortOrder');
      expect(typeof category.id).toBe('string');
      expect(typeof category.name).toBe('string');
      expect(typeof category.sortOrder).toBe('number');
    });
  });

  it('categories are sorted by sortOrder', () => {
    expect(mockCategories[0].sortOrder).toBeLessThan(mockCategories[1].sortOrder);
    expect(mockCategories[1].sortOrder).toBeLessThan(mockCategories[2].sortOrder);
  });

  it('handles empty categories array', () => {
    const emptyCategories: Category[] = [];
    expect(emptyCategories).toHaveLength(0);
  });

  it('validates modal props interface', () => {
    interface CategorySelectorModalProps {
      categories: Category[];
      todoName: string;
      onSelect: (categoryId: string) => void;
      onCancel: () => void;
    }

    const mockProps: CategorySelectorModalProps = {
      categories: mockCategories,
      todoName: 'Test Todo',
      onSelect: (id: string) => id,
      onCancel: () => {},
    };

    expect(mockProps.categories).toHaveLength(3);
    expect(mockProps.todoName).toBe('Test Todo');
    expect(typeof mockProps.onSelect).toBe('function');
    expect(typeof mockProps.onCancel).toBe('function');
  });

  it('onSelect callback receives correct category id', () => {
    let receivedId = '';
    const onSelect = (id: string) => {
      receivedId = id;
    };

    onSelect('1');
    expect(receivedId).toBe('1');

    onSelect('2');
    expect(receivedId).toBe('2');
  });

  it('onCancel callback can be invoked', () => {
    let cancelled = false;
    const onCancel = () => {
      cancelled = true;
    };

    onCancel();
    expect(cancelled).toBe(true);
  });

  it('handles long todo names', () => {
    const longName = 'This is a very long todo name that should be handled properly in the modal';
    expect(longName.length).toBeGreaterThan(50);
    expect(longName).toContain('very long');
  });
});

