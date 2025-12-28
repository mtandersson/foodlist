/**
 * Smart splitting: matches known items first, then splits remaining by spaces
 * This handles items with spaces by prioritizing multi-word known items
 */
export function smartSplit(text: string, knownItems: string[]): string[] {
  if (!text) return []

  const result: string[] = []
  let remaining = text.trim()

  // Sort known items by length (longest first) to match "mjölk sirap" before "mjölk"
  const sortedItems = [...knownItems].sort((a, b) => b.length - a.length)

  // Try to match known items greedily from the start
  while (remaining.length > 0) {
    let matched = false
    const remainingLower = remaining.toLowerCase()

    // Try each known item (longest first) to find a match at the start
    for (const item of sortedItems) {
      const itemLower = item.toLowerCase()

      // Check if remaining text starts with this item (case-insensitive)
      if (remainingLower.startsWith(itemLower)) {
        // Check if it's a complete word match (ends at space or end of string)
        const matchEnd = item.length
        if (matchEnd >= remaining.length || remaining[matchEnd] === " ") {
          // Match found! Extract it (preserve original case from remaining)
          const matchedText = remaining.substring(0, matchEnd)
          result.push(matchedText.trim())
          remaining = remaining.substring(matchEnd).trim()
          matched = true
          break
        }
      }
    }

    // If no match found, extract one word
    if (!matched) {
      const spaceIndex = remaining.indexOf(" ")
      if (spaceIndex > 0) {
        // Extract first word
        result.push(remaining.substring(0, spaceIndex))
        remaining = remaining.substring(spaceIndex + 1).trim()
      } else {
        // Last word
        if (remaining.length > 0) {
          result.push(remaining)
        }
        break
      }
    }
  }

  return result.filter((item) => item.length > 0)
}

/**
 * Extract queries from voice input (removes "handla" prefix)
 */
export function extractVoiceInput(text: string): string {
  if (!text) return ""

  // Trim input first
  const trimmed = text.trim()
  if (!trimmed) return ""

  // Remove common prefixes like "handla", "köp", etc.
  const patterns = [
    /^handla\s+(.+)$/i, // "handla mjölk sirap potatis" -> "mjölk sirap potatis"
    /^köp\s+(.+)$/i, // "köp mjölk sirap" -> "mjölk sirap"
    /^lägg\s+till\s+(.+)$/i, // "lägg till mjölk" -> "mjölk"
    /^add\s+(.+)$/i, // "add milk" -> "milk"
    /^buy\s+(.+)$/i, // "buy milk" -> "milk"
  ]

  for (const pattern of patterns) {
    const match = trimmed.match(pattern)
    if (match && match[1]) {
      return match[1].trim()
    }
  }

  // If no pattern matches, return the text as-is
  return trimmed
}
