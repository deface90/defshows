import { Card, Group, Stack, Text, Title } from '@mantine/core'
import { Link } from 'react-router-dom'
import { BlockButton } from '@/features/block-user/BlockButton'
import { useListBlocks } from '@/shared/api/social/endpoints'
import type { FollowUser } from '@/shared/api/social/model'
import { EmptyState, ErrorState, LoadingState } from '@/shared/ui/states'

/**
 * BlockedUsersList shows the users the current user has blocked (GET /me/blocks) with an
 * Unblock action per row. Unblocking invalidates the blocked-list query (via BlockButton's
 * useBlockMutations), so the row disappears on success.
 */
export function BlockedUsersList() {
  const query = useListBlocks()

  return (
    <Card withBorder padding="lg">
      <Stack gap="sm">
        <Title order={4}>Заблокированные</Title>
        {query.isLoading ? (
          <LoadingState />
        ) : query.isError ? (
          <ErrorState message="Не удалось загрузить список" />
        ) : (query.data?.users.length ?? 0) === 0 ? (
          <EmptyState title="Пусто" description="Вы никого не заблокировали." />
        ) : (
          <Stack gap="xs">
            {query.data?.users.map((u: FollowUser) => (
              <Group key={u.id} justify="space-between" wrap="nowrap">
                <Text
                  component={Link}
                  to={`/users/${u.id}`}
                  fw={600}
                  truncate
                  style={{ color: 'inherit', textDecoration: 'none' }}
                >
                  {u.display_name}
                </Text>
                <BlockButton userId={u.id} />
              </Group>
            ))}
          </Stack>
        )}
      </Stack>
    </Card>
  )
}
