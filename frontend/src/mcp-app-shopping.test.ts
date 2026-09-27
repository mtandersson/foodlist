import { beforeEach, describe, expect, it, vi } from 'vitest'

const bridge = vi.hoisted(() => ({
  app: null as null | {
    ontoolresult: (result: unknown) => void
    callServerTool: ReturnType<typeof vi.fn>
  },
}))

vi.mock('@modelcontextprotocol/ext-apps', () => ({
  App: class {
    ontoolresult = (_result: unknown) => {}
    callServerTool = vi.fn(async () => ({
      structuredContent: { id: 'milk', completedAt: '2026-09-27T00:00:00Z', starred: true, sortOrder: 2 },
    }))
    connect = vi.fn(async () => {})
    constructor() { bridge.app = this }
  },
}))

describe('MCP shopping view', () => {
  beforeEach(async () => {
    vi.resetModules()
    document.body.innerHTML = '<h1 id="title"></h1><p id="summary"></p><p id="message"></p><div id="groups"></div>'
    await import('../mcp-app/shopping')
  })

  it('renders hostile names as text and reconciles controls from the returned item', async () => {
    const app = bridge.app!
    app.ontoolresult({ structuredContent: {
      title: 'Groceries <script>alert(1)</script>', includeCompleted: true,
      categories: [{ id: 'dairy', name: 'Dairy <img src=x>', sortOrder: 1 }],
      items: [{ id: 'milk', name: 'Milk <script>alert(1)</script>', categoryId: 'dairy', count: 2.5, unit: 'litres', completed: false, starred: false, sortOrder: 2 }],
    } })
    expect(document.querySelector('h1')?.textContent).toContain('<script>')
    expect(document.querySelector('h2')?.textContent).toContain('<img')
    expect(document.querySelector('.name')?.textContent).toContain('<script>')
    expect(document.querySelector('script, img')).toBeNull()
    expect(document.querySelector('.quantity')?.textContent).toBe('2.5 litres')

    ;(document.querySelector('button[aria-label^="Complete"]') as HTMLButtonElement).click()
    ;(document.querySelector('button[aria-label^="Star"]') as HTMLButtonElement).click()
    await vi.waitFor(() => expect(document.querySelector('button[aria-label^="Reopen"]')).not.toBeNull())
    expect(app.callServerTool).toHaveBeenCalledTimes(1)
    expect(app.callServerTool).toHaveBeenCalledWith({ name: 'foodlist_update_item', arguments: { todo_id: 'milk', done: true } })
    expect(document.querySelector('button[aria-label^="Unstar"]')).not.toBeNull()
  })

  it('removes a completed item when include_completed is false', async () => {
    const app = bridge.app!
    app.ontoolresult({ structuredContent: {
      title: 'List', includeCompleted: false, categories: [],
      items: [{ id: 'milk', name: 'Milk', categoryId: null, count: null, unit: null, completed: false, starred: false, sortOrder: 1 }],
    } })
    ;(document.querySelector('button[aria-label^="Complete"]') as HTMLButtonElement).click()
    await vi.waitFor(() => expect(document.querySelector('.empty')).not.toBeNull())
  })
})
