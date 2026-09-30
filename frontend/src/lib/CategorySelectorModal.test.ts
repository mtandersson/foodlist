import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import CategorySelectorModal from './CategorySelectorModal.svelte';
import type { Category } from './types';

const categories: Category[] = Array.from({ length: 30 }, (_, index) => ({
  id: `category-${index}`,
  name: `Category ${index + 1}`,
  createdAt: '2026-09-30',
  sortOrder: 30000 - index * 1000,
}));

function renderPicker(options: { categories?: Category[]; todoName?: string } = {}) {
  const onSelect = vi.fn();
  const onCancel = vi.fn();
  const result = render(CategorySelectorModal, {
    props: { categories, todoName: 'Milk', onSelect, onCancel, ...options },
  });
  return { ...result, onSelect, onCancel };
}

describe('CategorySelectorModal', () => {
  it('shows the food name and categories in the supplied order', () => {
    renderPicker();
    expect(screen.getByRole('dialog', { name: 'Välj kategori' })).toBeInTheDocument();
    expect(screen.getByText('Milk')).toBeInTheDocument();
    const options = screen.getAllByRole('button', { name: /^Category / });
    expect(options.map(option => option.textContent?.trim())).toEqual(categories.map(category => category.name));
  });

  it('selects the final category in a long list without cancelling', async () => {
    const { onSelect, onCancel } = renderPicker();
    await fireEvent.click(screen.getByRole('button', { name: 'Category 30' }));
    expect(onSelect).toHaveBeenCalledExactlyOnceWith('category-29');
    expect(onCancel).not.toHaveBeenCalled();
  });

  it.each(['Stäng', 'Avbryt'])('cancels using %s without selecting a category', async name => {
    const { onSelect, onCancel } = renderPicker();
    await fireEvent.click(screen.getByRole('button', { name }));
    expect(onCancel).toHaveBeenCalledOnce();
    expect(onSelect).not.toHaveBeenCalled();
  });

  it('cancels when the backdrop is clicked', async () => {
    const { onCancel, onSelect } = renderPicker();
    await fireEvent.click(screen.getByRole('dialog'));
    expect(onCancel).toHaveBeenCalledOnce();
    expect(onSelect).not.toHaveBeenCalled();
  });

  it('keeps the picker open when its food name is clicked', async () => {
    const { onCancel, onSelect } = renderPicker();
    await fireEvent.click(screen.getByText('Milk'));
    expect(onCancel).not.toHaveBeenCalled();
    expect(onSelect).not.toHaveBeenCalled();
  });

  it('cancels when Escape is pressed', async () => {
    const { onCancel, onSelect } = renderPicker();
    await fireEvent.keyDown(window, { key: 'Escape' });
    expect(onCancel).toHaveBeenCalledOnce();
    expect(onSelect).not.toHaveBeenCalled();
  });

  it('allows dismissal when no categories are supplied', async () => {
    const { onCancel, onSelect } = renderPicker({ categories: [] });
    expect(screen.queryByRole('button', { name: /^Category / })).not.toBeInTheDocument();
    await fireEvent.click(screen.getByRole('button', { name: 'Avbryt' }));
    expect(onCancel).toHaveBeenCalledOnce();
    expect(onSelect).not.toHaveBeenCalled();
  });

  it('displays long food and category names', () => {
    const todoName = 'Milk and other ingredients for a large family dinner';
    const categoryName = 'A long category name with several words';
    renderPicker({ todoName, categories: [{ ...categories[0], name: categoryName }] });
    expect(screen.getByText(todoName)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: categoryName })).toBeInTheDocument();
  });
});
