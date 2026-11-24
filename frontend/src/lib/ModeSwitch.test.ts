import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import ModeSwitch from './ModeSwitch.svelte';

function renderSwitch(props: Record<string, unknown> = {}) {
  const change = vi.fn();
  const result = render(ModeSwitch, { props, events: { change } } as never);
  return { ...result, change };
}

describe('ModeSwitch', () => {
  it('offers only the modes that are enabled', () => {
    renderSwitch({ showRecipes: true });
    const names = screen.getAllByRole('button').map((b) => b.getAttribute('aria-label'));
    expect(names).toEqual(['Normal', 'Kategorier', 'Recept']);
  });

  it('marks the current mode as pressed', () => {
    renderSwitch({ value: 'categories', showSuggestions: true, showRecipes: true });
    expect(screen.getByRole('button', { name: 'Kategorier' })).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByRole('button', { name: 'Normal' })).toHaveAttribute('aria-pressed', 'false');
  });

  it('asks to change mode only when another mode is chosen', async () => {
    const { change } = renderSwitch({ value: 'normal', showSuggestions: true });
    await fireEvent.click(screen.getByRole('button', { name: 'Normal' }));
    expect(change).not.toHaveBeenCalled();
    await fireEvent.click(screen.getByRole('button', { name: 'Förslag' }));
    expect(change).toHaveBeenCalledOnce();
    expect(change.mock.calls[0][0].detail).toBe('suggestions');
  });
});
