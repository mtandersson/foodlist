/**
 * Toggle the page viewport meta so the browser allows native
 * pinch-to-zoom. Used by the recipe image lightbox.
 *
 * The app's index.html locks the viewport with
 *   maximum-scale=1.0, user-scalable=no
 * to prevent accidental page zoom while editing the shopping list.
 * That same lock means a fullscreen image overlay can't be pinched
 * either - which is exactly the gesture the user wants on a phone
 * looking at a recipe photo.
 *
 * Rather than re-implement pinch + pan + double-tap-to-toggle in
 * JavaScript (the previous approach used PointerEvents and a
 * scale/translate transform), we just hand the zoom to the browser
 * for the duration of the lightbox: the lightbox is a fullscreen
 * overlay, so the visual viewport effectively *is* the image.
 * iOS Safari and Android Chrome both honor a runtime change of the
 * viewport meta `content` attribute, and re-applying the locked
 * value on close hard-resets the zoom level.
 *
 * Returns a `restore()` callback that puts the original meta back.
 * The callback is idempotent so a defensive double-call from a
 * component that unmounts oddly cannot bounce the page back into
 * pinch-zoom mode.
 */

const PINCH_ZOOM_CONTENT =
  "width=device-width, initial-scale=1.0, maximum-scale=5.0, user-scalable=yes, viewport-fit=cover"

export function enablePinchZoom(): () => void {
  if (typeof document === "undefined") return () => {}
  const meta = document.querySelector(
    'meta[name="viewport"]'
  ) as HTMLMetaElement | null
  if (!meta) return () => {}

  const original = meta.getAttribute("content") ?? ""
  meta.setAttribute("content", PINCH_ZOOM_CONTENT)

  let restored = false
  return () => {
    if (restored) return
    restored = true
    meta.setAttribute("content", original)
  }
}
