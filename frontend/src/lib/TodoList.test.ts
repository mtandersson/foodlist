import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/svelte';
import type { Command, ServerMessage } from './types';

const transport = vi.hoisted(() => ({
  send: vi.fn(),
  receive: null as ((message: ServerMessage) => void) | null,
}));

// Keep the production component and store together; capture commands at
// the transport boundary without opening a network connection.
vi.mock('./websocket', () => ({
  ConnectionState: { CONNECTING: 'CONNECTING', CONNECTED: 'CONNECTED' },
  TodoWebSocket: class {
    send(command: Command) { transport.send(command); }
    onMessage(handler: (message: ServerMessage) => void) {
      transport.receive = handler;
      return () => {};
    }
    onConnectionChange() { return () => {}; }
    onAutocomplete() { return () => {}; }
    close() {}
  },
}));

import TodoList from './TodoList.svelte';

describe('new category form', () => {
  beforeEach(() => {
    localStorage.clear();
    transport.send.mockClear();
    transport.receive = null;
  });

  async function openForm(name = '  Fruit  ') {
    render(TodoList);
    transport.receive!({ type: 'StateRollup', todos: [], categories: [], listTitle: 'Shopping' });
    await fireEvent.click(screen.getByRole('button', { name: 'Menu' }));
    await fireEvent.click(screen.getByRole('button', { name: /Ny kategori/ }));
    const input = screen.getByPlaceholderText('Namn på ny kategori...');
    await waitFor(() => expect(input).toHaveFocus());
    await fireEvent.input(input, { target: { value: name } });
    return input;
  }

  it('discards a typed name when Cancel is clicked after the input loses focus', async () => {
    const input = await openForm();
    const cancel = screen.getByRole('button', { name: 'Avbryt' });
    await fireEvent.blur(input, { relatedTarget: cancel });
    await fireEvent.click(cancel);
    expect(transport.send).not.toHaveBeenCalled();
  });

  it('keeps an unfinished category when focus leaves the input', async () => {
    const input = await openForm();
    await fireEvent.blur(input);
    expect(transport.send).not.toHaveBeenCalled();
    expect(input).toBeVisible();
    expect(input).toHaveValue('  Fruit  ');
  });

  it('submits one trimmed category when Create is clicked after blur', async () => {
    const input = await openForm();
    const create = screen.getByRole('button', { name: 'Skapa' });
    await fireEvent.blur(input, { relatedTarget: create });
    expect(transport.send).not.toHaveBeenCalled();
    await fireEvent.click(create);
    expect(transport.send).toHaveBeenCalledTimes(1);
    expect(transport.send).toHaveBeenCalledWith(expect.objectContaining({ type: 'CreateCategory', name: 'Fruit' }));
  });

  it('submits one category with Enter, including subsequent blur', async () => {
    const input = await openForm();
    await fireEvent.keyDown(input, { key: 'Enter' });
    await fireEvent.blur(input);
    expect(transport.send).toHaveBeenCalledTimes(1);
    expect(transport.send).toHaveBeenCalledWith(expect.objectContaining({ type: 'CreateCategory', name: 'Fruit' }));
  });

  it('discards a typed name with Escape without submitting on subsequent blur', async () => {
    const input = await openForm();
    await fireEvent.keyDown(input, { key: 'Escape' });
    await fireEvent.blur(input);
    expect(transport.send).not.toHaveBeenCalled();
  });

  it('does not submit a whitespace-only name', async () => {
    const input = await openForm('   ');
    expect(screen.getByRole('button', { name: 'Skapa' })).toBeDisabled();
    await fireEvent.keyDown(input, { key: 'Enter' });
    await fireEvent.blur(input);
    expect(transport.send).not.toHaveBeenCalled();
  });
});
