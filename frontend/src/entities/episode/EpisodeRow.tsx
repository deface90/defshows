import { Box, Group, Text, Tooltip } from '@mantine/core'
import { WatchToggle } from '@/features/mark-watched/WatchToggle'
import type { Episode } from '@/shared/api/shows/model'
import { formatAirDate, isAired } from './airDate'
import { episodeCode } from './episodeCode'

export function EpisodeRow({
  episode,
  showId,
  tracked,
  watched,
}: {
  episode: Episode
  showId: number
  tracked: boolean
  watched: boolean
}) {
  const aired = isAired(episode.air_date)
  const airLabel = formatAirDate(episode.air_date)

  return (
    <Group wrap="nowrap" gap="sm" align="center" py={8}>
      {tracked ? (
        <Tooltip label="Эпизод ещё не вышел" disabled={aired} withArrow>
          {/* Wrapper span keeps the tooltip working over a disabled checkbox. */}
          <Box style={{ flex: 'none', display: 'flex' }}>
            <WatchToggle
              showId={showId}
              episodeId={episode.id}
              watched={watched}
              disabled={!aired}
            />
          </Box>
        </Tooltip>
      ) : (
        <Box w={20} style={{ flex: 'none' }} />
      )}
      <Text
        size="sm"
        c="dimmed"
        w={54}
        style={{ flex: 'none', fontVariantNumeric: 'tabular-nums' }}
      >
        {episodeCode(episode.season_number, episode.episode_number)}
      </Text>
      <Text size="sm" truncate style={{ flex: 1, minWidth: 0 }} c={watched ? 'dimmed' : undefined}>
        {episode.name}
      </Text>
      {airLabel ? (
        <Text size="xs" c={aired ? 'dimmed' : 'brand'} style={{ flex: 'none' }}>
          {aired ? airLabel : `📅 ${airLabel}`}
        </Text>
      ) : null}
    </Group>
  )
}
