/** Formats a season/episode pair as the canonical `S02E07` code. */
export function episodeCode(seasonNumber: number, episodeNumber: number): string {
  const s = String(seasonNumber).padStart(2, '0')
  const e = String(episodeNumber).padStart(2, '0')
  return `S${s}E${e}`
}
