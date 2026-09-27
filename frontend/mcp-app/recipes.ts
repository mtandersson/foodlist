import { App } from '@modelcontextprotocol/ext-apps'

type RecipeMeta = {
  id: string
  title: string
  imageFilename: string
  createdAt: string
  updatedAt: string
}
type RecipeList = { enabled: boolean; recipes: RecipeMeta[] }

const app = new App({ name: 'Foodlist recipe cards', version: '1.0.0' })
const cards = document.getElementById('cards')!
const message = document.getElementById('message')!
const detail = document.getElementById('detail')!
const recipeText = document.getElementById('recipe-text')!
let openSequence = 0
document.getElementById('close')!.addEventListener('click', () => { openSequence++; detail.hidden = true })
let observer: IntersectionObserver | null = null

function dateLabel(value: string): string {
  const date = new Date(value)
  return Number.isNaN(date.valueOf()) ? value : date.toLocaleDateString()
}

function resultText(result: { content?: Array<{ type: string; text?: string }> }): string {
  return result.content?.find(content => content.type === 'text')?.text ?? ''
}

async function loadThumbnail(recipe: RecipeMeta, image: HTMLImageElement): Promise<void> {
  const fallback = image.previousElementSibling
  try {
    const result = await app.readServerResource({ uri: `foodlist://recipe-thumbnail/${recipe.id}` })
    const first = result.contents?.[0]
    if (first && 'blob' in first && typeof first.blob === 'string' && first.mimeType === 'image/jpeg') {
      image.src = `data:image/jpeg;base64,${first.blob}`
      image.hidden = false
      fallback?.remove()
      return
    }
  } catch {
    // The recipe or image may have been removed since listing.
  }
  if (fallback) fallback.textContent = 'No image'
}

async function openRecipe(recipe: RecipeMeta, button: HTMLButtonElement): Promise<void> {
  const sequence = ++openSequence
  button.disabled = true
  message.textContent = ''
  try {
    const result = await app.callServerTool({ name: 'foodlist_recipe_get', arguments: { recipe_id: recipe.id } })
    if (sequence !== openSequence) return
    if (result.isError) throw new Error('Could not open this recipe.')
    recipeText.textContent = resultText(result)
    detail.hidden = false
    detail.scrollIntoView({ block: 'nearest' })
  } catch {
    if (sequence === openSequence) message.textContent = 'Could not open this recipe.'
  } finally {
    button.disabled = false
  }
}

async function addIngredients(recipe: RecipeMeta, button: HTMLButtonElement): Promise<void> {
  button.disabled = true
  message.textContent = ''
  try {
    const result = await app.callServerTool({ name: 'foodlist_recipe_add_ingredients', arguments: { recipe_id: recipe.id } })
    if (result.isError) throw new Error('Could not add ingredients.')
    message.textContent = resultText(result) || 'Ingredients added.'
  } catch {
    message.textContent = 'Could not add ingredients.'
  } finally {
    button.disabled = false
  }
}

function render(data: RecipeList): void {
  openSequence++
  observer?.disconnect()
  cards.replaceChildren()
  detail.hidden = true
  if (!data.enabled) {
    message.textContent = 'Recipes feature is disabled.'
    return
  }
  message.textContent = ''
  if (data.recipes.length === 0) {
    const empty = document.createElement('p')
    empty.className = 'empty'
    empty.textContent = 'No saved recipes yet.'
    cards.append(empty)
    return
  }
  observer = new IntersectionObserver(entries => {
    for (const entry of entries) {
      if (!entry.isIntersecting) continue
      observer?.unobserve(entry.target)
      const recipe = data.recipes.find(r => r.id === (entry.target as HTMLElement).dataset.recipeId)
      const image = entry.target.querySelector('img')
      if (recipe && image) void loadThumbnail(recipe, image)
    }
  })
  for (const recipe of data.recipes) {
    const card = document.createElement('article')
    card.dataset.recipeId = recipe.id
    const imageBox = document.createElement('div')
    imageBox.className = 'image'
    const fallback = document.createElement('span')
    fallback.textContent = recipe.imageFilename ? 'Image loading…' : 'No image'
    imageBox.append(fallback)
    if (recipe.imageFilename) {
      const image = document.createElement('img')
      image.alt = ''
      image.hidden = true
      imageBox.append(image)
      observer.observe(card)
    }
    const body = document.createElement('div')
    body.className = 'body'
    const heading = document.createElement('h2')
    heading.textContent = recipe.title
    const date = document.createElement('p')
    date.className = 'date'
    date.textContent = `Saved ${dateLabel(recipe.createdAt)} · Updated ${dateLabel(recipe.updatedAt)}`
    const actions = document.createElement('div')
    actions.className = 'actions'
    const open = document.createElement('button')
    open.type = 'button'
    open.textContent = 'Open recipe'
    open.setAttribute('aria-label', `Open ${recipe.title}`)
    open.addEventListener('click', () => openRecipe(recipe, open))
    const add = document.createElement('button')
    add.type = 'button'
    add.textContent = 'Add ingredients'
    add.setAttribute('aria-label', `Add all ingredients from ${recipe.title}`)
    add.addEventListener('click', () => addIngredients(recipe, add))
    actions.append(open, add)
    body.append(heading, date, actions)
    card.append(imageBox, body)
    cards.append(card)
  }
}

app.ontoolresult = result => {
  if (result.isError) { message.textContent = 'Could not load recipes.'; return }
  const data = result.structuredContent as RecipeList | undefined
  if (!data || !Array.isArray(data.recipes) || typeof data.enabled !== 'boolean') {
    message.textContent = 'Invalid recipe list response.'
    return
  }
  render(data)
}

app.connect().catch(() => { message.textContent = 'Could not connect to the recipe host.' })
