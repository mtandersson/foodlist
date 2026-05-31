<script lang="ts">
  import { onDestroy } from 'svelte';
  import { fade } from 'svelte/transition';
  import { normalizeImage, parseRecipeImage, saveRecipe } from './recipes';
  import type { Recipe, RecipeSection } from './types';
  import { renderMarkdown } from './markdown';
  import RecipeSectionEditor from './RecipeSectionEditor.svelte';

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

  // Use the $state<T> type parameter (not a `: T | null` annotation
  // on the let) so Svelte 5 runes preserve the union under
  // tsconfig.app.json's stricter typecheck; the annotation form
  // narrows to `null` and breaks downstream property access.
  let step = $state<Step>({ kind: 'pick' });
  let parseError = $state<string | null>(null);
  let saveError = $state<string | null>(null);
  // Hold the normalized image once and reuse it for both parse and save
  // so we never re-pick or re-upload on retry.
  let imageBlob = $state<Blob | null>(null);
  let imageUrl = $state<string | null>(null);

  let title = $state('');
  let description = $state('');
  let sections = $state<RecipeSection[]>([]);

  let parseAbort: AbortController | null = null;
  let saveAbort: AbortController | null = null;

  onDestroy(() => {
    parseAbort?.abort();
    saveAbort?.abort();
    if (imageUrl) URL.revokeObjectURL(imageUrl);
  });

  function setRecipeFromParsed(parsed: Recipe) {
    title = parsed.title || '';
    description = parsed.description ?? '';
    sections = parsed.sections?.length
      ? parsed.sections.map((s) => ({
          name: s.name ?? '',
          ingredients: (s.ingredients ?? []).map((i) => ({
            amount: i.amount ?? null,
            unit: i.unit ?? '',
            name: i.name ?? '',
          })),
          instructions: [...(s.instructions ?? [])],
        }))
      : [];
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
      // can still save without re-uploading the image. The section
      // editor seeds an empty section if sections is empty.
      if (!title && sections.length === 0) {
        sections = [];
      }
    }
  }

  function cancelParse() {
    parseAbort?.abort();
    if ('image' in step) {
      step = { kind: 'review', image: step.image };
    } else {
      step = { kind: 'pick' };
    }
  }

  async function handleSave() {
    if (!imageBlob) return;
    saveError = null;
    if (!title.trim()) {
      saveError = 'Titel krävs.';
      return;
    }
    // Drop sections that wound up empty after trimming. The backend
    // would do the same, but pruning client-side keeps the wire
    // payload tight AND lets us produce a Swedish error inline
    // instead of waiting for a 400 "recipe invalid" round-trip.
    const trimmedSections = sections
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
    if (trimmedSections.length === 0) {
      saveError = 'Lägg till minst en ingrediens eller ett steg innan du sparar.';
      return;
    }
    saveAbort = new AbortController();
    step = { kind: 'saving', image: imageBlob };
    try {
      const resp = await saveRecipe(
        {
          title: title.trim(),
          description: description.trim(),
          sections: trimmedSections,
        },
        imageBlob,
        saveAbort.signal
      );
      onSaved(resp.recipe.id);
    } catch (e) {
      saveError = (e as Error).message || 'Kunde inte spara receptet.';
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

        <label>
          Beskrivning
          <textarea
            bind:value={description}
            rows="4"
            maxlength={4000}
            placeholder="Intro, portioner, källa… Markdown: **fet**, *kursiv*, listor, > citat, [länkar](https://…). Bilder och tabeller stöds inte."
          ></textarea>
        </label>
        {#if description.trim()}
          <div class="md-preview" aria-label="Förhandsvisning av beskrivning">
            <span class="md-preview-label">Förhandsvisning</span>
            <div class="recipe-description">{@html renderMarkdown(description)}</div>
          </div>
        {/if}

        <RecipeSectionEditor bind:sections />

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

  /* The preview sits INSIDE the modal/card surface, so it must NOT
     paint a fill. --surface-muted is white-glass tuned for the
     colored page background and renders invisible on the white light
     card and bright-on-dark on the dark card. Dashed border + label
     is enough visual affordance. */
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

  /* Scoped description styling shared with RecipeDetailView.
     Headings are intentionally subdued so they don't compete with
     the page title (h1) or section labels (h2). */
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
