import {describe, it, expect, vi, beforeEach, afterEach} from "vitest"
import viteConfigFn from "../../vite.config"
import {extractSharedSecretFromEnv} from "../../vite.config"
import {isHeicLike, normalizeImage, recipeImageUrl, listRecipes, getRecipe} from "./recipes"

/**
 * The Vite dev server proxies WS and MCP traffic to the Go backend
 * (see vite.config.ts), but the recipe REST API also lives on the
 * backend. Without a proxy entry for `/api/v1/...`, Vite's dev server
 * falls back to serving `index.html` for unknown paths - that HTML
 * body produces a JSON.parse error in `recipes.ts` and surfaces in
 * the UI as
 *
 *   "Kunde inte ladda recept: JSON.parse: unexpected character at
 *    line 1 column 1 of the JSON data"
 *
 * Once that proxy exists, the *next* trap is the backend's secret
 * path prefix: with SHARED_SECRET=dev the backend mounts every recipe
 * route under /dev/api/v1/recipes, so Vite forwarding the unprefixed
 * URL still gets a 404. The proxy MUST rewrite to add that prefix.
 *
 * These tests pin both behaviors so the dev experience cannot
 * silently regress.
 */
describe("vite dev proxy", () => {
  const resolved =
    typeof viteConfigFn === "function"
      ? (viteConfigFn as any)({mode: "development", command: "serve"})
      : viteConfigFn

  it("proxies the recipe REST API to the backend in dev", () => {
    const proxy = resolved?.server?.proxy ?? {}
    const keys = Object.keys(proxy)
    const matched = keys.find((k) => {
      // Accept either an exact `/api/v1/recipes` entry or any prefix
      // that covers it (e.g. `/api/v1` or `/api`). A missing prefix is
      // exactly what produces the JSON.parse bug above.
      const probe = "/api/v1/recipes"
      if (k.startsWith("^")) return new RegExp(k).test(probe)
      return probe === k || probe.startsWith(k.endsWith("/") ? k : k + "/")
    })
    expect(matched, `expected vite.config server.proxy to cover /api/v1/recipes; got keys=${JSON.stringify(keys)}`).toBeDefined()

    const target = (proxy as any)[matched as string]
    // Either a string target or a ProxyOptions with `target`. Both must
    // point at an HTTP(S) backend - never ws://, since this is REST.
    const url = typeof target === "string" ? target : target?.target
    expect(typeof url === "string" && /^https?:\/\//.test(url)).toBe(true)
  })

  describe("extractSharedSecretFromEnv", () => {
    it("returns the secret value", () => {
      expect(extractSharedSecretFromEnv("SHARED_SECRET=dev\n")).toBe("dev")
    })
    it("trims surrounding whitespace and quotes", () => {
      expect(extractSharedSecretFromEnv('SHARED_SECRET = "my secret" \n')).toBe(
        "my secret"
      )
      expect(extractSharedSecretFromEnv("SHARED_SECRET='abc'\n")).toBe("abc")
    })
    it("ignores commented and blank lines", () => {
      const text = "# SHARED_SECRET=ignored\n\nSHARED_SECRET=real\n"
      expect(extractSharedSecretFromEnv(text)).toBe("real")
    })
    it("returns empty string when not set", () => {
      expect(extractSharedSecretFromEnv("DATA_DIR=.\n")).toBe("")
      expect(extractSharedSecretFromEnv("")).toBe("")
    })
  })

  it("attaches a rewrite() to the /api proxy entry so the secret prefix can be added in dev", () => {
    // We deliberately do NOT assert the *value* of the rewrite here:
    // that depends on backend/.env being present in the working tree
    // with SHARED_SECRET=dev, which holds locally but not in CI. The
    // pure parser is covered above; here we only pin that the proxy
    // is wired up to do *some* rewrite, which is what catches a
    // future "removed the rewrite hook entirely" regression.
    const proxy = resolved?.server?.proxy ?? {}
    const apiEntry = proxy["/api"] as any
    expect(apiEntry, "missing /api proxy entry").toBeDefined()
    expect(apiEntry.rewrite, "expected a rewrite() that adds the secret prefix").toBeTypeOf("function")
  })
})

describe("isHeicLike", () => {
  it("matches the iOS MIME types", () => {
    expect(isHeicLike(new Blob([], {type: "image/heic"}))).toBe(true)
    expect(isHeicLike(new Blob([], {type: "image/heif"}))).toBe(true)
  })
  it("matches by filename when MIME is missing", () => {
    expect(isHeicLike(new File([], "IMG_1234.HEIC"))).toBe(true)
    expect(isHeicLike(new File([], "IMG_1234.heif"))).toBe(true)
    expect(isHeicLike(new File([], "blob.bin", {type: ""}))).toBe(false)
  })
  it("rejects normal raster mimes", () => {
    expect(isHeicLike(new Blob([], {type: "image/jpeg"}))).toBe(false)
    expect(isHeicLike(new File([], "photo.jpg", {type: "image/jpeg"}))).toBe(false)
  })
})

describe("normalizeImage", () => {
  it("returns HEIC blobs unchanged so the server can transcode them", async () => {
    // createImageBitmap is unavailable in jsdom, so a HEIC file would
    // throw if normalizeImage tried to decode it. The bypass is what
    // we're verifying.
    const heic = new File([new Uint8Array([0, 0, 0, 0x20, 0x66, 0x74, 0x79, 0x70, 0x68, 0x65, 0x69, 0x63])], "IMG_1234.HEIC", {
      type: "image/heic",
    })
    const out = await normalizeImage(heic)
    expect(out).toBe(heic)
  })
})

// recipeImageUrl is the helper that fixes "image not loading in dev":
// the backend returns an absolute path with the SHARED_SECRET prefix
// (e.g. /dev/api/v1/recipes/<id>/image) so production deployments work
// without any client logic, but in the Vite dev server that path has
// no proxy and the browser ends up loading index.html as `text/html`,
// breaking the <img> tag in both normal and cook mode. The frontend
// must therefore build the URL through its own /api proxy entry.
describe("recipeImageUrl", () => {
  const originalLocation = window.location

  beforeEach(() => {
    // jsdom's location is read-only via assignment but writable via
    // defineProperty. Mirror the dev server: SPA at :5173/, no path.
    Object.defineProperty(window, "location", {
      value: new URL("http://localhost:5173/") as unknown as Location,
      configurable: true,
    })
  })
  afterEach(() => {
    Object.defineProperty(window, "location", {
      value: originalLocation,
      configurable: true,
    })
  })

  it("returns a URL the Vite /api proxy can rewrite (no /dev prefix on the client)", () => {
    const url = recipeImageUrl("832a7078-f8af-40c6-a810-01cc15f0c123")
    // The client-visible URL must hit /api/... so Vite's proxy adds
    // the secret prefix. If we hand back /dev/api/... the browser
    // would request it from the dev server directly and get HTML.
    expect(url).toMatch(/\/api\/v1\/recipes\/[0-9a-f-]+\/image$/)
    expect(url).not.toContain("/dev/")
  })

  it("URL-encodes the recipe id", () => {
    const url = recipeImageUrl("a/b%c")
    expect(url).toContain("a%2Fb%25c")
  })
})

// listRecipes/getRecipe must replace the backend-supplied imageUrl
// with the client-built one so the <img> tag works regardless of
// where the SPA is hosted (Vite dev, /<secret>/ in prod, root in
// prod). Without this rewrite the dev experience silently shows no
// image because Vite serves index.html for /dev/api/... requests.
describe("imageUrl rewriting in API responses", () => {
  let fetchMock: ReturnType<typeof vi.fn>
  beforeEach(() => {
    fetchMock = vi.fn()
    vi.stubGlobal("fetch", fetchMock)
    Object.defineProperty(window, "location", {
      value: new URL("http://localhost:5173/") as unknown as Location,
      configurable: true,
    })
  })
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it("getRecipe replaces /dev/... with a Vite-proxiable URL", async () => {
    fetchMock.mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          recipe: {
            id: "abc",
            title: "t",
            ingredients: [],
            instructions: [],
            imageFilename: "abc.jpg",
            imageMime: "image/jpeg",
            createdAt: "2026-01-01T00:00:00Z",
            updatedAt: "2026-01-01T00:00:00Z",
          },
          imageUrl: "/dev/api/v1/recipes/abc/image",
        }),
        {status: 200, headers: {"Content-Type": "application/json"}}
      )
    )
    const resp = await getRecipe("abc")
    expect(resp.imageUrl).not.toContain("/dev/")
    expect(resp.imageUrl).toMatch(/\/api\/v1\/recipes\/abc\/image$/)
  })

  it("listRecipes replaces every imageUrl in the list", async () => {
    fetchMock.mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          recipes: [
            {id: "a", title: "A", imageUrl: "/dev/api/v1/recipes/a/image", createdAt: "x", updatedAt: "x"},
            {id: "b", title: "B", imageUrl: "/dev/api/v1/recipes/b/image", createdAt: "x", updatedAt: "x"},
          ],
        }),
        {status: 200, headers: {"Content-Type": "application/json"}}
      )
    )
    const resp = await listRecipes()
    for (const r of resp.recipes) {
      expect(r.imageUrl).not.toContain("/dev/")
      expect(r.imageUrl).toMatch(/\/api\/v1\/recipes\/[ab]\/image$/)
    }
  })
})
