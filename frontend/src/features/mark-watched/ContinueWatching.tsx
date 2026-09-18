import { Button, Card, Group, Stack, Text } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { episodeCode } from '@/entities/episode/episodeCode'
import type { Episode } from '@/shared/api/shows/model'
import { getGetTrackedQueryKey, watchEpisode } from '@/shared/api/tracking/endpoints'

/**
 * ContinueWatching is the "next episode to watch" card. `next` is the first
 * unwatched episode (or null when everything available has been seen).
 */
export function ContinueWatching({ showId, next }: { showId: number; next: Episode | null }) {
  const queryClient = useQueryClient()
  const done = next == null

  const mark = useMutation({
    mutationFn: () => watchEpisode(showId, next!.id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: getGetTrackedQueryKey(showId) }),
    onError: () => notifications.show({ message: 'Не удалось отметить', color: 'red' }),
  })

  return (
    <Card withBorder padding="lg">
      <Group justify="space-between" align="center" wrap="wrap" gap="md">
        <Stack gap={2} style={{ minWidth: 0 }}>
          <Text tt="uppercase" fw={800} fz={11} c="dimmed" style={{ letterSpacing: '0.09em' }}>
            Продолжить просмотр
          </Text>
          <Text fw={700} fz="lg">
            {done
              ? 'Все эпизоды просмотрены'
              : `${episodeCode(next.season_number, next.episode_number)} · ${next.name}`}
          </Text>
          {!done && next.runtime ? (
            <Text c="dimmed" size="sm">
              {next.runtime} мин
            </Text>
          ) : null}
        </Stack>
        <Button disabled={done} loading={mark.isPending} onClick={() => mark.mutate()}>
          {done ? 'Всё просмотрено' : '✓ Отметить просмотренным'}
        </Button>
      </Group>
    </Card>
  )
}
