import { Box, Group, SimpleGrid, Stack, Text, Title } from '@mantine/core'
import { Link } from 'react-router-dom'
import { AddShowButton } from '@/features/add-show/AddShowButton'
import { Poster } from '@/entities/show/Poster'
import { useTrendingShows } from '@/shared/api/shows/endpoints'
import { LoadingState } from '@/shared/ui/states'

/**
 * TrendingShows is a compact "what's popular this week" row (TMDB weekly
 * trending). Used on the guest landing page and the dashboard home so there's
 * always real catalog content to look at, independent of personal tracking data.
 */
export function TrendingShows({ limit = 12 }: { limit?: number }) {
  const { data, isLoading, isError } = useTrendingShows()

  if (isLoading) return <LoadingState label="Загружаем тренды…" />
  if (isError || !data || data.results.length === 0) return null

  return (
    <Stack gap="sm">
      <Group justify="space-between" align="center">
        <Title order={3} fz="h4">
          В тренде
        </Title>
        <Text component={Link} to="/discover" c="brand" size="sm">
          Подбор сериала →
        </Text>
      </Group>
      <SimpleGrid cols={{ base: 2, xs: 3, sm: 4, md: 6 }} spacing="md">
        {data.results.slice(0, limit).map((show) => (
          <Stack key={show.tmdb_id} gap={6} align="center">
            <Box component={Link} to={`/catalog/${show.tmdb_id}`} aria-label={show.title} style={{ display: 'block' }}>
              <Poster src={show.poster_url || undefined} title={show.title} w={120} voteAverage={show.vote_average} voteCount={show.vote_count} />
            </Box>
            <Text size="sm" fw={500} ta="center" lineClamp={2} w={120}>
              {show.title}
            </Text>
            <AddShowButton tmdbId={show.tmdb_id} />
          </Stack>
        ))}
      </SimpleGrid>
    </Stack>
  )
}
