/**
 * Extracts the item part from ingredient input for autocomplete.
 * Strips leading quantity+unit so "2l Mj" -> "Mj", "ca 2 dl mj" -> "mj".
 * Uses the same Swedish unit lexicon as the server (backend/data/swedish_units.json).
 */

const MODIFIERS = [
  'ca',
  'cirka',
  'typ',
  'ungefär',
  'lite',
  'halv',
  'en',
  'ett',
]

// Units from swedish_units.json - sorted by length descending for greedy matching
const UNITS = [
  'deciliter',
  'centiliter',
  'milliliter',
  'kilogram',
  'hektogram',
  'matskedar',
  'teskedar',
  'kryddmått',
  'förpackningar',
  'förpackning',
  'liter',
  'matsked',
  'tesked',
  'burkar',
  'flaskor',
  'kartonger',
  'påsar',
  'rullar',
  'stycken',
  'kilo',
  'gram',
  'hekto',
  'förp',
  'förpk',
  'burk',
  'paket',
  'flaska',
  'kartong',
  'påse',
  'rulle',
  'styck',
  'dl',
  'cl',
  'ml',
  'msk',
  'tsk',
  'krm',
  'kg',
  'g',
  'hg',
  'l',
  'st',
]

function matchModifier(input: string): string | null {
  const lower = input.toLowerCase().trimStart()
  for (const m of MODIFIERS) {
    if (lower === m || lower.startsWith(m + ' ')) {
      return m
    }
  }
  return null
}

function matchNumber(input: string): string | null {
  const trimmed = input.trimStart()
  const match = trimmed.match(/^(\d+[.,]?\d*)\s*/)
  return match ? match[1] : null
}

function matchUnit(input: string): string | null {
  const lower = input.toLowerCase().trimStart()
  for (const u of UNITS) {
    if (lower === u || lower.startsWith(u + ' ') || lower.startsWith(u + '\t')) {
      return u
    }
  }
  return null
}

/**
 * Strips leading modifier + number + unit from ingredient input.
 * Returns the remaining text (the item name part) for autocomplete.
 * E.g. "2l Mj" -> "Mj", "ca 2 dl mjölk" -> "mjölk", "1 burk tomater" -> "tomater"
 * Only strips modifier when followed by number or unit (e.g. "lite mjöl" stays "lite mjöl").
 */
export function extractAutocompleteQuery(fullInput: string): string {
  if (!fullInput || !fullInput.trim()) return fullInput

  let rest = fullInput.trimStart()

  // Optional modifier - only strip if followed by number or unit (e.g. "lite mjöl" stays as-is)
  const mod = matchModifier(rest)
  if (mod) {
    const afterMod = rest.slice(mod.length).trimStart()
    const num = matchNumber(afterMod)
    const afterNum = num ? afterMod.slice(num.length).trimStart() : afterMod
    if (num || matchUnit(afterNum)) {
      rest = afterMod
    }
  }

  // Optional number
  const numMatch = matchNumber(rest)
  if (numMatch) {
    rest = rest.slice(numMatch.length).trimStart()
  }

  // Optional unit
  const unit = matchUnit(rest)
  if (unit) {
    rest = rest.slice(unit.length).trimStart()
  }

  return rest
}
