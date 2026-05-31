// Recipe description renderer.
//
// We accept a small markdown subset from user uploads and from the
// LLM that parses recipe photos. Output is sanitized HTML safe to
// drop into a Svelte component via {@html ...}.
//
// Tag allowlist is intentionally narrow:
//   - text emphasis: strong, em
//   - block structure: p, br, blockquote
//   - lists: ul, ol, li
//   - links: a (href only, http(s) + mailto, target/rel forced)
//   - small headings: h3-h6 (h1 is the recipe title, h2 is the
//     section card label - excluded so a description heading cannot
//     impersonate them)
//   - inline code + fenced code: code, pre
//
// Notably forbidden: <img>, <style>, <script>, <iframe>, <object>,
// <svg>, <math>, on* attributes, and any href whose URL parser
// rejects it. Backed by both the marked + DOMPurify allowlist and
// the SecurityHeadersMiddleware CSP (script-src 'self', etc.).
import { marked } from "marked"
import DOMPurify from "dompurify"

const ALLOWED_TAGS = [
  "p",
  "br",
  "strong",
  "em",
  "ul",
  "ol",
  "li",
  "a",
  "blockquote",
  "h3",
  "h4",
  "h5",
  "h6",
  "code",
  "pre",
] as const

// Hard cap on description length passed to marked. The server already
// enforces maxRecipeDescriptionLen (4000 chars), but defending in
// depth means a future code path that forgets to validate cannot DoS
// the renderer with megabytes of input.
const MAX_DESCRIPTION_LEN = 4096

// sanitizeHref validates an anchor's href and returns the normalized
// safe URL or null when the link must be dropped. Uses URL parsing
// instead of regex so percent-encoded payloads, control characters,
// and protocol-relative URLs are normalized first.
function sanitizeHref(raw: string): string | null {
  if (!raw) return null
  // Strip ASCII control characters (incl. NUL, tab, newline) which
  // some bypasses inject inside the scheme to defeat regex checks.
  const cleaned = raw.trim().replace(/[\u0000-\u001F\u007F]/g, "")
  if (!cleaned) return null
  // Require an explicit safe scheme on the raw input. Relative URLs
  // (`/path`), protocol-relative (`//evil.com`), and unknown schemes
  // are all rejected. Comparing case-insensitively against the start
  // means a leading `JavaScript:` is caught even after the URL parser
  // would lowercase it.
  const lower = cleaned.toLowerCase()
  const acceptable = lower.startsWith("http://") || lower.startsWith("https://") || lower.startsWith("mailto:")
  if (!acceptable) return null
  try {
    const u = new URL(cleaned)
    const proto = u.protocol.toLowerCase()
    if (proto === "mailto:") return u.href
    if (proto === "http:" || proto === "https:") {
      // Strip phishing-style user:pass@ embedded credentials.
      if (u.username || u.password) return null
      return u.href
    }
    return null
  } catch {
    return null
  }
}

// Configure DOMPurify hooks once at module load. The hooks are
// idempotent: calling addHook multiple times for the same name only
// registers them once per handler reference, but to be safe we guard
// with a module-scoped flag.
let purifyConfigured = false
function configurePurify() {
  if (purifyConfigured) return
  purifyConfigured = true
  DOMPurify.addHook("uponSanitizeAttribute", (_node, data) => {
    if (data.attrName !== "href") return
    const safe = sanitizeHref(data.attrValue)
    if (!safe) {
      data.keepAttr = false
    } else {
      data.attrValue = safe
    }
  })
  DOMPurify.addHook("afterSanitizeAttributes", (node) => {
    if (node.tagName === "A" && node.hasAttribute("href")) {
      node.setAttribute("target", "_blank")
      // noopener: blocks window.opener tabnabbing.
      // noreferrer: hides our secret-path URL from the destination.
      node.setAttribute("rel", "noopener noreferrer")
    }
  })
}

// gfm + breaks gives newline-to-<br> semantics that match the
// inline-formatted recipe captions we expect (e.g. "**4 portioner**\n
// 30 min"). We intentionally do NOT enable mangle/headerIds extensions
// that have caused XSS regressions in marked history.
marked.use({ gfm: true, breaks: true })

export function renderMarkdown(src: string | undefined | null): string {
  if (!src) return ""
  configurePurify()
  const safeSrc = src.length > MAX_DESCRIPTION_LEN ? src.slice(0, MAX_DESCRIPTION_LEN) : src
  const html = marked.parse(safeSrc, { async: false }) as string
  return DOMPurify.sanitize(html, {
    ALLOWED_TAGS: ALLOWED_TAGS as unknown as string[],
    ALLOWED_ATTR: ["href"],
    ALLOW_DATA_ATTR: false,
    ALLOW_UNKNOWN_PROTOCOLS: false,
    KEEP_CONTENT: true,
  })
}
