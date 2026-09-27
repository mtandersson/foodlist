import { beforeEach, describe, expect, it, vi } from 'vitest'

const bridge = vi.hoisted(() => ({
  app: null as null | {
    ontoolresult: (result: unknown) => void
    callServerTool: ReturnType<typeof vi.fn>
    readServerResource: ReturnType<typeof vi.fn>
  },
  observers: [] as Array<{ trigger: (element: Element) => void }>,
}))

vi.mock('@modelcontextprotocol/ext-apps', () => ({
  App: class {
    ontoolresult = (_result: unknown) => {}
    callServerTool = vi.fn(async ({ name }: { name: string }) => ({
      content: [{ type: 'text', text: name === 'foodlist_recipe_get' ? 'Recipe <script>do not run</script>' : 'Added 2 ingredients.' }],
    }))
    readServerResource = vi.fn(async () => ({ contents: [{ uri: 'foodlist://recipe-thumbnail/photo', mimeType: 'image/jpeg', blob: 'aW1hZ2U=' }] }))
    connect = vi.fn(async () => {})
    constructor() { bridge.app = this }
  },
}))

describe('MCP recipe cards view', () => {
  beforeEach(async () => {
    vi.resetModules()
    bridge.observers.length = 0
    vi.stubGlobal('IntersectionObserver', class {
      private callback: IntersectionObserverCallback
      constructor(callback: IntersectionObserverCallback) {
        this.callback = callback
        bridge.observers.push(this)
      }
      observe = vi.fn()
      unobserve = vi.fn()
      disconnect = vi.fn()
      trigger(element: Element) { this.callback([{ target: element, isIntersecting: true } as IntersectionObserverEntry], this as unknown as IntersectionObserver) }
    })
    HTMLElement.prototype.scrollIntoView = vi.fn()
    document.body.innerHTML = '<div id="cards"></div><p id="message"></p><section id="detail" hidden><button id="close"></button><pre id="recipe-text"></pre></section>'
    await import('../mcp-app/recipes')
  })

  it('loads images only when visible and keeps untrusted text inert', async () => {
    const app = bridge.app!
    app.ontoolresult({ structuredContent: { enabled: true, recipes: [
      { id: 'photo', title: 'Cake <img src=x>', imageFilename: 'photo.png', createdAt: '2026-09-27T00:00:00Z', updatedAt: '2026-09-27T00:00:00Z' },
      { id: 'plain', title: 'Soup', imageFilename: '', createdAt: '2026-09-27T00:00:00Z', updatedAt: '2026-09-27T00:00:00Z' },
    ] } })
    expect(document.querySelectorAll('article')).toHaveLength(2)
    expect(document.querySelector('h2')?.textContent).toBe('Cake <img src=x>')
    expect(document.querySelectorAll('img')).toHaveLength(1)
    expect(document.querySelector('img')?.getAttribute('src')).toBeNull()
    expect(app.readServerResource).not.toHaveBeenCalled()
    expect(document.querySelectorAll('article')[1].textContent).toContain('No image')

    bridge.observers[0].trigger(document.querySelector('article')!)
    await vi.waitFor(() => expect(document.querySelector('img')?.getAttribute('src')).toBe('data:image/jpeg;base64,aW1hZ2U='))
    expect(app.readServerResource).toHaveBeenCalledWith({ uri: 'foodlist://recipe-thumbnail/photo' })

    ;(document.querySelector('button[aria-label^="Open"]') as HTMLButtonElement).click()
    await vi.waitFor(() => expect(document.querySelector('#recipe-text')?.textContent).toContain('<script>'))
    expect(document.querySelector('script')).toBeNull()
    expect(document.querySelector('#detail')?.hasAttribute('hidden')).toBe(false)
    ;(document.querySelector('button[aria-label^="Add all"]') as HTMLButtonElement).click()
    await vi.waitFor(() => expect(document.querySelector('#message')?.textContent).toContain('Added 2 ingredients'))
    expect(app.callServerTool).toHaveBeenCalledWith({ name: 'foodlist_recipe_add_ingredients', arguments: { recipe_id: 'photo' } })
  })

  it('shows a disabled state without fetching images', () => {
    bridge.app!.ontoolresult({ structuredContent: { enabled: false, recipes: [] } })
    expect(document.querySelector('#message')?.textContent).toContain('disabled')
    expect(document.querySelectorAll('article')).toHaveLength(0)
    expect(bridge.app!.readServerResource).not.toHaveBeenCalled()
  })

  it('keeps the most recently opened recipe when responses arrive out of order', async () => {
    const app = bridge.app!
    app.ontoolresult({ structuredContent: { enabled: true, recipes: [
      { id: 'first', title: 'First', imageFilename: '', createdAt: '2026-09-27T00:00:00Z', updatedAt: '2026-09-27T00:00:00Z' },
      { id: 'second', title: 'Second', imageFilename: '', createdAt: '2026-09-27T00:00:00Z', updatedAt: '2026-09-27T00:00:00Z' },
    ] } })
    let finishFirst!: (value: unknown) => void
    let finishSecond!: (value: unknown) => void
    app.callServerTool
      .mockImplementationOnce(() => new Promise(resolve => { finishFirst = resolve }))
      .mockImplementationOnce(() => new Promise(resolve => { finishSecond = resolve }))
    ;(document.querySelectorAll('button[aria-label^="Open"]')[0] as HTMLButtonElement).click()
    ;(document.querySelectorAll('button[aria-label^="Open"]')[1] as HTMLButtonElement).click()
    finishSecond({ content: [{ type: 'text', text: 'Second recipe' }] })
    await vi.waitFor(() => expect(document.querySelector('#recipe-text')?.textContent).toBe('Second recipe'))
    finishFirst({ content: [{ type: 'text', text: 'First recipe' }] })
    await Promise.resolve()
    expect(document.querySelector('#recipe-text')?.textContent).toBe('Second recipe')
  })
})
