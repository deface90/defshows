import { Accordion, Center, Stack, Text } from '@mantine/core'
import { SeasonWatchToggle } from '@/features/mark-watched/SeasonWatchToggle'
import type { Season } from '@/shared/api/shows/model'
import { isAired } from './airDate'
import { EpisodeRow } from './EpisodeRow'

export function SeasonAccordion({
  seasons,
  showId,
  tracked,
  watchedIds,
}: {
  seasons: Season[]
  showId: number
  tracked: boolean
  watchedIds: Set<number>
}) {
  if (seasons.length === 0) {
    return <Text c="dimmed">Нет информации о сезонах</Text>
  }
  const sortedSeasons = [...seasons].sort((a, b) => b.season_number - a.season_number)
  return (
    <Accordion multiple defaultValue={[String(sortedSeasons[0].season_number)]}>
      {sortedSeasons.map((season) => {
        const airedEpisodes = season.episodes.filter((ep) => isAired(ep.air_date))
        const watchedCount = airedEpisodes.reduce(
          (n, ep) => (watchedIds.has(ep.id) ? n + 1 : n),
          0,
        )
        // "Fully watched" ignores episodes that have not aired yet.
        const airedCount = airedEpisodes.length
        const allWatched = airedCount > 0 && watchedCount >= airedCount
        return (
        <Accordion.Item key={season.id} value={String(season.season_number)}>
          <Center pr="md">
            <Accordion.Control>
              {season.name || `Сезон ${season.season_number}`}{' '}
              <Text span c="dimmed" size="sm">
                {tracked ? `${watchedCount} / ${airedCount}${allWatched ? ' ✓' : ''}` : `${season.episodes.length} эпизодов`}
              </Text>
            </Accordion.Control>
            {tracked && (
              <SeasonWatchToggle
                showId={showId}
                episodes={season.episodes}
                watchedIds={watchedIds}
              />
            )}
          </Center>
          <Accordion.Panel>
            <Stack gap={2}>
              {[...season.episodes].sort((a, b) => b.episode_number - a.episode_number).map((ep) => (
                <EpisodeRow
                  key={ep.id}
                  episode={ep}
                  showId={showId}
                  tracked={tracked}
                  watched={watchedIds.has(ep.id)}
                />
              ))}
            </Stack>
          </Accordion.Panel>
        </Accordion.Item>
        )
      })}
    </Accordion>
  )
}
