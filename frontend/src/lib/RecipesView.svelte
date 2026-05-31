<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { fade } from 'svelte/transition';
  import { listRecipes, deleteRecipe } from './recipes';
  import type { RecipeListItem } from './types';
  import type { TodoStore } from './store';
  import RecipeUploadModal from './RecipeUploadModal.svelte';
  import RecipeDetailView from './RecipeDetailView.svelte';
  import { recipesRouteStore } from './recipesState';

  interface Props {
    store: TodoStore;
    parseEnabled: boolean;
  }

  let { store, parseEnabled }: Props = $props();

  // In-tab routing: list <-> detail. We deliberately avoid a global
  // router; this keeps the recipe tab self-contained and survives mode
  // switches without polluting the URL.
  //
  // The route lives in a module-scoped writable store
  // (recipesRouteStore) so it's already hydrated from localStorage
  // by the time this component mounts and is auto-persisted on every
  // .set() - synchronously, before the click handler returns. That
  // means a quick "open recipe -> switch to Inköp" cannot race the
  // unmount: persistence is a property of the value, not of the
  // component lifecycle.
  //
  // If the persisted recipe was deleted in the meantime, the detail
  // view shows a generic error with a Back button - we intentionally
  // do NOT validate the id up front to avoid an extra round-trip on
  // every tab switch.

  let recipes: RecipeListItem[] = $state([]);
  let loading = $state(true);
  let error: string | null = $state(null);
  let showUpload = $state(false);

  async function refresh() {
    loading = true;
    error = null;
    try {
      const resp = await listRecipes();
      // Newest first; the server already sorts but resort defensively
      // in case a future server change weakens that guarantee.
      recipes = [...resp.recipes].sort((a, b) =>
        b.createdAt.localeCompare(a.createdAt)
      );
    } catch (e) {
      error = (e as Error).message || 'Något gick fel';
    } finally {
      loading = false;
    }
  }

  // Re-fetch any time another client (or our own writes) bumps the
  // recipesVersion writable via a RecipeChanged WS broadcast.
  let lastVersion = -1;
  const unsubVersion = store.recipesVersion.subscribe((v) => {
    if (v !== lastVersion) {
      lastVersion = v;
      // Skip the very first call - onMount handles the initial load.
      if (lastVersion >= 0 && $recipesRouteStore.kind === 'list') {
        refresh();
      }
    }
  });

  onMount(() => {
    refresh();
  });

  onDestroy(() => {
    unsubVersion();
  });

  function openDetail(id: string) {
    recipesRouteStore.set({ kind: 'detail', id });
  }

  function backToList() {
    recipesRouteStore.set({ kind: 'list' });
    refresh();
  }

  async function handleDelete(id: string) {
    if (!confirm('Vill du ta bort receptet?')) return;
    try {
      await deleteRecipe(id);
      recipesRouteStore.set({ kind: 'list' });
      await refresh();
    } catch (e) {
      alert('Kunde inte ta bort receptet');
    }
  }
</script>

{#if $recipesRouteStore.kind === 'list'}
  <div class="recipes-view" transition:fade={{ duration: 150 }}>
    <div class="actions">
      {#if parseEnabled}
        <button type="button" class="primary" onclick={() => (showUpload = true)}>
          <span aria-hidden="true">📷</span> Lägg till recept
        </button>
      {:else}
        <p class="hint">
          Recept-tolkning är inte konfigurerad. Be administratören att sätta
          <code>RECIPE_LLM_*</code> för att kunna ladda upp nya recept.
        </p>
      {/if}
    </div>

    {#if loading}
      <p class="status">Laddar recept…</p>
    {:else if error}
      <p class="status error">Kunde inte ladda recept: {error}</p>
    {:else if recipes.length === 0}
      <div class="empty-state">
        <p>Inga recept än — lägg till ett med en bild.</p>
      </div>
    {:else}
      <div class="recipe-grid" role="list">
        {#each recipes as recipe (recipe.id)}
          <button
            type="button"
            class="recipe-card"
            role="listitem"
            onclick={() => openDetail(recipe.id)}
          >
            <div class="thumb">
              <img
                src={recipe.imageUrl}
                alt=""
                loading="lazy"
                decoding="async"
              />
            </div>
            <div class="title">{recipe.title}</div>
          </button>
        {/each}
      </div>
    {/if}
  </div>
{:else if $recipesRouteStore.kind === 'detail'}
  <RecipeDetailView
    recipeId={$recipesRouteStore.id}
    {store}
    onBack={backToList}
    onDelete={() => handleDelete($recipesRouteStore.kind === 'detail' ? $recipesRouteStore.id : '')}
  />
{/if}

{#if showUpload}
  <RecipeUploadModal
    onClose={() => (showUpload = false)}
    onSaved={(id) => {
      showUpload = false;
      recipesRouteStore.set({ kind: 'detail', id });
    }}
  />
{/if}

<style>
  .recipes-view {
    display: flex;
    flex-direction: column;
    gap: var(--spacing-md);
  }

  .actions {
    display: flex;
    justify-content: flex-end;
  }

  .actions .primary {
    background: var(--card-bg);
    color: var(--text-primary);
    border: none;
    padding: var(--spacing-sm) var(--spacing-md);
    border-radius: var(--radius-full);
    cursor: pointer;
    font-weight: var(--font-weight-semibold);
    box-shadow: var(--shadow-md);
  }

  .hint {
    color: var(--text-on-primary);
    opacity: 0.85;
    font-size: var(--font-size-sm);
  }

  .status {
    color: var(--text-on-primary);
    text-align: center;
    padding: var(--spacing-md);
  }

  .status.error {
    color: var(--error-color, #ffb3b3);
  }

  .empty-state {
    text-align: center;
    padding: var(--spacing-2xl) var(--spacing-lg);
    color: var(--text-on-primary);
    opacity: 0.85;
  }

  .recipe-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
    gap: var(--spacing-md);
  }

  .recipe-card {
    display: flex;
    flex-direction: column;
    background: var(--card-bg);
    border: none;
    border-radius: var(--radius-md);
    overflow: hidden;
    cursor: pointer;
    box-shadow: var(--shadow-sm);
    text-align: left;
    padding: 0;
  }

  .recipe-card:hover {
    box-shadow: var(--shadow-md);
  }

  .thumb {
    aspect-ratio: 4 / 3;
    background: var(--surface-muted);
    overflow: hidden;
  }

  .thumb img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }

  .recipe-card .title {
    padding: var(--spacing-sm) var(--spacing-md);
    font-weight: var(--font-weight-semibold);
    color: var(--text-primary);
    font-size: var(--font-size-sm);
    line-height: var(--line-height-snug);
  }
</style>
