import {describe, it, expect, vi, beforeEach, afterEach} from "vitest"
import {render, screen, fireEvent, waitFor} from "@testing-library/svelte"
import {writable} from "svelte/store"
import RecipeDetailView from "./RecipeDetailView.svelte"
import type {Recipe} from "./types"
import {recipeDetailModeStore} from "./recipesState"

// Minimal stub TodoStore. We only need the slices RecipeDetailView
// touches: cookSessions store, recipesVersion store, the cook
// mutator functions, and createTodo for the +add button. Everything
// else is left undefined and never invoked by the component.
function makeStub(
  cookSessionsMap = new Map<string, Set<number>>()
) {
  const cookSessions = writable<Map<string, Set<number>>>(cookSessionsMap)
  const recipesVersion = writable<number>(0)
  const calls = {
    cookCheck: vi.fn(),
    cookUncheck: vi.fn(),
    cookReset: vi.fn(),
    createTodo: vi.fn(),
  }
  return {
    store: {
      cookSessions,
      recipesVersion,
      ...calls,
    } as any,
    calls,
  }
}

function makeRecipe(over: Partial<Recipe> = {}): Recipe {
  return {
    id: "r1",
    title: "Tacos",
    description: "",
    sections: [],
    imageFilename: "r1.jpg",
    imageMime: "image/jpeg",
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
    ...over,
  }
}

