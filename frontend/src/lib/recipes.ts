import type {
  RecipeListResponse,
  RecipeDetailResponse,
  RecipeParseResponse,
  Recipe,
  Ingredient,
} from "./types"

/**
 * Build the absolute base URL for the recipe REST API. The frontend
 * runs at the same origin as the backend, optionally under a secret
 * path prefix configured via SHARED_SECRET. This MUST match the
 * wsUrl logic in TodoList.svelte exactly so that a deployment served
 * from `/<secret>/` (or `/<secret>` with no trailing slash) resolves
 * the recipes path the same way the WebSocket does.
 *
 * Recipe routes have NO application-level authentication. Access
 * control is the operator's deployment posture (SHARED_SECRET +
 * CIDR_WHITELIST or network/reverse-proxy controls). All requests use
 * `credentials: 'same-origin'` so any future cookie-based session can
 * be added without changing call sites.
 */
export function recipeApiBase(): string {
  if (import.meta.env.VITE_BACKEND_URL) {
    // Mirrors TodoList.svelte's VITE_BACKEND_URL parsing so a value
    // like `https://example.com/secret-123` is preserved as the path
    // prefix instead of being collapsed to bare host+protocol.
    const backendUrl = String(import.meta.env.VITE_BACKEND_URL)
    const protocolMatch = backendUrl.match(/^(https?):\/\//)
    const protocol = protocolMatch ? `${protocolMatch[1]}:` : "https:"
    const urlWithoutProtocol = backendUrl.replace(/^https?:\/\//, "")
    const [host, ...pathParts] = urlWithoutProtocol.split("/")
    const path = pathParts.length > 0 ? "/" + pathParts.join("/") : ""
    // Strip a trailing slash if any so we can append `/api/v1/...`
    return `${protocol}//${host}${path}`.replace(/\/$/, "")
  }
  // Mirrors the wsUrl branch in TodoList.svelte: derive the directory
  // the SPA is served from. We force a trailing slash, then drop it
  // so callers can append `/api/v1/...` cleanly.
  const pathname = window.location.pathname
  const basePath = pathname.endsWith("/") ? pathname : pathname + "/"
  const trimmed = basePath.replace(/\/$/, "")
  return `${window.location.protocol}//${window.location.host}${trimmed}`
}

async function jsonOrThrow<T>(resp: Response): Promise<T> {
  if (!resp.ok) {
    let msg = `HTTP ${resp.status}`
    try {
      const data = await resp.json()
      if (data && typeof data.error === "string") msg = data.error
    } catch {}
    throw new Error(msg)
  }
  return (await resp.json()) as T
}

/**
 * Build the same-origin URL for a recipe's image sidecar.
 *
 * The backend already returns an absolute path in `imageUrl`, but it
 * is the SPA-side path (e.g. `/dev/api/v1/recipes/<id>/image` when
 * SHARED_SECRET=dev) which is correct for production where the SPA
 * lives at `/dev/`, but wrong for `npm run dev` where the SPA is
 * hosted by Vite at `localhost:5173/` and only `/api/...` is proxied.
 * Loading `/dev/...` in dev silently returns index.html as text/html,
 * so the <img> tag fails and the recipe photo never appears in either
 * normal or cook mode.
 *
 * Building the URL from `recipeApiBase()` reuses the same logic the
 * REST calls already trust, so an image URL works in every deployment
 * without conditional code in the components.
 */
export function recipeImageUrl(id: string): string {
  return `${recipeApiBase()}/api/v1/recipes/${encodeURIComponent(id)}/image`
}

export async function listRecipes(): Promise<RecipeListResponse> {
  const resp = await fetch(`${recipeApiBase()}/api/v1/recipes`, {
    credentials: "same-origin",
  })
  const data = await jsonOrThrow<RecipeListResponse>(resp)
  // See recipeImageUrl(): replace the backend's SPA-relative imageUrl
  // with one that goes through the dev proxy / same-origin base.
  return {
    ...data,
    recipes: data.recipes.map((r) => ({...r, imageUrl: recipeImageUrl(r.id)})),
  }
}

export async function getRecipe(id: string): Promise<RecipeDetailResponse> {
  const resp = await fetch(`${recipeApiBase()}/api/v1/recipes/${encodeURIComponent(id)}`, {
    credentials: "same-origin",
  })
  const data = await jsonOrThrow<RecipeDetailResponse>(resp)
  return {...data, imageUrl: recipeImageUrl(id)}
}

/**
 * Parse a recipe image without persisting it. The same Blob the caller
 * has in memory should be reused for saveRecipe so we never re-upload.
 */
export async function parseRecipeImage(
  image: Blob,
  signal?: AbortSignal
): Promise<RecipeParseResponse> {
  const form = new FormData()
  form.append("image", image, "recipe.jpg")
  const resp = await fetch(`${recipeApiBase()}/api/v1/recipes/parse`, {
    method: "POST",
    credentials: "same-origin",
    body: form,
    signal,
  })
  return jsonOrThrow(resp)
}

export interface RecipeMetadataInput {
  title: string
  ingredients: Ingredient[]
  instructions: string[]
}

export async function saveRecipe(
  metadata: RecipeMetadataInput,
  image: Blob,
  signal?: AbortSignal
): Promise<RecipeDetailResponse> {
  const form = new FormData()
  form.append("image", image, "recipe.jpg")
  form.append("metadata", JSON.stringify(metadata))
  const resp = await fetch(`${recipeApiBase()}/api/v1/recipes`, {
    method: "POST",
    credentials: "same-origin",
    body: form,
    signal,
  })
  const data = await jsonOrThrow<RecipeDetailResponse>(resp)
  return {...data, imageUrl: recipeImageUrl(data.recipe.id)}
}

export async function updateRecipe(
  id: string,
  patch: Partial<Pick<Recipe, "title" | "ingredients" | "instructions">>
): Promise<RecipeDetailResponse> {
  const resp = await fetch(`${recipeApiBase()}/api/v1/recipes/${encodeURIComponent(id)}`, {
    method: "PATCH",
    headers: {"Content-Type": "application/json"},
    credentials: "same-origin",
    body: JSON.stringify(patch),
  })
  const data = await jsonOrThrow<RecipeDetailResponse>(resp)
  return {...data, imageUrl: recipeImageUrl(id)}
}

export async function deleteRecipe(id: string): Promise<void> {
  const resp = await fetch(`${recipeApiBase()}/api/v1/recipes/${encodeURIComponent(id)}`, {
    method: "DELETE",
    credentials: "same-origin",
  })
  if (!resp.ok) {
    throw new Error(`HTTP ${resp.status}`)
  }
}

/**
 * Returns true when the file looks like an iPhone HEIC/HEIF photo.
 * Both the MIME type and the filename are checked because some
 * browsers/desktop file pickers report `application/octet-stream` or
 * an empty string for HEIC files dragged in from a phone.
 *
 * Exported for the upload modal so we can route HEIC files straight
 * to the server (which transcodes them to JPEG with libheif/goheif)
 * instead of trying to run them through `createImageBitmap`, which
 * silently fails on every desktop browser today.
 */
export function isHeicLike(file: File | Blob): boolean {
  const type = (file as File).type ?? ""
  if (type === "image/heic" || type === "image/heif") return true
  const name = (file as File).name ?? ""
  return /\.(heic|heif)$/i.test(name)
}

/**
 * normalizeImage handles two things before upload:
 *
 *  1. EXIF rotation: createImageBitmap with imageOrientation: 'from-image'
 *     re-renders the image so the on-disk pixel buffer already reflects
 *     the photographer's intent. This avoids surprising landscape/portrait
 *     swaps on first display.
 *  2. Resize: the longest edge is capped at 2048 px. Phone photos are
 *     routinely 4000+px wide, which would inflate uploads and thumbnail
 *     storage with no readable detail gain.
 *
 * HEIC/HEIF files are returned UNCHANGED: only Safari can decode them
 * via createImageBitmap, and we don't want a desktop user staring at
 * "kunde inte tolka receptet" for what is a perfectly valid iPhone
 * photo. The backend transcodes HEIC to JPEG before storing or sending
 * to the LLM, so passing the raw bytes through is the correct path.
 *
 * Returns the input Blob (HEIC) or a JPEG Blob ready for both
 * /recipes/parse and /recipes.
 */
export async function normalizeImage(file: File | Blob): Promise<Blob> {
  if (isHeicLike(file)) {
    return file
  }
  // Attempt to honor EXIF orientation if the browser supports it.
  let bitmap: ImageBitmap
  try {
    bitmap = await createImageBitmap(file, {imageOrientation: "from-image"} as any)
  } catch {
    bitmap = await createImageBitmap(file)
  }
  const maxEdge = 2048
  const longest = Math.max(bitmap.width, bitmap.height)
  const scale = longest > maxEdge ? maxEdge / longest : 1
  const targetW = Math.round(bitmap.width * scale)
  const targetH = Math.round(bitmap.height * scale)

  const canvas =
    typeof OffscreenCanvas !== "undefined"
      ? new OffscreenCanvas(targetW, targetH)
      : (() => {
          const c = document.createElement("canvas")
          c.width = targetW
          c.height = targetH
          return c
        })()
  const ctx = (canvas as any).getContext("2d")
  if (!ctx) throw new Error("canvas 2d context unavailable")
  ctx.drawImage(bitmap, 0, 0, targetW, targetH)
  bitmap.close?.()

  if (typeof (canvas as any).convertToBlob === "function") {
    return await (canvas as OffscreenCanvas).convertToBlob({
      type: "image/jpeg",
      quality: 0.85,
    })
  }
  return await new Promise<Blob>((resolve, reject) => {
    ;(canvas as HTMLCanvasElement).toBlob(
      (blob) => (blob ? resolve(blob) : reject(new Error("toBlob returned null"))),
      "image/jpeg",
      0.85
    )
  })
}
