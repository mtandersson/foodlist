<script lang="ts">
  import { onDestroy } from 'svelte';
  import { fade } from 'svelte/transition';
  import { normalizeImage, parseRecipeImage, saveRecipe } from './recipes';
  import type { Recipe, Ingredient } from './types';

  interface Props {
    onClose: () => void;
    onSaved: (id: string) => void;
  }

  let { onClose, onSaved }: Props = $props();

  type Step =
    | { kind: 'pick' }
    | { kind: 'parsing'; image: Blob }
    | { kind: 'review'; image: Blob }
    | { kind: 'saving'; image: Blob };

  let step: Step = $state({ kind: 'pick' });
  let parseError: string | null = $state(null);
  let saveError: string | null = $state(null);
  // Hold the normalized image once and reuse it for both parse and save
  // so we never re-pick or re-upload on retry.
  let imageBlob: Blob | null = $state(null);
  let imageUrl: string | null = $state(null);

  let title = $state('');
  let ingredients: Ingredient[] = $state([]);
  let instructions: string[] = $state([]);

  let parseAbort: AbortController | null = null;
  let saveAbort: AbortController | null = null;

  onDestroy(() => {
    parseAbort?.abort();
    saveAbort?.abort();
    if (imageUrl) URL.revokeObjectURL(imageUrl);
  });

  function setRecipeFromParsed(parsed: Recipe) {
    title = parsed.title || '';
    ingredients = parsed.ingredients?.length
      ? parsed.ingredients.map((i) => ({
          amount: i.amount ?? null,
          unit: i.unit ?? '',
          name: i.name ?? '',
        }))
      : [{ amount: null, unit: '', name: '' }];
    instructions = parsed.instructions?.length ? [...parsed.instructions] : [''];
  }

  async function handleFile(file: File) {
    parseError = null;
    try {
      const blob = await normalizeImage(file);
      imageBlob = blob;
      if (imageUrl) URL.revokeObjectURL(imageUrl);
      imageUrl = URL.createObjectURL(blob);
      step = { kind: 'parsing', image: blob };
      await runParse(blob);
    } catch (e) {
      parseError =
        'Kunde inte läsa bilden. HEIC stöds inte — välj en JPEG/PNG/WebP.';
      step = { kind: 'pick' };
    }
  }

  async function runParse(blob: Blob) {
    parseError = null;
    parseAbort?.abort();
    parseAbort = new AbortController();
    step = { kind: 'parsing', image: blob };
    try {
      const resp = await parseRecipeImage(blob, parseAbort.signal);
      setRecipeFromParsed(resp.parsed);
      step = { kind: 'review', image: blob };
    } catch (e) {
      if ((e as Error).name === 'AbortError') {
        step = { kind: 'pick' };
        return;
      }
      parseError = (e as Error).message || 'Kunde inte tolka receptet';
      step = { kind: 'review', image: blob };
      // Fall back to manual entry on parse failure: empty form, user
      // can still save without re-uploading the image.
      if (!title && ingredients.length === 0 && instructions.length === 0) {
        ingredients = [{ amount: null, unit: '', name: '' }];
        instructions = [''];
      }
    }
  }

  function cancelParse() {
    parseAbort?.abort();
    if ('image' in step) {
      step = { kind: 'review', image: step.image };
      ingredients = ingredients.length ? ingredients : [{ amount: null, unit: '', name: '' }];
      instructions = instructions.length ? instructions : [''];
    } else {
      step = { kind: 'pick' };
    }
  }

  function addIngredient() {
    ingredients = [...ingredients, { amount: null, unit: '', name: '' }];
  }

  function removeIngredient(idx: number) {
    ingredients = ingredients.filter((_, i) => i !== idx);
  }

  function addInstruction() {
    instructions = [...instructions, ''];
  }

  function removeInstruction(idx: number) {
    instructions = instructions.filter((_, i) => i !== idx);
  }

  function moveInstruction(idx: number, delta: number) {
    const target = idx + delta;
    if (target < 0 || target >= instructions.length) return;
    const next = [...instructions];
    [next[idx], next[target]] = [next[target], next[idx]];
    instructions = next;
  }

  async function handleSave() {
    if (!imageBlob) return;
    if (!title.trim()) {
      saveError = 'Titel krävs';
      return;
    }
    saveError = null;
    saveAbort = new AbortController();
    step = { kind: 'saving', image: imageBlob };
    try {
      const resp = await saveRecipe(
        {
          title: title.trim(),
          ingredients: ingredients
            .filter((i) => i.name.trim())
            .map((i) => ({
              amount: i.amount,
              unit: i.unit?.trim() || '',
              name: i.name.trim(),
            })),
          instructions: instructions.filter((s) => s.trim()).map((s) => s.trim()),
        },
        imageBlob,
        saveAbort.signal
      );
      onSaved(resp.recipe.id);
    } catch (e) {
      saveError = (e as Error).message || 'Kunde inte spara receptet';
      step = { kind: 'review', image: imageBlob };
    }
  }
