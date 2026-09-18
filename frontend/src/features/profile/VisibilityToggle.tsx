import { Card, Group, Stack, Switch, Text, Title } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
  getGetSettingsQueryKey,
  updateSettings,
  useGetSettings,
} from '@/shared/api/tracking/endpoints'
import { ErrorState, LoadingState } from '@/shared/ui/states'

/**
 * VisibilityToggle flips the profile between public and private. Public profiles
 * are visible to other users on the /users directory; private ones are hidden.
 */
export function VisibilityToggle() {
  const queryClient = useQueryClient()
  const settingsQuery = useGetSettings()

  const mutation = useMutation({
    mutationFn: (isPublic: boolean) =>
      updateSettings({ timezone: settingsQuery.data?.timezone ?? 'UTC', is_public: isPublic }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: getGetSettingsQueryKey() })
      notifications.show({ message: 'Видимость профиля обновлена', color: 'green' })
    },
    onError: () =>
      notifications.show({ message: 'Не удалось обновить видимость', color: 'red' }),
  })

  if (settingsQuery.isLoading) return <LoadingState />
  if (settingsQuery.isError || !settingsQuery.data)
    return <ErrorState message="Не удалось загрузить настройки профиля" />

  const isPublic = settingsQuery.data.is_public

  return (
    <Card withBorder padding="lg">
      <Group justify="space-between" align="flex-start" wrap="nowrap">
        <Stack gap={4}>
          <Title order={4}>Профиль</Title>
          <Text c="dimmed" size="sm">
            {isPublic
              ? 'Публичный: другие пользователи видят ваши сериалы и прогресс.'
              : 'Приватный: ваш список сериалов скрыт от других пользователей.'}
          </Text>
        </Stack>
        <Switch
          checked={isPublic}
          onChange={(e) => mutation.mutate(e.currentTarget.checked)}
          disabled={mutation.isPending}
          label="Публичный"
          aria-label="Публичный профиль"
        />
      </Group>
    </Card>
  )
}
