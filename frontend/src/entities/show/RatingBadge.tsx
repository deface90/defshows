import { Badge, type MantineSize, Tooltip } from '@mantine/core'

function formatVotes(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${Math.round(n / 1_000)}k`
  return String(n)
}

/**
 * RatingBadge shows the TMDB vote average as `★ 8.4` with the vote count in a
 * tooltip. It renders nothing when there is no meaningful rating (no votes yet),
 * and uses a dark translucent chip so it stays readable overlaid on any poster.
 */
export function RatingBadge({
  average,
  votes,
  size = 'sm',
}: {
  average?: number | null
  votes?: number | null
  size?: MantineSize
}) {
  if (average == null || average <= 0) return null
  return (
    <Tooltip label={`${formatVotes(votes ?? 0)} голосов на TMDB`} withArrow disabled={!votes}>
      <Badge
        size={size}
        radius="sm"
        leftSection="★"
        aria-label={`Рейтинг TMDB ${average.toFixed(1)}`}
        styles={{
          root: { backgroundColor: 'rgba(0, 0, 0, 0.72)', color: '#ffd43b' },
          label: { fontWeight: 700 },
        }}
      >
        {average.toFixed(1)}
      </Badge>
    </Tooltip>
  )
}
