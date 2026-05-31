/**
 * Persistence helpers for the Recipes tab.
 *
 * The Recipes tab is mounted/unmounted as the user switches between
 * shopping list / categories / recipes. Anything declared as $state
 * inside <RecipesView> or <RecipeDetailView> is therefore wiped on
 * every tab switch. Without persistence, the user's "I was viewing
 * Recipe X in Cook mode" intent is lost the moment they switch over
 * to add a missing ingredient to the shopping list.
 *
 * We model the persisted UI state as Svelte writable stores rather
 * than as component-local $state synced via $effect. The reason is
 * subtle but important: `writable.set(v)` fires every subscriber
 * SYNCHRONOUSLY before returning, so the localStorage write has
 * already happened by the time the click handler that called .set()
 * returns. There is no microtask / animation-frame window during
 * which a quick "toggle Cook -> switch tab" sequence can unmount
 * the component before the save flushes - which is exactly the bug
 * a $effect-based version exhibits, because $effect cancellation on
 * unmount silently drops the queued write.
 *
 * Persistence becomes a property of the value, not of any component.
 * Consumers just bind to the store ($recipeDetailModeStore = 'cook')
 * and the localStorage write happens for free.
 *
 * The state lives in localStorage rather than memory because the
 * user expects it to survive a full PWA reload, not just a tab
 * switch within the same session.
 *
 * Every getter is defensive: localStorage access can throw in
 * private mode / quota-exceeded scenarios, JSON parsing can throw
 * on corrupt or downgrade values, and a previous version of the app
 * could have written a shape we no longer support. A thrown error
 * from a helper that runs on first paint is what bricks the tab, so
 * we always swallow and fall back to the safe default.
 */
import {writable, type Writable} from "svelte/store"

export const RECIPES_ROUTE_KEY = "recipesRoute"
export const RECIPES_DETAIL_MODE_KEY = "recipeDetailMode"

export type RecipesRoute =
  | {kind: "list"}
  | {kind: "detail"; id: string}

export type RecipeDetailMode = "normal" | "cook"

function safeStorage(): Storage | null {
  try {
    if (typeof localStorage === "undefined") return null
    return localStorage
  } catch {
    return null
  }
}

export function loadRecipesRoute(): RecipesRoute | null {
  const ls = safeStorage()
  if (!ls) return null
  let raw: string | null
  try {
    raw = ls.getItem(RECIPES_ROUTE_KEY)
  } catch {
    return null
  }
  if (!raw) return null
  let parsed: unknown
  try {
    parsed = JSON.parse(raw)
  } catch {
    return null
  }
  if (!parsed || typeof parsed !== "object") return null
  const obj = parsed as Record<string, unknown>
  if (obj.kind === "list") return {kind: "list"}
  if (obj.kind === "detail" && typeof obj.id === "string" && obj.id.length > 0) {
    return {kind: "detail", id: obj.id}
  }
  return null
}

export function saveRecipesRoute(route: RecipesRoute): void {
  const ls = safeStorage()
  if (!ls) return
  try {
    if (route.kind === "list") {
      // The list view is the implicit default. Persisting it would
      // bloat localStorage and risk reviving a stale id on a future
      // schema change; just clear instead.
      ls.removeItem(RECIPES_ROUTE_KEY)
      return
    }
    ls.setItem(RECIPES_ROUTE_KEY, JSON.stringify(route))
  } catch {
    // Quota / private-mode failures: we accept losing persistence
    // rather than crashing the UI on a click.
  }
}

export function loadRecipeDetailMode(): RecipeDetailMode {
  const ls = safeStorage()
  if (!ls) return "normal"
  let raw: string | null
  try {
    raw = ls.getItem(RECIPES_DETAIL_MODE_KEY)
  } catch {
    return "normal"
  }
  if (raw === "cook" || raw === "normal") return raw
  return "normal"
}

export function saveRecipeDetailMode(mode: RecipeDetailMode): void {
  const ls = safeStorage()
  if (!ls) return
  try {
    ls.setItem(RECIPES_DETAIL_MODE_KEY, mode)
  } catch {
    // see saveRecipesRoute
  }
}

/**
 * Auto-persisted writable for the in-tab route. Hydrates from
 * localStorage at module load and persists every subsequent .set()
 * synchronously before the call returns. Components bind to it
 * directly:
 *
 *   import { recipesRouteStore } from './recipesState';
 *   ...
 *   {#if $recipesRouteStore.kind === 'list'} ...
 *   recipesRouteStore.set({ kind: 'detail', id });
 */
export const recipesRouteStore: Writable<RecipesRoute> = (() => {
  const initial = loadRecipesRoute() ?? {kind: "list"}
  const store = writable<RecipesRoute>(initial)
  // subscribers fire synchronously on every .set(), so the save
  // happens before the handler that called .set() returns - no
  // race with component unmount.
  store.subscribe(saveRecipesRoute)
  return store
})()

/**
 * Auto-persisted writable for the Normal/Cook toggle inside a
 * recipe. Same persistence guarantees as recipesRouteStore.
 */
export const recipeDetailModeStore: Writable<RecipeDetailMode> = (() => {
  const store = writable<RecipeDetailMode>(loadRecipeDetailMode())
  store.subscribe(saveRecipeDetailMode)
  return store
})()
