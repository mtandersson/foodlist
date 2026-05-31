<script lang="ts">
  import { onMount } from 'svelte';
  import { fade } from 'svelte/transition';
  import { getRecipe, updateRecipe } from './recipes';
  import type { Recipe, RecipeSection, Ingredient } from './types';
  import type { TodoStore } from './store';
  import { recipeDetailModeStore } from './recipesState';
  import RecipeImageLightbox from './RecipeImageLightbox.svelte';
  import RecipeSectionEditor from './RecipeSectionEditor.svelte';
  import { renderMarkdown } from './markdown';

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

  let lightboxOpen = $state(false);
  let editing = $state(false);
  let editTitle = $state('');
  let editDescription = $state('');
  let editSections: RecipeSection[] = $state([]);
  let saveError: string | null = $state(null);

  // Subscribe to the cook session for this recipe so checkboxes reflect
  // other clients' state in real time. Indices are FLAT across sections
  // matching the backend's recipeTotalSteps contract. The subscriptions
  // live inside $effect so Svelte 5 re-reads `store` and `recipeId`
  // through their reactive proxies rather than capturing the initial
  // values once at module evaluation.
  let checkedSet = $state<Set<number>>(new Set());
  $effect(() => {
    const unsub = store.cookSessions.subscribe((m) => {
      checkedSet = new Set(m.get(recipeId) ?? []);
    });
    return unsub;
  });

  let lastVersion = -1;
  $effect(() => {
    const unsub = store.recipesVersion.subscribe((v) => {
      if (v !== lastVersion) {
        lastVersion = v;
        if (lastVersion >= 0) refresh();
      }
    });
    return unsub;
  });

  // `stepOffsets[i]` is the global zero-based index of the first
  // step in section i, which keeps the cook checkboxes and the
  // visual step numbering in sync with the backend's flat
  // recipeTotalSteps model. Ingredient flattening is not needed in
  // the UI because the +add button always operates on the row's
  // direct reference - the MCP server handles 1-based global
  // ingredient indexing on its own.
  let sections = $derived<RecipeSection[]>(recipe?.sections ?? []);
  let isSingleUnnamed = $derived(sections.length === 1 && !sections[0].name);
  let stepOffsets = $derived(
    sections.reduce<number[]>((acc, _) => {
      acc.push(acc.length === 0 ? 0 : acc[acc.length - 1] + sections[acc.length - 1].instructions.length);
      return acc;
    }, [])
  );

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

  function startEdit() {
    if (!recipe) return;
    editTitle = recipe.title;
    editDescription = recipe.description ?? '';
    editSections = (recipe.sections ?? []).map((s) => ({
      name: s.name ?? '',
      ingredients: (s.ingredients ?? []).map((i) => ({
        amount: i.amount ?? null,
        unit: i.unit ?? '',
        name: i.name ?? '',
      })),
      instructions: [...(s.instructions ?? [])],
    }));
    saveError = null;
    editing = true;
  }

  async function saveEdit() {
    if (!recipe) return;
    saveError = null;
    const trimmedSections = editSections
      .map((s) => ({
        name: s.name?.trim() ?? '',
        ingredients: s.ingredients
          .filter((i) => i.name?.trim())
          .map((i) => ({
            amount: i.amount,
            unit: i.unit?.trim() || '',
            name: i.name.trim(),
          })),
        instructions: s.instructions
          .filter((line) => line.trim())
          .map((line) => line.trim()),
      }))
      .filter((s) => s.ingredients.length > 0 || s.instructions.length > 0);
    // Client-side guards mirror the backend validator so the user gets
    // a Swedish, inline error instead of a generic "recipe invalid"
    // string bubbled up from writeAPIError.
    if (!editTitle.trim()) {
      saveError = 'Titel krävs.';
      return;
    }
    if (trimmedSections.length === 0) {
      saveError = 'Receptet behöver minst en sektion med ingredienser eller steg.';
      return;
    }
    try {
      const resp = await updateRecipe(recipe.id, {
        title: editTitle.trim(),
        description: editDescription.trim(),
        sections: trimmedSections,
      });
      recipe = resp.recipe;
      imageUrl = resp.imageUrl;
      editing = false;
    } catch (e) {
      saveError = (e as Error).message || 'Kunde inte spara ändringarna.';
    }
  }

  function addIngredientToList(ing: Ingredient) {
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

  function toggleStep(globalIdx: number) {
    if (checkedSet.has(globalIdx)) {
      store.cookUncheck(recipeId, globalIdx);
    } else {
      store.cookCheck(recipeId, globalIdx);
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

      {#if recipe.description && recipe.description.trim()}
        <!--
          {@html} is safe here because renderMarkdown() runs the source
          through marked + DOMPurify with a narrow allowlist and a
          URL-validating href hook. The SecurityHeadersMiddleware CSP
          additionally blocks inline script execution as defense in
          depth. See frontend/src/lib/markdown.ts for the full
          allowlist and href sanitization rules.
        -->
        <section class="description recipe-description">
          {@html renderMarkdown(recipe.description)}
        </section>
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

      {#if isSingleUnnamed}
        <!-- Single unnamed section renders with the old h2 labels so
             a recipe with no logical grouping looks identical to the
             pre-sections layout the user is used to. -->
        <section class="ingredients">
          <h2>Ingredienser</h2>
          {#if sections[0].ingredients.length === 0}
            <p class="empty">Inga ingredienser registrerade.</p>
          {:else}
            <ul>
              {#each sections[0].ingredients as ing, i}
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
                      onclick={() => addIngredientToList(ing)}
                    >+</button>
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
              <button type="button" class="secondary" onclick={resetCook}>Återställ</button>
            {/if}
          </div>
          {#if sections[0].instructions.length === 0}
            <p class="empty">Inga instruktioner registrerade.</p>
          {:else}
            <ol>
              {#each sections[0].instructions as step, i}
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
        <!-- Multi-section or named-single layout: one card per
             section, with h3 headings (h2 is reserved for the
             "Redigera" mode label only). Ingredient + step indices
             stay GLOBAL across sections because the cook session and
             the backend bounds check use a flat int. -->
        {#if $recipeDetailModeStore === 'cook'}
          <div class="cook-toolbar">
            <button type="button" class="secondary" onclick={resetCook}>Återställ alla</button>
          </div>
        {/if}
        {#each sections as section, sIdx}
          <section class="section-card">
            <h3>{section.name || `Sektion ${sIdx + 1}`}</h3>
            {#if section.ingredients.length > 0}
              <div class="subblock">
                <h4>Ingredienser</h4>
                <ul>
                  {#each section.ingredients as ing}
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
                          onclick={() => addIngredientToList(ing)}
                        >+</button>
                      {/if}
                    </li>
                  {/each}
                </ul>
              </div>
            {/if}
            {#if section.instructions.length > 0}
              <div class="subblock">
                <h4>Steg</h4>
                <ol class="manual-numbers">
                  {#each section.instructions as step, iIdx}
                    {@const globalIdx = stepOffsets[sIdx] + iIdx}
                    <li
                      class="step-row"
                      class:checked={$recipeDetailModeStore === 'cook' && checkedSet.has(globalIdx)}
                    >
                      <span class="step-num">{globalIdx + 1}.</span>
                      {#if $recipeDetailModeStore === 'cook'}
                        <label class="step-check">
                          <input
                            type="checkbox"
                            checked={checkedSet.has(globalIdx)}
                            onchange={() => toggleStep(globalIdx)}
                            aria-label={`Steg ${globalIdx + 1}`}
                          />
                          <span>{step}</span>
                        </label>
                      {:else}
                        <span>{step}</span>
                      {/if}
                    </li>
                  {/each}
                </ol>
              </div>
            {/if}
          </section>
        {/each}
      {/if}
    {:else}
      <h2>Redigera recept</h2>
      <label>
        Titel
        <input type="text" bind:value={editTitle} maxlength={200} />
      </label>
      <label>
        Beskrivning
        <textarea
          bind:value={editDescription}
          rows="4"
          maxlength={4000}
          placeholder="Intro, portioner, källa… Markdown: **fet**, *kursiv*, listor, > citat, [länkar](https://…)."
        ></textarea>
      </label>
      {#if editDescription.trim()}
        <div class="md-preview" aria-label="Förhandsvisning av beskrivning">
          <span class="md-preview-label">Förhandsvisning</span>
          <div class="recipe-description">{@html renderMarkdown(editDescription)}</div>
        </div>
      {/if}
      <RecipeSectionEditor bind:sections={editSections} />
      {#if saveError}
        <p class="error" role="alert">{saveError}</p>
      {/if}
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

  /* Scoped description styling shared with RecipeUploadModal.
     Headings stay subdued so they cannot compete with the page
     title (h1) or section labels (h2/h3). */
  .description {
    /* `section` rule above already gives background + padding. */
  }
  .recipe-description :global(p) {
    margin: 0 0 var(--spacing-xs);
  }
  .recipe-description :global(p:last-child) {
    margin-bottom: 0;
  }
  .recipe-description :global(h3),
  .recipe-description :global(h4),
  .recipe-description :global(h5),
  .recipe-description :global(h6) {
    margin: var(--spacing-xs) 0;
    font-size: var(--font-size-base);
    color: var(--text-secondary, var(--text-muted));
  }
  .recipe-description :global(blockquote) {
    margin: 0;
    padding-left: var(--spacing-sm);
    border-left: 3px solid var(--border-color, rgba(0, 0, 0, 0.15));
    color: var(--text-secondary, var(--text-muted));
  }
  .recipe-description :global(ul),
  .recipe-description :global(ol) {
    margin: 0;
    padding-left: var(--spacing-md);
  }
  .recipe-description :global(a) {
    color: var(--primary-color);
  }
  .recipe-description :global(a)::after {
    content: ' \2197';
    font-size: 0.85em;
  }

  /* Multi-section card layout. h3 is the section heading; h4 labels
     the ingredients/steps subblocks. Manual numbering is used in
     <ol class="manual-numbers"> so the per-section <ol> doesn't
     reset numbering when we want the GLOBAL step index visible to
     match the cook session model. */
  .section-card {
    /* Inherits the .section base style. */
  }
  .section-card h3 {
    margin: 0 0 var(--spacing-sm);
    color: var(--text-primary);
  }
  .section-card h4 {
    margin: var(--spacing-sm) 0 var(--spacing-xs);
    font-size: var(--font-size-base);
    color: var(--text-secondary, var(--text-muted));
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }
  .subblock {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-xs);
  }
  .manual-numbers {
    list-style: none;
    padding-left: 0;
  }
  .manual-numbers .step-row {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: var(--spacing-xs);
    align-items: start;
  }
  .cook-toolbar {
    display: flex;
    justify-content: flex-end;
  }

  /* The preview sits INSIDE the .modal/.card surface, so it must NOT
     paint a fill. --surface-muted is white-glass tuned for the
     colored page background and renders invisible on the white light
     card and bright-on-dark on the dark card. The dashed border plus
     the "Förhandsvisning" label is enough to separate the preview
     visually without re-introducing a theme bug. */
  .md-preview {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-xs);
    padding: var(--spacing-sm);
    border: 1px dashed var(--border-color);
    border-radius: var(--radius-md);
    background: transparent;
    color: var(--text-primary);
  }
  .md-preview-label {
    font-size: var(--font-size-sm);
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.04em;
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

  .error {
    color: var(--error-color, #c0392b);
    margin: 0;
  }

  .empty {
    color: var(--text-secondary, #777);
    font-style: italic;
  }
</style>
