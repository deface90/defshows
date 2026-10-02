import { Button, Card, Group, Stack, Text } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import {
  approveFollower,
  getListIncomingRequestsQueryKey,
  rejectFollower,
  useListIncomingRequests,
} from '@/shared/api/social/endpoints'
import { EmptyState, ErrorState, LoadingState } from '@/shared/ui/states'

/** FollowRequests lists pending incoming follow requests with approve/reject. */
export function FollowRequests() {
  const query = useListIncomingRequests()
  const queryClient = useQueryClient()
  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: getListIncomingRequestsQueryKey() })

  const approve = useMutation({
    mutationFn: (userId: number) => approveFollower(userId),
    onSuccess: invalidate,
    onError: () => notifications.show({ message: 'Не удалось принять запрос', color: 'red' }),
  })
  const reject = useMutation({
    mutationFn: (userId: number) => rejectFollower(userId),
    onSuccess: invalidate,
    onError: () => notifications.show({ message: 'Не удалось отклонить запрос', color: 'red' }),
  })
  const busy = approve.isPending || reject.isPending

  if (query.isLoading) return <LoadingState />
  if (query.isError) return <ErrorState message="Не удалось загрузить запросы" />
  const users = query.data?.users ?? []
  if (users.length === 0) return <EmptyState title="Нет новых запросов" />

  return (
    <Stack gap="xs">
      {users.map((u) => (
        <Card key={u.id} withBorder padding="sm">
          <Group justify="space-between" wrap="nowrap">
            <Text component={Link} to={`/users/${u.id}`} fw={600} truncate>
              {u.display_name}
            </Text>
            <Group gap="xs" wrap="nowrap">
              <Button size="xs" loading={busy} onClick={() => approve.mutate(u.id)}>
                Принять
              </Button>
              <Button size="xs" variant="default" loading={busy} onClick={() => reject.mutate(u.id)}>
                Отклонить
              </Button>
            </Group>
          </Group>
        </Card>
      ))}
    </Stack>
  )
}
