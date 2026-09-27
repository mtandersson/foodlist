import { App } from '@modelcontextprotocol/ext-apps'

type Item = {
  id: string
  name: string
  categoryId: string | null
  count: number | null
  unit: string | null
  completed: boolean
  starred: boolean
  sortOrder: number
}
type Category = { id: string; name: string; sortOrder: number }
type List = { title: string; includeCompleted: boolean; categories: Category[]; items: Item[] }
type UpdatedItem = { id: string; completedAt: string | null; starred: boolean; sortOrder: number }

const app = new App({ name: 'Foodlist shopping list', version: '1.0.0' })
const title = document.getElementById('title')!
const summary = document.getElementById('summary')!
const message = document.getElementById('message')!
const groups = document.getElementById('groups')!
let list: List | null = null
const pendingItems = new Set<string>()

function label(item: Item): string {
  const parts: string[] = []
  if (item.count !== null) parts.push(String(item.count))
  if (item.unit) parts.push(item.unit)
  return parts.join(' ')
}

function render(): void {
  if (!list) return
  title.textContent = list.title
  summary.textContent = `${list.items.length} item${list.items.length === 1 ? '' : 's'}`
  groups.replaceChildren()
  if (list.items.length === 0) {
    const empty = document.createElement('p')
    empty.className = 'empty'
    empty.textContent = 'No matching grocery items.'
    groups.append(empty)
    return
  }

  const ordered: Array<{ id: string | null; name: string }> = list.categories.map(c => ({ id: c.id, name: c.name }))
  ordered.push({ id: null, name: 'Uncategorized' })
  const known = new Set(list.categories.map(c => c.id))
  for (const item of list.items) {
    if (item.categoryId && !known.has(item.categoryId)) {
      ordered.push({ id: item.categoryId, name: item.categoryId })
      known.add(item.categoryId)
    }
  }
  for (const category of ordered) {
    const items = list.items.filter(item => item.categoryId === category.id)
    if (items.length === 0) continue
    const heading = document.createElement('h2')
    heading.textContent = category.name
    groups.append(heading)
    const ul = document.createElement('ul')
    for (const item of items) {
      const row = document.createElement('li')
      if (item.completed) row.className = 'done'
      const done = document.createElement('button')
      done.type = 'button'
      done.textContent = item.completed ? '↺' : '✓'
      done.setAttribute('aria-label', `${item.completed ? 'Reopen' : 'Complete'} ${item.name}`)
      done.setAttribute('aria-pressed', String(item.completed))
      done.addEventListener('click', () => update(item, 'done', done))
      const name = document.createElement('span')
      name.className = 'name'
      name.textContent = item.name
      const quantity = document.createElement('span')
      quantity.className = 'quantity'
      quantity.textContent = label(item)
      const star = document.createElement('button')
      star.type = 'button'
      star.textContent = item.starred ? '★' : '☆'
      star.setAttribute('aria-label', `${item.starred ? 'Unstar' : 'Star'} ${item.name}`)
      star.setAttribute('aria-pressed', String(item.starred))
      star.addEventListener('click', () => update(item, 'starred', star))
      row.append(done, name, quantity, star)
      ul.append(row)
    }
    groups.append(ul)
  }
}

async function update(item: Item, field: 'done' | 'starred', button: HTMLButtonElement): Promise<void> {
  if (pendingItems.has(item.id)) return
  pendingItems.add(item.id)
  const rowButtons = button.closest('li')?.querySelectorAll('button') ?? []
  rowButtons.forEach(control => { control.disabled = true })
  message.textContent = ''
  try {
    const value = field === 'done' ? !item.completed : !item.starred
    const result = await app.callServerTool({
      name: 'foodlist_update_item',
      arguments: { todo_id: item.id, [field]: value },
    })
    if (result.isError) throw new Error('Could not update this item.')
    const updated = result.structuredContent as UpdatedItem | undefined
    if (!updated || updated.id !== item.id || typeof updated.starred !== 'boolean') throw new Error('Invalid item response.')
    if (!list) return
    list.items = list.items.flatMap(current => {
      if (current.id !== updated.id) return [current]
      const next = { ...current, completed: updated.completedAt != null, starred: updated.starred, sortOrder: updated.sortOrder }
      return list!.includeCompleted || !next.completed ? [next] : []
    }).sort((a, b) => b.sortOrder - a.sortOrder)
    render()
  } catch (error) {
    message.textContent = error instanceof Error ? error.message : 'Could not update this item.'
    rowButtons.forEach(control => { control.disabled = false })
  } finally {
    pendingItems.delete(item.id)
  }
}

app.ontoolresult = result => {
  if (result.isError) {
    message.textContent = 'Could not load the shopping list.'
    return
  }
  const data = result.structuredContent as List | undefined
  if (!data || !Array.isArray(data.items) || !Array.isArray(data.categories)) {
    message.textContent = 'Invalid shopping list response.'
    return
  }
  list = data
  render()
}

app.connect().catch(() => { message.textContent = 'Could not connect to the shopping list host.' })
