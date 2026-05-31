import {describe, it, expect, beforeEach} from "vitest"
import {get} from "svelte/store"
import {
  loadRecipesRoute,
  saveRecipesRoute,
  loadRecipeDetailMode,
  saveRecipeDetailMode,
  recipesRouteStore,
  recipeDetailModeStore,
  RECIPES_ROUTE_KEY,
  RECIPES_DETAIL_MODE_KEY,
} from "./recipesState"

// The recipes tab is mounted/unmounted with the rest of the view
// when the user toggles between Inköp / Kategorier / Recept etc., so
// any in-tab state stored as Svelte $state is wiped on every switch.
// The user wants their tab state preserved: if they were viewing a
// specific recipe (and inside it, Cook mode), switching to Inköp and
// back must restore exactly that.
//
// loadRecipesRoute / saveRecipesRoute is the localStorage-backed
// persistence layer the components hook into. It MUST tolerate every
// way localStorage can fail (private mode, quota, hostile values from
// another version) without crashing the SPA - a thrown error from a
// helper that runs on first paint is what bricks the tab.
describe("recipes route persistence", () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it("returns null when nothing is stored (fresh user)", () => {
    expect(loadRecipesRoute()).toBeNull()
  })

  it("treats list as the default and does not persist it", () => {
    // The list view is the implicit default; clearing rather than
    // serializing keeps localStorage tidy and avoids reviving a
    // stale shape after a future schema change.
    saveRecipesRoute({kind: "list"})
    expect(loadRecipesRoute()).toBeNull()
  })

  it("round-trips a detail route with the recipe id", () => {
    saveRecipesRoute({kind: "detail", id: "abc-123"})
    expect(loadRecipesRoute()).toEqual({kind: "detail", id: "abc-123"})
  })

  it("rejects garbage shapes (kind missing, id wrong type) and returns null", () => {
    localStorage.setItem(RECIPES_ROUTE_KEY, JSON.stringify({foo: "bar"}))
    expect(loadRecipesRoute()).toBeNull()

    localStorage.setItem(
      RECIPES_ROUTE_KEY,
      JSON.stringify({kind: "detail", id: 123})
    )
    expect(loadRecipesRoute()).toBeNull()

    localStorage.setItem(RECIPES_ROUTE_KEY, "not json")
    expect(loadRecipesRoute()).toBeNull()
  })

  it("rejects unknown kinds", () => {
    localStorage.setItem(
      RECIPES_ROUTE_KEY,
      JSON.stringify({kind: "edit", id: "x"})
    )
    expect(loadRecipesRoute()).toBeNull()
  })

  it("clears the stored route when saving the list (so `back` sticks)", () => {
    // We deliberately treat list as the implicit default; storing
    // it would only bloat localStorage and surface stale ids on
    // SSR/cold-start.
    saveRecipesRoute({kind: "detail", id: "abc"})
    saveRecipesRoute({kind: "list"})
    expect(localStorage.getItem(RECIPES_ROUTE_KEY)).toBeNull()
  })
})

// The Normal/Cook toggle is a session-level preference: a user who
// is mid-cooking and switches to Inköp to add a missed ingredient
// expects to land back in Cook mode. We persist a single mode rather
// than per-recipe because opening a different recipe naturally
// follows the same intent ("I'm cooking right now").
describe("recipe detail mode persistence", () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it("defaults to normal when nothing is stored", () => {
    expect(loadRecipeDetailMode()).toBe("normal")
  })

  it("round-trips both modes", () => {
    saveRecipeDetailMode("cook")
    expect(loadRecipeDetailMode()).toBe("cook")
    saveRecipeDetailMode("normal")
    expect(loadRecipeDetailMode()).toBe("normal")
  })

  it("falls back to normal for unknown stored values", () => {
    localStorage.setItem(RECIPES_DETAIL_MODE_KEY, "haxx0r")
    expect(loadRecipeDetailMode()).toBe("normal")
  })

  // Regression: the original implementation wrote the mode via a
  // Svelte $effect. $effect fires once on mount with the just-loaded
  // value, which is fine, but it ALSO meant that any code path
  // resetting `mode` momentarily (e.g. a parent re-instantiating the
  // component with `recipeId` changing) would clobber a later save.
  // The persistence layer must therefore behave as a plain
  // synchronous write so the call site can guarantee ordering.
  it("save is synchronous and visible on the very next load", () => {
    saveRecipeDetailMode("cook")
    // No await, no flush: the next read must already see the value.
    expect(loadRecipeDetailMode()).toBe("cook")
  })
})

// The store-based API is what components consume. The tests below
// pin the contract that makes the design race-free: subscribers
// (i.e. our auto-persist hook) MUST fire synchronously on .set(),
// so a quick "toggle Cook -> switch tab" sequence cannot unmount
// the component before the localStorage write has flushed.
describe("recipes UI stores", () => {
  beforeEach(() => {
    // The stores are module-level singletons; reset them to a known
    // baseline so each test is independent. We reset BEFORE clearing
    // localStorage so the auto-persist subscriber doesn't write a
    // value we're about to clear.
    recipesRouteStore.set({kind: "list"})
    recipeDetailModeStore.set("normal")
    localStorage.clear()
  })

  it("recipesRouteStore.set persists synchronously - no flush needed", () => {
    recipesRouteStore.set({kind: "detail", id: "abc-123"})
    // The very next localStorage.getItem call - in the same tick,
    // before any microtask drain - must already see the new value.
    expect(localStorage.getItem(RECIPES_ROUTE_KEY)).toContain("abc-123")
    expect(loadRecipesRoute()).toEqual({kind: "detail", id: "abc-123"})
  })

  it("recipeDetailModeStore.set persists synchronously - no flush needed", () => {
    recipeDetailModeStore.set("cook")
    expect(localStorage.getItem(RECIPES_DETAIL_MODE_KEY)).toBe("cook")
    expect(loadRecipeDetailMode()).toBe("cook")
  })

  it("setting list clears the persisted route (still uses the saver)", () => {
    recipesRouteStore.set({kind: "detail", id: "x"})
    recipesRouteStore.set({kind: "list"})
    expect(localStorage.getItem(RECIPES_ROUTE_KEY)).toBeNull()
    expect(get(recipesRouteStore)).toEqual({kind: "list"})
  })
})
