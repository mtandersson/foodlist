<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { fade } from 'svelte/transition';
  import { getRecipe, updateRecipe } from './recipes';
  import type { Recipe } from './types';
  import type { TodoStore } from './store';
  import { recipeDetailModeStore } from './recipesState';
  import RecipeImageLightbox from './RecipeImageLightbox.svelte';

  interface Props {
    recipeId: string;
    store: TodoStore;
    onBack: () => void;
    onDelete: () => void;
  }

  let { recipeId, store, onBack, onDelete }: Props = $props();

  let recipe: Recipe | null = $state(null);
  let imageUrl: string | null = $state(null);
  let loading = $state(true);
  let error: string | null = $state(null);

  // Normal/Cook is shared, persisted state - bound to the
  // module-scoped recipeDetailModeStore so the toggle survives a
  // tab switch with no special handling at the call site. The
  // store's subscriber writes to localStorage synchronously when
  // .set() is called, so even a "click Cook -> click Inköp" gesture
  // can't unmount this component before the save is durable.
  // See ./recipesState.ts for the rationale.

  // Fullscreen, pinch-zoomable image viewer. Opens on a tap of the
  // hero image. The user lands back on the same scroll position when
  // closing because we never unmount the detail view itself.
  let lightboxOpen = $state(false);
  let editing = $state(false);
  let editTitle = $state('');
  let editIngredients = $state<{amount: number | null; unit: string; name: string}[]>([]);
  let editInstructions = $state<string[]>([]);

  // Subscribe to the cook session for this recipe so checkboxes reflect
  // other clients' state in real time.
  let checkedSet = $state<Set<number>>(new Set());
  const unsubCook = store.cookSessions.subscribe((m) => {
    checkedSet = new Set(m.get(recipeId) ?? []);
  });

  // Re-fetch when something changes server-side (PATCH from another tab,
  // step pruning after edit, etc.).
  let lastVersion = -1;
  const unsubVersion = store.recipesVersion.subscribe((v) => {
    if (v !== lastVersion) {
      lastVersion = v;
      if (lastVersion >= 0) refresh();
    }
  });

  async function refresh() {
    loading = true;
    error = null;
    try {
      const resp = await getRecipe(recipeId);
      recipe = resp.recipe;
      imageUrl = resp.imageUrl;
    } catch (e) {
      error = (e as Error).message || 'Något gick fel';
    } finally {
      loading = false;
    }
  }

  onMount(refresh);
  onDestroy(() => {
    unsubCook();
    unsubVersion();
  });

  function startEdit() {
    if (!recipe) return;
    editTitle = recipe.title;
    editIngredients = recipe.ingredients.map((i) => ({
      amount: i.amount ?? null,
      unit: i.unit ?? '',
      name: i.name ?? '',
    }));
    editInstructions = [...recipe.instructions];
    editing = true;
  }

  async function saveEdit() {
    if (!recipe) return;
    try {
      const resp = await updateRecipe(recipe.id, {
        title: editTitle.trim(),
        ingredients: editIngredients
          .filter((i) => i.name.trim())
          .map((i) => ({
            amount: i.amount,
            unit: i.unit?.trim() || '',
            name: i.name.trim(),
          })),
        instructions: editInstructions.filter((s) => s.trim()).map((s) => s.trim()),
      });
      recipe = resp.recipe;
      imageUrl = resp.imageUrl;
      editing = false;
    } catch (e) {
      alert('Kunde inte spara ändringarna');
    }
  }

  function addEditIngredient() {
    editIngredients = [...editIngredients, { amount: null, unit: '', name: '' }];
  }
  function removeEditIngredient(idx: number) {
    editIngredients = editIngredients.filter((_, i) => i !== idx);
  }
  function addEditInstruction() {
    editInstructions = [...editInstructions, ''];
  }
  function removeEditInstruction(idx: number) {
    editInstructions = editInstructions.filter((_, i) => i !== idx);
  }

  function addIngredientToList(idx: number) {
    if (!recipe) return;
    const ing = recipe.ingredients[idx];
    if (!ing || !ing.name.trim()) return;
    const original = formatIngredient(ing);
    store.createTodo(ing.name.trim(), null, {
      count: ing.amount ?? null,
      unit: ing.unit ?? '',
      originalInput: original,
    });
  }

  function formatIngredient(ing: {amount?: number | null; unit?: string; name: string}): string {
    const parts: string[] = [];
    if (ing.amount != null) parts.push(String(ing.amount));
    if (ing.unit) parts.push(ing.unit);
    parts.push(ing.name);
    return parts.join(' ');
  }

  function toggleStep(idx: number) {
    if (checkedSet.has(idx)) {
      store.cookUncheck(recipeId, idx);
    } else {
      store.cookCheck(recipeId, idx);
    }
  }

  function resetCook() {
    if (!confirm('Återställ alla steg för alla?')) return;
    store.cookReset(recipeId);
  }

  function handleDelete() {
    onDelete();
  }
