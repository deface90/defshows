import { Badge, Group } from '@mantine/core'
import type { Rating } from '@/shared/api/shows/model'

const labels: Record<string, string> = {
  imdb: 'IMDb',
  rotten_tomatoes: 'RT',
  metacritic: 'Metacritic',
  kinopoisk: 'Кинопоиск',
  tmdb: 'TMDB',
}

export function RatingBadges({ ratings }: { ratings?: Rating[] }) {
  if (!ratings || ratings.length === 0) return null
  return (
    <Group gap="xs">
      {ratings.map((r) => (
        <Badge key={r.source} variant="light" color="brand">
          {(labels[r.source] ?? r.source) + ': ' + r.value}
        </Badge>
      ))}
    </Group>
  )
}
