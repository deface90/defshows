import { Badge, Button, Group, Stack, Text } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation } from '@tanstack/react-query'
import { getTelegramLink, useGetTelegramStatus } from '@/shared/api/notifications/endpoints'

/**
 * LinkTelegramButton shows the current link status and fetches a one-time Telegram
 * deep link from the backend to open. The bot name / URL are constructed server-side.
 */
export function LinkTelegramButton() {
  const statusQuery = useGetTelegramStatus()
  const linked = statusQuery.data?.linked ?? false

  const mutation = useMutation({
    mutationFn: () => getTelegramLink(),
    onSuccess: (res) => {
      window.open(res.url, '_blank', 'noopener,noreferrer')
    },
    onError: () => notifications.show({ message: 'Не удалось получить ссылку', color: 'red' }),
  })

  return (
    <Stack gap="xs">
      <Group gap="xs">
        <Text fw={600}>Telegram</Text>
        {statusQuery.isSuccess &&
          (linked ? (
            <Badge color="green" variant="light">
              привязан
            </Badge>
          ) : (
            <Badge color="gray" variant="light">
              не привязан
            </Badge>
          ))}
      </Group>
      <Text size="sm" c="dimmed">
        {linked
          ? 'Аккаунт привязан. Можно перепривязать другой чат по ссылке.'
          : 'Привяжите Telegram, чтобы получать уведомления в чате.'}
      </Text>
      <Button variant="light" loading={mutation.isPending} onClick={() => mutation.mutate()} maw={260}>
        {linked ? 'Перепривязать Telegram' : 'Привязать Telegram'}
      </Button>
    </Stack>
  )
}
