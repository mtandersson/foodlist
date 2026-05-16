import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import SuggestionsView from './SuggestionsView.svelte';
import type { Suggestion } from './types';

const mk = (overrides: Partial<Suggestion> = {}): Suggestion => ({
  id: 'sug-1',
  name: 'Mjölk',
  categoryId: null,
  categoryName: null,
  lastPurchasedAt: '2026-05-10T12:00:00Z',
  purchaseCount: 4,
  avgIntervalSeconds: 7 * 86400,
  ...overrides,
});

describe('SuggestionsView', () => {
  it('shows an empty state when there are no suggestions', () => {
    render(SuggestionsView, {
      props: { suggestions: [], onAdd: vi.fn() },
    });
    expect(screen.getByText(/Inga förslag/i)).toBeInTheDocument();
  });

  it('renders a row per suggestion with name', () => {
    render(SuggestionsView, {
      props: {
        suggestions: [mk({ id: 'a', name: 'Mjölk' }), mk({ id: 'b', name: 'Bröd' })],
        onAdd: vi.fn(),
      },
    });
    expect(screen.getByText('Mjölk')).toBeInTheDocument();
    expect(screen.getByText('Bröd')).toBeInTheDocument();
  });

  it('renders the category badge when one is present', () => {
    render(SuggestionsView, {
      props: {
        suggestions: [mk({ categoryName: 'Mejeri' })],
        onAdd: vi.fn(),
      },
    });
    expect(screen.getByText('Mejeri')).toBeInTheDocument();
  });

  it('calls onAdd with the suggestion when the + button is clicked', async () => {
    const onAdd = vi.fn();
    const sg = mk({ id: 'x', name: 'Bananer' });
    render(SuggestionsView, { props: { suggestions: [sg], onAdd } });

    const btn = screen.getByRole('button', { name: /Lägg till Bananer/i });
    await fireEvent.click(btn);
    expect(onAdd).toHaveBeenCalledTimes(1);
    expect(onAdd).toHaveBeenCalledWith(sg);
  });

  it('renders each suggestion in its received order (parent is responsible for sort)', () => {
    render(SuggestionsView, {
      props: {
        suggestions: [
          mk({ id: '1', name: 'Alpha' }),
          mk({ id: '2', name: 'Bravo' }),
          mk({ id: '3', name: 'Charlie' }),
        ],
        onAdd: vi.fn(),
      },
    });
    const rows = screen.getAllByRole('listitem');
    expect(rows).toHaveLength(3);
  });
});
