import { Card, Group, Pagination, Stack, Text, Title } from '@mantine/core'
import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { BlockButton } from '@/features/block-user/BlockButton'
import { useListBlocks } from '@/shared/api/social/endpoints'
import type { FollowUser } from '@/shared/api/social/model'
import { EmptyState, ErrorState, LoadingState } from '@/shared/ui/states'

const PAGE_SIZE = 20

/**
 * BlockedUsersList shows the users the current user has blocked (GET /me/blocks) with an
 * Unblock action per row, paginated so older blocks stay reachable. Unblocking invalidates
 * the blocked-list query (via BlockButton's useBlockMutations), so the row disappears on
 * success; the page clamps back if the last entry on a trailing page is removed.
 */
export function BlockedUsersList() {
  const [page, setPage] = useState(1)
  const query = useListBlocks({ page, page_size: PAGE_SIZE })
  const total = query.data?.total ?? 0
  const totalPages = Math.ceil(total / PAGE_SIZE)

  // Clamp back if unblocking emptied the current (trailing) page.
  useEffect(() => {
    if (query.isSuccess && page > 1 && (query.data?.users.length ?? 0) === 0) {
      setPage((p) => Math.max(1, p - 1))
    }
  }, [query.isSuccess, query.data, page])

  return (
    <Card withBorder padding="lg">
      <Stack gap="sm">
        <Title order={4}>Заблокированные</Title>
        {query.isLoading ? (
          <LoadingState />
        ) : query.isError ? (
          <ErrorState message="Не удалось загрузить список" />
        ) : total === 0 ? (
          <EmptyState title="Пусто" description="Вы никого не заблокировали." />
        ) : (
          <>
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
            {totalPages > 1 && (
              <Group justify="center">
                <Pagination total={totalPages} value={page} onChange={setPage} size="sm" />
              </Group>
            )}
          </>
        )}
      </Stack>
    </Card>
  )
}
