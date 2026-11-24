/**
 * Stable per-category colour slot (0-7), resolved to --cat-N tokens in app.css
 * so light and dark themes each get a readable shade.
 */
export const CATEGORY_COLOR_SLOTS = 8

const KNOWN: Record<string, number> = {
  'frukt & grönt': 0,
  mejeri: 1,
  rengöring: 2,
  torrvaror: 3,
  'asiatiskt & taco': 4,
  kött: 5,
  'läsk & öl': 6,
  dryck: 7,
}

export function categoryColorSlot(name: string): number {
  // Drop emoji and stray symbols so "Frukt & Grönt 🍅🥬" matches "frukt & grönt".
  const key = name
    .toLowerCase()
    .replace(/[^\p{L}\p{N}&\s]/gu, '')
    .replace(/\s+/g, ' ')
    .trim()
  if (key in KNOWN) return KNOWN[key]
  let h = 0
  for (const ch of key) h = (h * 31 + ch.codePointAt(0)!) >>> 0
  return h % CATEGORY_COLOR_SLOTS
}

export function categoryColorVar(name: string): string {
  return `var(--cat-${categoryColorSlot(name)})`
}
