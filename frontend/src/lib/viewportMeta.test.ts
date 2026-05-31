import {describe, it, expect, beforeEach, afterEach} from "vitest"
import {enablePinchZoom} from "./viewportMeta"

const LOCKED = "width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no, viewport-fit=cover"

describe("enablePinchZoom", () => {
  beforeEach(() => {
    document.head.querySelectorAll('meta[name="viewport"]').forEach((n) => n.remove())
    const m = document.createElement("meta")
    m.setAttribute("name", "viewport")
    m.setAttribute("content", LOCKED)
    document.head.appendChild(m)
  })
  afterEach(() => {
    document.head.querySelectorAll('meta[name="viewport"]').forEach((n) => n.remove())
  })

  it("rewrites the viewport meta to allow pinch zoom", () => {
    enablePinchZoom()
    const c = document.querySelector('meta[name="viewport"]')!.getAttribute("content")!
    expect(c).toMatch(/user-scalable=yes/)
    expect(c).not.toMatch(/user-scalable=no/)
    // The fixed maximum-scale=1.0 is what iOS Safari actually keys
    // off to lock zoom; without raising it, pinch is still capped.
    expect(c).not.toMatch(/maximum-scale=1\.0/)
  })

  it("restore() puts the exact original content back", () => {
    const m = document.querySelector('meta[name="viewport"]')!
    const original = m.getAttribute("content")!
    const restore = enablePinchZoom()
    restore()
    expect(m.getAttribute("content")).toBe(original)
  })

  it("does not throw when the viewport meta is missing", () => {
    document.querySelectorAll('meta[name="viewport"]').forEach((n) => n.remove())
    let restore: () => void = () => {}
    expect(() => {
      restore = enablePinchZoom()
    }).not.toThrow()
    expect(() => restore()).not.toThrow()
  })

  it("calling restore twice is a no-op (defensive against double-unmount)", () => {
    const m = document.querySelector('meta[name="viewport"]')!
    const original = m.getAttribute("content")!
    const restore = enablePinchZoom()
    restore()
    // A second restore must not toggle the meta back to the
    // pinch-friendly version - we want a clean idempotent contract.
    restore()
    expect(m.getAttribute("content")).toBe(original)
  })
})
