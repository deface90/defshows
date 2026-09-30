/** Formats a minute count as a compact "Xд Yч" / "Yч Zмин" watch-time string. */
export function formatWatchTime(minutes: number): string {
  if (minutes <= 0) return '0 мин'
  const days = Math.floor(minutes / (60 * 24))
  const hours = Math.floor((minutes % (60 * 24)) / 60)
  const mins = minutes % 60
  const parts: string[] = []
  if (days > 0) parts.push(`${days} д`)
  if (hours > 0) parts.push(`${hours} ч`)
  if (mins > 0 && days === 0) parts.push(`${mins} мин`)
  return parts.join(' ')
}