describe("RecipeDetailView", () => {
  const originalFetch = globalThis.fetch
  let fetchMock: ReturnType<typeof vi.fn>

  beforeEach(() => {
    fetchMock = vi.fn()
    vi.stubGlobal("fetch", fetchMock)
    // The Normal/Cook toggle persists across the suite via
    // localStorage; reset between tests so a "switch to cook" in one
    // test cannot leak into the next and hide the + buttons.
    recipeDetailModeStore.set("normal")
  })
  afterEach(() => {
    vi.unstubAllGlobals()
    globalThis.fetch = originalFetch
  })

  function mockGet(recipe: Recipe) {
    // RecipeDetailView issues a GET from both onMount AND the
    // initial recipesVersion subscriber tick, so we need ANY number
    // of GETs to resolve identically. Tests that also expect a
    // mutating call (PATCH) chain mockResolvedValueOnce on top.
    fetchMock.mockImplementation(async () =>
      new Response(
        JSON.stringify({recipe, imageUrl: `/api/v1/recipes/${recipe.id}/image`}),
        {status: 200, headers: {"Content-Type": "application/json"}}
      )
    )
  }

  it("single unnamed section renders the legacy h2 layout for visual parity", async () => {
    const recipe = makeRecipe({
      sections: [{
        name: "",
        ingredients: [{name: "Salt", unit: "tsk", amount: 1}],
        instructions: ["Krydda", "Servera"],
      }],
    })
    mockGet(recipe)
    const {store} = makeStub()
    render(RecipeDetailView, {props: {recipeId: "r1", store, onBack: vi.fn(), onDelete: vi.fn()}})

    await waitFor(() => expect(screen.getByText("Tacos")).toBeInTheDocument())
    // The unnamed-single layout uses the literal "Ingredienser" and
    // "Instruktioner" headings the user had before sections existed.
    expect(screen.getByRole("heading", {level: 2, name: "Ingredienser"})).toBeInTheDocument()
    expect(screen.getByRole("heading", {level: 2, name: "Instruktioner"})).toBeInTheDocument()
    // And it does NOT add a "Sektion 1" placeholder heading.
    expect(screen.queryByText(/Sektion 1/)).not.toBeInTheDocument()
  })

  it("multi-section recipe renders one card per section with h3 headings", async () => {
    const recipe = makeRecipe({
      sections: [
        {name: "Sås", ingredients: [{name: "Tomat"}], instructions: ["Mixa"]},
        {name: "Sallad", ingredients: [{name: "Lök"}], instructions: ["Strimla", "Blanda"]},
      ],
    })
    mockGet(recipe)
    const {store} = makeStub()
    render(RecipeDetailView, {props: {recipeId: "r1", store, onBack: vi.fn(), onDelete: vi.fn()}})

    await waitFor(() => expect(screen.getByRole("heading", {level: 3, name: "Sås"})).toBeInTheDocument())
    expect(screen.getByRole("heading", {level: 3, name: "Sallad"})).toBeInTheDocument()
  })

  it("multi-section cook mode uses GLOBAL step numbering across sections", async () => {
    const recipe = makeRecipe({
      sections: [
        {name: "Sås", ingredients: [], instructions: ["Mixa", "Värm"]},
        {name: "Sallad", ingredients: [], instructions: ["Strimla", "Blanda"]},
      ],
    })
    mockGet(recipe)
    const {store, calls} = makeStub()
    render(RecipeDetailView, {props: {recipeId: "r1", store, onBack: vi.fn(), onDelete: vi.fn()}})

    await waitFor(() => expect(screen.getByRole("heading", {level: 3, name: "Sallad"})).toBeInTheDocument())
    // Switch to cook mode.
    await fireEvent.click(screen.getByRole("button", {name: /Kock-läge/}))

    // Step numbers MUST be 1,2,3,4 - not 1,2,1,2 - because the cook
    // session uses a flat index and the backend bounds check is
    // recipeTotalSteps(sections).
    const stepLabels = screen.getAllByText(/^\d+\.$/).map((el) => el.textContent?.trim())
    expect(stepLabels).toEqual(["1.", "2.", "3.", "4."])

    // Clicking the third checkbox (index 2 globally = "Strimla") must
    // dispatch cookCheck with globalIdx=2.
    const checkboxes = screen.getAllByRole("checkbox")
    expect(checkboxes).toHaveLength(4)
    await fireEvent.click(checkboxes[2])
    expect(calls.cookCheck).toHaveBeenCalledWith("r1", 2)
  })

  it("multi-section normal mode dispatches add-to-list with the correct ingredient", async () => {
    const recipe = makeRecipe({
      sections: [
        {name: "Sås", ingredients: [{name: "Tomat"}], instructions: ["Mixa"]},
        {name: "Sallad", ingredients: [{name: "Lök"}, {name: "Sallad"}], instructions: ["Blanda"]},
      ],
    })
    mockGet(recipe)
    const {store, calls} = makeStub()
    render(RecipeDetailView, {props: {recipeId: "r1", store, onBack: vi.fn(), onDelete: vi.fn()}})

    await waitFor(() => expect(screen.getByRole("heading", {level: 3, name: "Sallad"})).toBeInTheDocument())

    // The three + buttons are labelled with their ingredient name;
    // grab the second-section first ingredient ("Lök").
    const addBtn = screen.getByRole("button", {name: "Lägg till Lök i listan"})
    await fireEvent.click(addBtn)
    expect(calls.createTodo).toHaveBeenCalledTimes(1)
    expect(calls.createTodo.mock.calls[0][0]).toBe("Lök")
  })

  it("renders description as sanitized markdown (script tags stripped)", async () => {
    const recipe = makeRecipe({
      description: "**4 portioner** · 30 min\n\n<script>alert(1)</script>",
      sections: [{name: "", ingredients: [{name: "Salt"}], instructions: ["Smaka"]}],
    })
    mockGet(recipe)
    const {store} = makeStub()
    const {container} = render(RecipeDetailView, {props: {recipeId: "r1", store, onBack: vi.fn(), onDelete: vi.fn()}})

    await waitFor(() => expect(screen.getByText("Tacos")).toBeInTheDocument())
    expect(container.querySelector(".recipe-description strong")?.textContent).toBe("4 portioner")
    expect(container.querySelector("script")).toBeNull()
    expect(container.innerHTML).not.toContain("alert(1)")
  })

  it("saving edits sends sections+description PATCH body (no legacy fields)", async () => {
    const recipe = makeRecipe({
      description: "intro",
      sections: [{name: "", ingredients: [{name: "Salt"}], instructions: ["Smaka"]}],
    })
    mockGet(recipe)
    const {store} = makeStub()
    render(RecipeDetailView, {props: {recipeId: "r1", store, onBack: vi.fn(), onDelete: vi.fn()}})

    await waitFor(() => expect(screen.getByText("Tacos")).toBeInTheDocument())
    await fireEvent.click(screen.getByRole("button", {name: "Redigera"}))
    await fireEvent.click(screen.getByRole("button", {name: "Spara"}))

    // Find the PATCH call regardless of how many GETs (initial mount
    // + recipesVersion subscriber tick + post-PATCH refresh) fired.
    await waitFor(() => {
      const patchCalled = fetchMock.mock.calls.some(
        ([, init]) => (init as RequestInit | undefined)?.method === "PATCH"
      )
      expect(patchCalled).toBe(true)
    })
    const patchCall = fetchMock.mock.calls.find(
      ([, init]) => (init as RequestInit | undefined)?.method === "PATCH"
    )!
    const init = patchCall[1] as RequestInit
    const body = JSON.parse(init.body as string)
    expect(body).toHaveProperty("sections")
    expect(body).toHaveProperty("description")
    expect(body).not.toHaveProperty("ingredients")
    expect(body).not.toHaveProperty("instructions")
  })
})