</script>

<div
  class="modal-backdrop"
  role="presentation"
  onclick={(e) => {
    if (e.target === e.currentTarget) onClose();
  }}
  transition:fade={{ duration: 150 }}
>
  <div class="modal" role="dialog" aria-modal="true" aria-label="Lägg till recept">
    <header>
      <h2>Lägg till recept</h2>
      <button type="button" class="close" aria-label="Stäng" onclick={onClose}>×</button>
    </header>

    {#if step.kind === 'pick'}
      <div class="pick-area">
        <p>Välj eller fotografera ett recept.</p>
        <div class="pick-buttons">
          <label class="file-button">
            <input
              type="file"
              accept="image/*"
              capture="environment"
              onchange={(e) => {
                const f = (e.target as HTMLInputElement).files?.[0];
                if (f) handleFile(f);
              }}
            />
            <span aria-hidden="true">📷</span> Ta foto
          </label>
          <label class="file-button secondary">
            <input
              type="file"
              accept="image/*,image/heic,image/heif"
              onchange={(e) => {
                const f = (e.target as HTMLInputElement).files?.[0];
                if (f) handleFile(f);
              }}
            />
            <span aria-hidden="true">🖼</span> Välj från album
          </label>
        </div>
        {#if parseError}
          <p class="error">{parseError}</p>
        {/if}
      </div>
    {:else if step.kind === 'parsing'}
      <div class="parsing">
        {#if imageUrl}
          <img src={imageUrl} alt="" class="preview" />
        {/if}
        <p>Läser receptet…</p>
        <button type="button" class="secondary" onclick={cancelParse}>
          Avbryt
        </button>
      </div>
    {:else if step.kind === 'review' || step.kind === 'saving'}
      <div class="review">
        {#if imageUrl}
          <img src={imageUrl} alt="" class="preview" />
        {/if}
        {#if parseError}
          <div class="error-banner">
            <p>{parseError}</p>
            <button
              type="button"
              class="secondary"
              onclick={() => imageBlob && runParse(imageBlob)}
            >
              Analysera igen
            </button>
          </div>
        {/if}
        <label>
          Titel
          <input type="text" bind:value={title} maxlength={200} />
        </label>

        <fieldset>
          <legend>Ingredienser</legend>
          {#each ingredients as ing, i}
            <div class="ing-row">
              <input
                type="number"
                step="any"
                placeholder="Mängd"
                bind:value={ing.amount}
                aria-label="Mängd"
              />
              <input
                type="text"
                placeholder="Enhet (dl, g…)"
                bind:value={ing.unit}
                aria-label="Enhet"
                maxlength={32}
              />
              <input
                type="text"
                placeholder="Ingrediens"
                bind:value={ing.name}
                aria-label="Ingrediens"
                maxlength={2000}
              />
              <button
                type="button"
                class="icon-btn"
                aria-label="Ta bort ingrediens"
                onclick={() => removeIngredient(i)}
              >
                ×
              </button>
            </div>
          {/each}
          <button type="button" class="secondary" onclick={addIngredient}>
            + Lägg till ingrediens
          </button>
        </fieldset>

        <fieldset>
          <legend>Steg</legend>
          {#each instructions as _, i}
            <div class="step-row">
              <span class="step-num">{i + 1}.</span>
              <textarea
                bind:value={instructions[i]}
                rows="2"
                aria-label={`Steg ${i + 1}`}
                maxlength={2000}
              ></textarea>
              <div class="step-controls">
                <button
                  type="button"
                  class="icon-btn"
                  aria-label="Flytta upp"
                  onclick={() => moveInstruction(i, -1)}
                  disabled={i === 0}
                >↑</button>
                <button
                  type="button"
                  class="icon-btn"
                  aria-label="Flytta ner"
                  onclick={() => moveInstruction(i, 1)}
                  disabled={i === instructions.length - 1}
                >↓</button>
                <button
                  type="button"
                  class="icon-btn"
                  aria-label="Ta bort steg"
                  onclick={() => removeInstruction(i)}
                >×</button>
              </div>
            </div>
          {/each}
          <button type="button" class="secondary" onclick={addInstruction}>
            + Lägg till steg
          </button>
        </fieldset>

        {#if saveError}
          <p class="error">{saveError}</p>
        {/if}

        <div class="actions">
          <button type="button" class="secondary" onclick={onClose}>Avbryt</button>
          <button
            type="button"
            class="primary"
            onclick={handleSave}
            disabled={step.kind === 'saving'}
          >
            {step.kind === 'saving' ? 'Sparar…' : 'Spara'}
          </button>
        </div>
      </div>
    {/if}
  </div>
</div>

<style>
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    padding: var(--spacing-md);
  }

  .modal {
    background: var(--card-bg);
    color: var(--text-primary);
    border-radius: var(--radius-lg);
    max-width: 720px;
    width: 100%;
    max-height: 92vh;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    box-shadow: var(--shadow-lg, 0 12px 24px rgba(0, 0, 0, 0.2));
  }

  header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: var(--spacing-md) var(--spacing-lg);
    border-bottom: 1px solid var(--border-color, rgba(0, 0, 0, 0.1));
  }

  header h2 {
    margin: 0;
    font-size: var(--font-size-lg);
  }

  .close {
    background: transparent;
    border: none;
    font-size: 24px;
    cursor: pointer;
    color: var(--text-primary);
  }

  .pick-area,
  .parsing,
  .review {
    padding: var(--spacing-lg);
    display: flex;
    flex-direction: column;
    gap: var(--spacing-md);
  }

  .preview {
    max-width: 100%;
    max-height: 280px;
    object-fit: contain;
    border-radius: var(--radius-md);
    align-self: center;
  }

  .pick-buttons {
    display: flex;
    flex-wrap: wrap;
    gap: var(--spacing-sm);
  }

  .file-button {
    display: inline-flex;
    align-items: center;
    gap: var(--spacing-xs);
    padding: var(--spacing-sm) var(--spacing-md);
    background: var(--primary-color, #4a90e2);
    color: white;
    border-radius: var(--radius-full);
    cursor: pointer;
    font-weight: var(--font-weight-semibold);
  }

  .file-button.secondary {
    background: transparent;
    /* Modal sits on --card-bg (themed); --text-primary contrasts
       correctly in both light and dark. The previous version used
       --text-on-primary, which is always white and went invisible
       on the white light-mode card. */
    color: var(--text-primary);
    border: 1px solid var(--border-color, rgba(0, 0, 0, 0.2));
  }

  .file-button input {
    display: none;
  }

  fieldset {
    border: 1px solid var(--border-color, rgba(0, 0, 0, 0.1));
    border-radius: var(--radius-md);
    padding: var(--spacing-md);
    display: flex;
    flex-direction: column;
    gap: var(--spacing-sm);
  }

  legend {
    font-weight: var(--font-weight-semibold);
    padding: 0 var(--spacing-sm);
  }

  .ing-row {
    display: grid;
    grid-template-columns: 70px 90px 1fr 32px;
    gap: var(--spacing-xs);
    align-items: center;
  }

  .step-row {
    display: grid;
    grid-template-columns: 28px 1fr auto;
    gap: var(--spacing-xs);
    align-items: start;
  }

  .step-num {
    font-weight: var(--font-weight-semibold);
    padding-top: var(--spacing-sm);
  }

  .step-controls {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .icon-btn {
    background: transparent;
    border: 1px solid var(--border-color, rgba(0, 0, 0, 0.1));
    border-radius: var(--radius-sm);
    width: 28px;
    height: 28px;
    cursor: pointer;
    color: var(--text-primary);
  }

  .icon-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  input,
  textarea {
    padding: var(--spacing-xs) var(--spacing-sm);
    border: 1px solid var(--border-color, rgba(0, 0, 0, 0.15));
    border-radius: var(--radius-sm);
    font-size: var(--font-size-base);
    font-family: inherit;
    color: var(--text-primary);
    /* See RecipeDetailView: --card-bg is themed; the previous
       var(--surface-bg, #fff) fallback rendered white-on-white in
       dark mode. */
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

  .primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
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

  .error,
  .error-banner {
    color: var(--error-color, #c0392b);
  }

  .error-banner {
    display: flex;
    align-items: center;
    gap: var(--spacing-sm);
    padding: var(--spacing-sm);
    background: rgba(192, 57, 43, 0.08);
    border-radius: var(--radius-sm);
  }
</style>
