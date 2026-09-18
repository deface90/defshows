import { Accordion, Center, Stack, Text } from '@mantine/core'
import { SeasonWatchToggle } from '@/features/mark-watched/SeasonWatchToggle'
import type { Season } from '@/shared/api/shows/model'
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
  return (
    <Accordion multiple defaultValue={[String(seasons[0].season_number)]}>
      {seasons.map((season) => {
        const watchedCount = season.episodes.reduce(
          (n, ep) => (watchedIds.has(ep.id) ? n + 1 : n),
          0,
        )
        const allWatched = season.episodes.length > 0 && watchedCount === season.episodes.length
        return (
        <Accordion.Item key={season.id} value={String(season.season_number)}>
          <Center pr="md">
            <Accordion.Control>
              {season.name || `Сезон ${season.season_number}`}{' '}
              <Text span c="dimmed" size="sm">
                {tracked ? `${watchedCount} / ${season.episodes.length}${allWatched ? ' ✓' : ''}` : `${season.episodes.length} эпизодов`}
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
              {season.episodes.map((ep) => (
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
