/**
 * Air-date helpers shared by the episode list and watch toggles. An episode
 * counts as "aired" only when it has a known date that is today or earlier;
 * unknown (null) dates are treated as not-yet-aired so we never mark a
 * placeholder episode watched.
 */

/** isAired reports whether an episode air date is known and in the past. */
export function isAired(airDate?: string | null): boolean {
  if (!airDate) return false
  const t = Date.parse(airDate)
  if (Number.isNaN(t)) return false
  // Compare by day: an episode airing "today" is considered aired.
  return t <= Date.now()
}

const fmt = new Intl.DateTimeFormat('ru', { day: 'numeric', month: 'short', year: 'numeric' })

/** formatAirDate renders an ISO date as e.g. "7 июл. 2024 г."; "" when unknown. */
export function formatAirDate(airDate?: string | null): string {
  if (!airDate) return ''
  const t = Date.parse(airDate)
  if (Number.isNaN(t)) return ''
  return fmt.format(new Date(t))
}
