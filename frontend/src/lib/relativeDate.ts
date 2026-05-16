// Swedish short-form relative date helpers. Used by the Suggestions tab.

const MINUTE = 60
const HOUR = 60 * MINUTE
const DAY = 24 * HOUR
const WEEK = 7 * DAY
const MONTH = 30 * DAY
const YEAR = 365 * DAY

/**
 * Format an ISO timestamp as a compact Swedish relative-time label.
 * Examples: "nu", "5m", "3h", "2d", "3v", "4mån", "2år".
 *
 * The "now" reference can be injected for deterministic tests.
 */
export function formatShortRelative(
  iso: string,
  now: Date = new Date()
): string {
  const then = new Date(iso)
  if (Number.isNaN(then.getTime())) return ""
  const diffSec = Math.max(0, Math.floor((now.getTime() - then.getTime()) / 1000))

  if (diffSec < MINUTE) return "nu"
  if (diffSec < HOUR) return `${Math.floor(diffSec / MINUTE)}m`
  if (diffSec < DAY) return `${Math.floor(diffSec / HOUR)}h`
  if (diffSec < 14 * DAY) return `${Math.floor(diffSec / DAY)}d`
  if (diffSec < 8 * WEEK) return `${Math.floor(diffSec / WEEK)}v`
  if (diffSec < YEAR) return `${Math.floor(diffSec / MONTH)}mån`
  return `${Math.floor(diffSec / YEAR)}år`
}