</script>

<div class="recipe-detail" transition:fade={{ duration: 150 }}>
  <header class="detail-header">
    <button type="button" class="back" onclick={onBack} aria-label="Tillbaka till recept">
      ← Recept
    </button>
    {#if recipe && !editing}
      <div class="header-actions">
        <button type="button" class="secondary" onclick={startEdit}>Redigera</button>
        <button type="button" class="danger" onclick={handleDelete}>Ta bort</button>
      </div>
    {/if}
  </header>

  {#if loading}
    <p class="status">Laddar…</p>
  {:else if error}
    <p class="status error">{error}</p>
  {:else if recipe}
    {#if !editing}
      <h1>{recipe.title}</h1>
      {#if imageUrl}
        <button
          type="button"
          class="hero-button"
          onclick={() => (lightboxOpen = true)}
          aria-label="Visa bilden i fullskärm"
        >
          <img src={imageUrl} alt="" class="hero" loading="lazy" />
        </button>
      {/if}

      <div class="mode-toggle" role="group" aria-label="Visa-läge">
        <button
          type="button"
          class:selected={$recipeDetailModeStore === 'normal'}
          aria-pressed={$recipeDetailModeStore === 'normal'}
          onclick={() => recipeDetailModeStore.set('normal')}
        >Normal</button>
        <button
          type="button"
          class:selected={$recipeDetailModeStore === 'cook'}
          aria-pressed={$recipeDetailModeStore === 'cook'}
          onclick={() => recipeDetailModeStore.set('cook')}
        >Kock-läge</button>
      </div>

      <section class="ingredients">
        <h2>Ingredienser</h2>
        {#if recipe.ingredients.length === 0}
          <p class="empty">Inga ingredienser registrerade.</p>
        {:else}
          <ul>
            {#each recipe.ingredients as ing, i}
              <li class="ing-row">
                <span class="ing-text">
                  {#if ing.amount != null}<strong>{ing.amount}</strong>{' '}{/if}
                  {#if ing.unit}<span class="unit">{ing.unit}</span>{' '}{/if}
                  <span class="name">{ing.name}</span>
                </span>
                {#if $recipeDetailModeStore === 'normal'}
                  <button
                    type="button"
                    class="add-btn"
                    aria-label={`Lägg till ${ing.name} i listan`}
                    onclick={() => addIngredientToList(i)}
                  >
                    +
                  </button>
                {/if}
              </li>
            {/each}
          </ul>
        {/if}
      </section>

      <section class="instructions">
        <div class="section-header">
          <h2>Instruktioner</h2>
          {#if $recipeDetailModeStore === 'cook'}
            <button type="button" class="secondary" onclick={resetCook}>
              Återställ
            </button>
          {/if}
        </div>
        {#if recipe.instructions.length === 0}
          <p class="empty">Inga instruktioner registrerade.</p>
        {:else}
          <ol>
            {#each recipe.instructions as step, i}
              <li class="step-row" class:checked={$recipeDetailModeStore === 'cook' && checkedSet.has(i)}>
                {#if $recipeDetailModeStore === 'cook'}
                  <label class="step-check">
                    <input
                      type="checkbox"
                      checked={checkedSet.has(i)}
                      onchange={() => toggleStep(i)}
                      aria-label={`Steg ${i + 1}`}
                    />
                    <span>{step}</span>
                  </label>
                {:else}
                  <span>{step}</span>
                {/if}
              </li>
            {/each}
          </ol>
        {/if}
      </section>
    {:else}
      <h2>Redigera recept</h2>
      <label>
        Titel
        <input type="text" bind:value={editTitle} maxlength={200} />
      </label>
      <fieldset>
        <legend>Ingredienser</legend>
        {#each editIngredients as ing, i}
          <div class="ing-edit">
            <input type="number" step="any" placeholder="Mängd" bind:value={ing.amount} />
            <input type="text" placeholder="Enhet" bind:value={ing.unit} maxlength={32} />
            <input type="text" placeholder="Ingrediens" bind:value={ing.name} maxlength={2000} />
            <button type="button" class="icon-btn" onclick={() => removeEditIngredient(i)} aria-label="Ta bort">×</button>
          </div>
        {/each}
        <button type="button" class="secondary" onclick={addEditIngredient}>+ Ingrediens</button>
      </fieldset>
      <fieldset>
        <legend>Steg</legend>
        {#each editInstructions as _, i}
          <div class="step-edit">
            <span class="step-num">{i + 1}.</span>
            <textarea bind:value={editInstructions[i]} rows="2" maxlength={2000}></textarea>
            <button type="button" class="icon-btn" onclick={() => removeEditInstruction(i)} aria-label="Ta bort">×</button>
          </div>
        {/each}
        <button type="button" class="secondary" onclick={addEditInstruction}>+ Steg</button>
      </fieldset>
      <div class="actions">
        <button type="button" class="secondary" onclick={() => (editing = false)}>Avbryt</button>
        <button type="button" class="primary" onclick={saveEdit}>Spara</button>
      </div>
    {/if}
  {/if}
</div>

{#if lightboxOpen && imageUrl}
  <RecipeImageLightbox
    src={imageUrl}
    alt={recipe?.title ?? ''}
    onClose={() => (lightboxOpen = false)}
  />
{/if}

<style>
  .recipe-detail {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-md);
  }

  .detail-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .back {
    background: transparent;
    border: none;
    color: var(--text-on-primary);
    font-size: var(--font-size-base);
    cursor: pointer;
    font-weight: var(--font-weight-semibold);
  }

  .header-actions {
    display: flex;
    gap: var(--spacing-xs);
  }

  h1 {
    margin: 0;
    color: var(--text-primary);
    background: var(--card-bg);
    padding: var(--spacing-md);
    border-radius: var(--radius-md);
  }

  h2 {
    margin: 0 0 var(--spacing-sm);
    color: var(--text-primary);
  }

  .hero {
    width: 100%;
    max-height: 320px;
    object-fit: cover;
    border-radius: var(--radius-md);
  }

  /* Wraps the hero <img> in a real <button> so keyboard users can
     reach the lightbox via Tab. The button itself has no chrome -
     the image fills it - and we expose the affordance via cursor +
     focus ring instead. */
  .hero-button {
    border: none;
    padding: 0;
    background: transparent;
    width: 100%;
    cursor: zoom-in;
    border-radius: var(--radius-md);
    display: block;
  }

  .hero-button:focus-visible {
    outline: 2px solid var(--primary-color);
    outline-offset: 2px;
  }

  .hero-button > .hero {
    display: block;
  }

  .mode-toggle {
    display: inline-flex;
    background: var(--surface-muted);
    border-radius: var(--radius-full);
    padding: var(--spacing-xs);
    gap: var(--spacing-xs);
    align-self: flex-start;
  }

  .mode-toggle button {
    border: none;
    background: transparent;
    padding: var(--spacing-sm) var(--spacing-md);
    border-radius: var(--radius-full);
    color: var(--primary-color);
    cursor: pointer;
    font-weight: var(--font-weight-semibold);
  }

  .mode-toggle button.selected {
    background: var(--card-bg);
    color: var(--text-primary);
    box-shadow: var(--shadow-sm);
  }

  section {
    background: var(--card-bg);
    color: var(--text-primary);
    padding: var(--spacing-md);
    border-radius: var(--radius-md);
  }

  .section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .ingredients ul,
  .instructions ol {
    margin: 0;
    padding-left: var(--spacing-lg);
    display: flex;
    flex-direction: column;
    gap: var(--spacing-xs);
  }

  .ing-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: var(--spacing-sm);
  }

  .ing-text .unit {
    color: var(--text-secondary, #555);
  }

  .add-btn {
    background: var(--primary-color, #4a90e2);
    color: white;
    border: none;
    width: 32px;
    height: 32px;
    border-radius: 50%;
    cursor: pointer;
    font-size: 18px;
    line-height: 1;
  }

  .step-row.checked {
    text-decoration: line-through;
    opacity: 0.6;
  }

  .step-check {
    display: flex;
    gap: var(--spacing-sm);
    align-items: flex-start;
    cursor: pointer;
  }

  .step-check input {
    margin-top: 4px;
  }

  fieldset {
    border: 1px solid var(--border-color, rgba(0, 0, 0, 0.1));
    border-radius: var(--radius-md);
    padding: var(--spacing-md);
    display: flex;
    flex-direction: column;
    gap: var(--spacing-sm);
    background: var(--card-bg);
    color: var(--text-primary);
  }

  .ing-edit {
    display: grid;
    grid-template-columns: 70px 90px 1fr 32px;
    gap: var(--spacing-xs);
    align-items: center;
  }

  .step-edit {
    display: grid;
    grid-template-columns: 28px 1fr 32px;
    gap: var(--spacing-xs);
    align-items: start;
  }

  .step-num {
    font-weight: var(--font-weight-semibold);
    padding-top: var(--spacing-sm);
  }

  input,
  textarea {
    padding: var(--spacing-xs) var(--spacing-sm);
    border: 1px solid var(--border-color, rgba(0, 0, 0, 0.15));
    border-radius: var(--radius-sm);
    font-size: var(--font-size-base);
    font-family: inherit;
    color: var(--text-primary);
    /* --card-bg is themed (white in light, #1c1c1e in dark) so we
       avoid the white-on-white invisible-text bug that the previous
       `var(--surface-bg, #fff)` fallback caused. The 1px border
       separates the input from the surrounding fieldset card. */
    background: var(--card-bg);
  }

  input::placeholder,
  textarea::placeholder {
    color: var(--text-muted);
    opacity: 1;
  }

  label {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-xs);
    font-weight: var(--font-weight-semibold);
    color: var(--text-on-primary);
  }

  .icon-btn {
    background: transparent;
    border: 1px solid var(--border-color, rgba(0, 0, 0, 0.1));
    border-radius: var(--radius-sm);
    width: 28px;
    height: 28px;
    cursor: pointer;
  }

  .actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--spacing-sm);
  }

  .primary {
    background: var(--primary-color, #4a90e2);
    color: white;
    border: none;
    padding: var(--spacing-sm) var(--spacing-md);
    border-radius: var(--radius-full);
    cursor: pointer;
    font-weight: var(--font-weight-semibold);
  }

  .secondary {
    background: transparent;
    color: var(--text-primary);
    border: 1px solid var(--border-color, rgba(0, 0, 0, 0.2));
    padding: var(--spacing-sm) var(--spacing-md);
    border-radius: var(--radius-full);
    cursor: pointer;
    font-weight: var(--font-weight-semibold);
  }

  .danger {
    background: var(--error-color, #c0392b);
    color: white;
    border: none;
    padding: var(--spacing-sm) var(--spacing-md);
    border-radius: var(--radius-full);
    cursor: pointer;
    font-weight: var(--font-weight-semibold);
  }

  .status {
    color: var(--text-on-primary);
    text-align: center;
    padding: var(--spacing-md);
  }

  .status.error {
    color: var(--error-color, #ffb3b3);
  }

  .empty {
    color: var(--text-secondary, #777);
    font-style: italic;
  }
</style>
