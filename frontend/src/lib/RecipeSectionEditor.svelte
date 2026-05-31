<script lang="ts">
  /**
   * Shared editor for an array of recipe sections, used by
   * RecipeUploadModal (initial entry / parsed result review) and
   * RecipeDetailView (edit mode). Sections each carry a name plus
   * ingredient and instruction rows; the parent owns the section
   * array via `bind:sections`.
   *
   * The empty section template is created here so callers can
   * initialize with `[]` and have the editor backfill on add.
   */
  import type { RecipeSection } from './types';

  interface Props {
    sections: RecipeSection[];
  }

  let { sections = $bindable() }: Props = $props();

  function ensureSeed() {
    if (sections.length === 0) {
      sections = [{ name: '', ingredients: [{ amount: null, unit: '', name: '' }], instructions: [''] }];
    }
  }
  $effect(() => ensureSeed());

  function addSection() {
    sections = [
      ...sections,
      { name: '', ingredients: [{ amount: null, unit: '', name: '' }], instructions: [''] },
    ];
  }

  function removeSection(idx: number) {
    // We never allow zero sections - the validator rejects empty
    // recipes. Replacing the last section with a fresh empty one
    // matches the seed behavior and avoids a flicker where the
    // user briefly sees nothing.
    if (sections.length === 1) {
      sections = [{ name: '', ingredients: [{ amount: null, unit: '', name: '' }], instructions: [''] }];
      return;
    }
    sections = sections.filter((_, i) => i !== idx);
  }

  function addIngredient(sIdx: number) {
    const next = sections.slice();
    next[sIdx] = {
      ...next[sIdx],
      ingredients: [...next[sIdx].ingredients, { amount: null, unit: '', name: '' }],
    };
    sections = next;
  }

  function removeIngredient(sIdx: number, iIdx: number) {
    const next = sections.slice();
    next[sIdx] = {
      ...next[sIdx],
      ingredients: next[sIdx].ingredients.filter((_, i) => i !== iIdx),
    };
    sections = next;
  }

  function addStep(sIdx: number) {
    const next = sections.slice();
    next[sIdx] = { ...next[sIdx], instructions: [...next[sIdx].instructions, ''] };
    sections = next;
  }

  function removeStep(sIdx: number, iIdx: number) {
    const next = sections.slice();
    next[sIdx] = {
      ...next[sIdx],
      instructions: next[sIdx].instructions.filter((_, i) => i !== iIdx),
    };
    sections = next;
  }

  function moveStep(sIdx: number, iIdx: number, delta: number) {
    const target = iIdx + delta;
    const steps = sections[sIdx].instructions;
    if (target < 0 || target >= steps.length) return;
    const next = sections.slice();
    const swapped = steps.slice();
    [swapped[iIdx], swapped[target]] = [swapped[target], swapped[iIdx]];
    next[sIdx] = { ...next[sIdx], instructions: swapped };
    sections = next;
  }
</script>

{#each sections as section, sIdx}
  <fieldset class="section">
    <legend>
      <input
        type="text"
        class="section-name"
        placeholder="Sektion (valfritt, t.ex. Sås)"
        bind:value={section.name}
        maxlength={2000}
        aria-label={`Sektion ${sIdx + 1} namn`}
      />
      <button
        type="button"
        class="icon-btn remove-section"
        aria-label="Ta bort sektion"
        onclick={() => removeSection(sIdx)}
      >×</button>
    </legend>

    <div class="rows">
      <h4>Ingredienser</h4>
      {#each section.ingredients as ing, iIdx}
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
            placeholder="Enhet"
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
            onclick={() => removeIngredient(sIdx, iIdx)}
          >×</button>
        </div>
      {/each}
      <button type="button" class="secondary" onclick={() => addIngredient(sIdx)}>
        + Ingrediens
      </button>
    </div>

    <div class="rows">
      <h4>Steg</h4>
      {#each section.instructions as _, iIdx}
        <div class="step-row">
          <span class="step-num">{iIdx + 1}.</span>
          <textarea
            bind:value={section.instructions[iIdx]}
            rows="2"
            aria-label={`Steg ${iIdx + 1}`}
            maxlength={2000}
          ></textarea>
          <div class="step-controls">
            <button
              type="button"
              class="icon-btn"
              aria-label="Flytta upp"
              onclick={() => moveStep(sIdx, iIdx, -1)}
              disabled={iIdx === 0}
            >↑</button>
            <button
              type="button"
              class="icon-btn"
              aria-label="Flytta ner"
              onclick={() => moveStep(sIdx, iIdx, 1)}
              disabled={iIdx === section.instructions.length - 1}
            >↓</button>
            <button
              type="button"
              class="icon-btn"
              aria-label="Ta bort steg"
              onclick={() => removeStep(sIdx, iIdx)}
            >×</button>
          </div>
        </div>
      {/each}
      <button type="button" class="secondary" onclick={() => addStep(sIdx)}>
        + Steg
      </button>
    </div>
  </fieldset>
{/each}

<button type="button" class="secondary add-section" onclick={addSection}>
  + Lägg till sektion
</button>

<style>
  .section {
    border: 1px solid var(--border-color, rgba(0, 0, 0, 0.1));
    border-radius: var(--radius-md);
    padding: var(--spacing-md);
    display: flex;
    flex-direction: column;
    gap: var(--spacing-sm);
    background: var(--card-bg);
    color: var(--text-primary);
  }

  legend {
    display: flex;
    align-items: center;
    gap: var(--spacing-xs);
    padding: 0 var(--spacing-sm);
    font-weight: var(--font-weight-semibold);
    width: 100%;
  }

  .section-name {
    flex: 1;
    min-width: 0;
    padding: var(--spacing-xs) var(--spacing-sm);
    border: 1px solid var(--border-color, rgba(0, 0, 0, 0.15));
    border-radius: var(--radius-sm);
    background: var(--card-bg);
    color: var(--text-primary);
  }

  .rows {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-xs);
  }

  .rows h4 {
    margin: var(--spacing-sm) 0 0;
    font-size: var(--font-size-base);
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

  input,
  textarea {
    padding: var(--spacing-xs) var(--spacing-sm);
    border: 1px solid var(--border-color, rgba(0, 0, 0, 0.15));
    border-radius: var(--radius-sm);
    font-size: var(--font-size-base);
    font-family: inherit;
    color: var(--text-primary);
    background: var(--card-bg);
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

  .secondary {
    background: transparent;
    color: var(--text-primary);
    border: 1px solid var(--border-color, rgba(0, 0, 0, 0.2));
    padding: var(--spacing-xs) var(--spacing-sm);
    border-radius: var(--radius-full);
    cursor: pointer;
    font-weight: var(--font-weight-semibold);
    align-self: flex-start;
  }

  .add-section {
    align-self: flex-start;
    margin-top: var(--spacing-sm);
  }

  @media (max-width: 600px) {
    .ing-row {
      grid-template-columns: 60px 70px 1fr 28px;
    }
  }
</style>
