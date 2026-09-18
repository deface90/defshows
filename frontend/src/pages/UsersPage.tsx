import { Badge, Box, Card, Group, Stack, Text, Title } from '@mantine/core'
import { Link } from 'react-router-dom'
import { useListUsers } from '@/shared/api/users/endpoints'
import { useAuthStore } from '@/shared/auth/authStore'
import { useDocumentTitle } from '@/shared/lib/useDocumentTitle'
import { EmptyState, ErrorState, LoadingState } from '@/shared/ui/states'

/** UsersPage lists all users with a link into each public/private profile. */
export function UsersPage() {
  useDocumentTitle('Пользователи')
  const query = useListUsers()
  const me = useAuthStore((s) => s.user)
  const users = query.data?.users ?? []

  return (
    <Stack gap="lg">
      <div>
        <Title order={2}>Пользователи</Title>
        <Text c="dimmed" mt={4}>
          Загляните в чужие коллекции сериалов
        </Text>
      </div>

      <Card withBorder padding={0}>
        {query.isLoading && <LoadingState />}
        {query.isError && <ErrorState message="Не удалось загрузить список пользователей" />}
        {query.isSuccess && users.length === 0 && <EmptyState title="Пока никого нет" />}
        {users.map((u, i) => (
          <Box
            key={u.id}
            component={Link}
            to={`/users/${u.id}`}
            style={{
              display: 'block',
              color: 'inherit',
              textDecoration: 'none',
              borderTop: i > 0 ? '1px solid var(--mantine-color-default-border)' : undefined,
            }}
          >
            <Group wrap="nowrap" justify="space-between" p="md">
              <Stack gap={2} style={{ minWidth: 0 }}>
                <Group gap={8} wrap="nowrap">
                  <Text fw={600} truncate>
                    {u.display_name}
                  </Text>
                  {me?.id === u.id && (
                    <Text span c="dimmed" size="sm">
                      (вы)
                    </Text>
                  )}
                </Group>
                <Text c="dimmed" size="sm">
                  {u.shows_count} сериалов
                </Text>
              </Stack>
              <Badge variant="light" color={u.is_public ? 'brand' : 'gray'} style={{ flexShrink: 0 }}>
                {u.is_public ? 'Публичный' : 'Приватный'}
              </Badge>
            </Group>
          </Box>
        ))}
      </Card>
    </Stack>
  )
}
