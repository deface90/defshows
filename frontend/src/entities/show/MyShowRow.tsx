import { Badge, Box, Group, Progress, Stack, Text } from '@mantine/core'
import { Link } from 'react-router-dom'
import { StatusSelect } from '@/features/show-status/StatusSelect'
import type { UserShowStatus } from '@/shared/api/tracking/model'
import { AiringStatusBadge } from './AiringStatusBadge'
import { Poster } from './Poster'
import { STATUS_LABELS } from './status'

/**
 * MyShowRowData is the structural subset of a tracked show the row renders.
 * Both the tracking and users API models satisfy it (their enums widen to
 * string), so the same row serves "My Shows" and read-only public profiles.
 */
export interface MyShowRowData {
  show: {
    id: number
    title: string
    original_title?: string
    poster_url?: string
    airing_status?: string
    next_episode_air_date?: string | null
    vote_average?: number
    vote_count?: number
  }
  user_show: { status: string }
  progress: { watched: number; total: number }
}

export function MyShowRow({
  tracked,
  readOnly = false,
}: {
  tracked: MyShowRowData
  readOnly?: boolean
}) {
  const { show, user_show, progress } = tracked
  const pct = progress.total > 0 ? Math.round((progress.watched / progress.total) * 100) : 0
  const hasOriginal = show.original_title && show.original_title !== show.title

  return (
    <Group wrap="nowrap" align="center" gap="xl" p="md">
      <Box
        component={Link}
        to={`/shows/${show.id}`}
        style={{ flex: 1, minWidth: 0, color: 'inherit', textDecoration: 'none' }}
      >
        <Group wrap="nowrap" gap="md" align="center">
          <Poster
            src={show.poster_url || undefined}
            title={show.title}
            w={64}
            voteAverage={show.vote_average}
            voteCount={show.vote_count}
          />
          <Stack gap={6} style={{ minWidth: 0, flex: 1 }}>
            <Group gap={8} align="baseline" wrap="nowrap" style={{ minWidth: 0 }}>
              <Text fw={600} lh={1.2} truncate style={{ minWidth: 0 }}>
                {show.title}
              </Text>
              {hasOriginal && (
                <Text c="dimmed" size="sm" truncate>
                  {show.original_title}
                </Text>
              )}
            </Group>

            <Group gap="xs" wrap="nowrap" align="center" style={{ minWidth: 0 }}>
              {show.airing_status && <AiringStatusBadge status={show.airing_status} />}
              <Text c="dimmed" size="sm" truncate>
                {progress.watched} / {progress.total} эпизодов
                {show.next_episode_air_date ? ` · следующий ${show.next_episode_air_date}` : ''}
              </Text>
            </Group>

            <Group gap="sm" wrap="nowrap" align="center">
              <Progress value={pct} size="sm" radius="xl" w={280} maw="100%" style={{ flex: 1 }} />
              <Text fz="xs" c="dimmed" ta="right" w={38} style={{ fontVariantNumeric: 'tabular-nums' }}>
                {pct}%
              </Text>
            </Group>
          </Stack>
        </Group>
      </Box>

      {readOnly ? (
        <Badge variant="light" color="brand" w={150} style={{ flexShrink: 0 }}>
          {STATUS_LABELS[user_show.status] ?? user_show.status}
        </Badge>
      ) : (
        <StatusSelect showId={show.id} status={user_show.status as UserShowStatus} />
      )}
    </Group>
  )
}
