import { Box, Group, Text } from '@mantine/core'
import { WatchToggle } from '@/features/mark-watched/WatchToggle'
import type { Episode } from '@/shared/api/shows/model'
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
  return (
    <Group wrap="nowrap" gap="sm" align="center" py={8}>
      {tracked ? (
        <WatchToggle showId={showId} episodeId={episode.id} watched={watched} />
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
      {episode.runtime ? (
        <Text size="xs" c="dimmed" style={{ flex: 'none' }}>
          {episode.runtime} мин
        </Text>
      ) : null}
    </Group>
  )
}
