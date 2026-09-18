import { ActionIcon, Badge, Card, Group, Text } from '@mantine/core'
import type { NotificationItem as NotificationItemModel } from '@/shared/api/notifications/model'

const typeLabels: Record<string, string> = {
  episode_released: 'Новый эпизод',
  episode_upcoming: 'Скоро эпизод',
  season_upcoming: 'Скоро сезон',
}

interface NotificationItemProps {
  item: NotificationItemModel
  onMarkRead?: (id: number) => void
  markPending?: boolean
}

/** NotificationItem renders a single in-app notification from the feed. */
export function NotificationItem({ item, onMarkRead, markPending }: NotificationItemProps) {
  const label = typeLabels[item.type] ?? item.type
  return (
    <Card withBorder padding="sm" bg={item.read ? undefined : 'var(--mantine-color-default-hover)'}>
      <Group justify="space-between" align="flex-start" wrap="nowrap">
        <div>
          <Group gap="xs" mb={4}>
            <Badge variant={item.read ? 'light' : 'filled'}>{label}</Badge>
            {!item.read && (
              <Badge variant="outline" color="orange" size="xs">
                новое
              </Badge>
            )}
          </Group>
          <Text size="sm">{item.payload}</Text>
        </div>
        <Group gap="xs" wrap="nowrap">
          <Text c="dimmed" size="xs" style={{ whiteSpace: 'nowrap' }}>
            {new Date(item.created_at).toLocaleString()}
          </Text>
          {!item.read && onMarkRead && (
            <ActionIcon
              variant="subtle"
              aria-label="Отметить прочитанным"
              loading={markPending}
              onClick={() => onMarkRead(item.id)}
            >
              ✓
            </ActionIcon>
          )}
        </Group>
      </Group>
    </Card>
  )
}
