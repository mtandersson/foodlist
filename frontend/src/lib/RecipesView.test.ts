import {afterEach, describe, expect, it, vi} from "vitest"
import {render, screen, waitFor} from "@testing-library/svelte"
import {writable} from "svelte/store"
import RecipesView from "./RecipesView.svelte"
import {recipesRouteStore} from "./recipesState"

describe("RecipesView", () => {
  afterEach(() => vi.unstubAllGlobals())

  it("shows a text-only recipe without requesting a missing image", async () => {
    recipesRouteStore.set({kind: "list"})
    vi.stubGlobal("fetch", vi.fn().mockImplementation(async () => new Response(JSON.stringify({
      recipes: [{id: "plain", title: "Soup", imageUrl: "", createdAt: "2026-01-01T00:00:00Z", updatedAt: "2026-01-01T00:00:00Z"}],
    }), {status: 200})))
    const store = {recipesVersion: writable(0)} as any
    render(RecipesView, {props: {store, parseEnabled: false}})
    await waitFor(() => expect(screen.getByText("Soup")).toBeInTheDocument())
    expect(screen.getByText("Ingen bild")).toBeInTheDocument()
    expect(screen.queryByRole("img")).not.toBeInTheDocument()
  })
})
