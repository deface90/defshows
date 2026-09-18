import { Button, Group, Stack, Title } from '@mantine/core'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { NotificationItem } from '@/entities/notification/NotificationItem'
import {
  getListNotificationsQueryKey,
  markAllNotificationsRead,
  markNotificationRead,
  useListNotifications,
} from '@/shared/api/notifications/endpoints'
import { useDocumentTitle } from '@/shared/lib/useDocumentTitle'
import { EmptyState, ErrorState, LoadingState } from '@/shared/ui/states'

/** NotificationsPage renders the in-app notification feed with read controls. */
export function NotificationsPage() {
  useDocumentTitle('Уведомления')
  const queryClient = useQueryClient()
  const query = useListNotifications()
  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: getListNotificationsQueryKey() })

  const markOne = useMutation({
    mutationFn: (id: number) => markNotificationRead(id),
    onSuccess: invalidate,
  })
  const markAll = useMutation({
    mutationFn: () => markAllNotificationsRead(),
    onSuccess: invalidate,
  })

  const items = query.data?.notifications ?? []
  const hasUnread = items.some((n) => !n.read)

  return (
    <Stack>
      <Group justify="space-between">
        <Title order={3}>Уведомления</Title>
        {hasUnread && (
          <Button variant="subtle" size="xs" loading={markAll.isPending} onClick={() => markAll.mutate()}>
            Прочитать всё
          </Button>
        )}
      </Group>
      {query.isLoading ? (
        <LoadingState />
      ) : query.isError ? (
        <ErrorState message="Не удалось загрузить уведомления" />
      ) : !items.length ? (
        <EmptyState title="Уведомлений нет" description="Здесь появятся напоминания о новых эпизодах." />
      ) : (
        items.map((item) => (
          <NotificationItem
            key={item.id}
            item={item}
            onMarkRead={(id) => markOne.mutate(id)}
            markPending={markOne.isPending && markOne.variables === item.id}
          />
        ))
      )}
    </Stack>
  )
}
