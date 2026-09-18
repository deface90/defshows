import { Box, Image } from '@mantine/core'
import { RatingBadge } from './RatingBadge'

function initials(title: string): string {
  return title
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((w) => w[0]?.toUpperCase() ?? '')
    .join('')
}

/**
 * Poster renders a 2:3 cover image when `src` is present, otherwise a neutral
 * gradient placeholder with the show's initials (matches the prototype look and
 * stays theme-agnostic — it reads fine on both the dark and light schemes).
 */
export function Poster({
  src,
  title,
  w,
  voteAverage,
  voteCount,
}: {
  src?: string
  title: string
  w: number
  voteAverage?: number | null
  voteCount?: number | null
}) {
  const cover = src ? (
    <Image
      src={src}
      w={w}
      radius="md"
      fit="cover"
      style={{ aspectRatio: '2 / 3', display: 'block' }}
      alt={title}
    />
  ) : (
    <Box
      w={w}
      aria-hidden
      style={{
        aspectRatio: '2 / 3',
        borderRadius: 'var(--mantine-radius-md)',
        display: 'grid',
        placeItems: 'center',
        background:
          'linear-gradient(145deg, rgba(255,255,255,0.12), transparent 45%), linear-gradient(160deg, #29334e, #6c78a8)',
        color: 'rgba(255,255,255,0.92)',
        fontWeight: 800,
        letterSpacing: '0.08em',
        fontSize: w >= 120 ? 30 : 18,
        boxShadow: 'inset 0 0 0 1px rgba(255,255,255,0.1)',
      }}
    >
      {initials(title)}
    </Box>
  )

  return (
    <Box w={w} style={{ position: 'relative', flex: 'none' }}>
      {cover}
      <Box style={{ position: 'absolute', top: 6, right: 6 }}>
        <RatingBadge average={voteAverage} votes={voteCount} size={w >= 120 ? 'sm' : 'xs'} />
      </Box>
    </Box>
  )
}
