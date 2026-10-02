import { Badge, Button, Card, Group, Stack, Tabs, Text, Title } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { useSearchParams } from 'react-router-dom'
import {
  getListFollowersQueryKey,
  removeFollower,
  useListFollowers,
  useListFollowing,
  useListIncomingRequests,
} from '@/shared/api/social/endpoints'
import type { FollowUser } from '@/shared/api/social/model'
import { useAuthStore } from '@/shared/auth/authStore'
import { useDocumentTitle } from '@/shared/lib/useDocumentTitle'
import { EmptyState, ErrorState, LoadingState } from '@/shared/ui/states'
import { FollowRequests } from '@/features/follow/FollowRequests'

const TABS = ['followers', 'following', 'requests'] as const
type Tab = (typeof TABS)[number]

/** FollowsPage shows the current user's followers, who they follow, and incoming requests. */
export function FollowsPage() {
  useDocumentTitle('Подписки')
  const me = useAuthStore((s) => s.user)
  const [params, setParams] = useSearchParams()
  const raw = params.get('tab')
  const tab: Tab = TABS.includes(raw as Tab) ? (raw as Tab) : 'followers'

  const incoming = useListIncomingRequests()
  const pendingCount = incoming.data?.users.length ?? 0

  if (!me) return null

  return (
    <Stack gap="lg">
      <Title order={2}>Подписки</Title>
      <Tabs value={tab} onChange={(v) => v && setParams({ tab: v })}>
        <Tabs.List>
          <Tabs.Tab value="followers">Мои подписчики</Tabs.Tab>
          <Tabs.Tab value="following">Я подписан</Tabs.Tab>
          <Tabs.Tab
            value="requests"
            rightSection={
              pendingCount > 0 ? (
                <Badge size="sm" circle color="orange">
                  {pendingCount}
                </Badge>
              ) : undefined
            }
          >
            Запросы
          </Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="followers" pt="md">
          <FollowList kind="followers" userId={me.id} emptyText="У вас пока нет подписчиков." />
        </Tabs.Panel>
        <Tabs.Panel value="following" pt="md">
          <FollowList kind="following" userId={me.id} emptyText="Вы пока ни на кого не подписаны." />
        </Tabs.Panel>
        <Tabs.Panel value="requests" pt="md">
          <FollowRequests />
        </Tabs.Panel>
      </Tabs>
    </Stack>
  )
}

function FollowList({
  kind,
  userId,
  emptyText,
}: {
  kind: 'followers' | 'following'
  userId: number
  emptyText: string
}) {
  const followers = useListFollowers(userId, undefined, { query: { enabled: kind === 'followers' } })
  const following = useListFollowing(userId, undefined, { query: { enabled: kind === 'following' } })
  const query = kind === 'followers' ? followers : following

  if (query.isLoading) return <LoadingState />
  if (query.isError) return <ErrorState message="Не удалось загрузить список" />
  const users = query.data?.users ?? []
  if (users.length === 0) return <EmptyState title="Пусто" description={emptyText} />

  return (
    <Stack gap="xs">
      {users.map((u: FollowUser) => (
        <Card key={u.id} withBorder padding="sm">
          <Group justify="space-between" wrap="nowrap">
            <Text
              component={Link}
              to={`/users/${u.id}`}
              fw={600}
              truncate
              style={{ color: 'inherit', textDecoration: 'none', flex: 1, minWidth: 0 }}
            >
              {u.display_name}
            </Text>
            <Group gap="sm" wrap="nowrap">
              {!u.is_public && (
                <Text c="dimmed" size="sm">
                  Закрытый
                </Text>
              )}
              {kind === 'followers' && <RemoveFollowerButton ownerId={userId} followerId={u.id} />}
            </Group>
          </Group>
        </Card>
      ))}
    </Stack>
  )
}

/** RemoveFollowerButton ejects an accepted follower via DELETE /me/followers/{id}. */
function RemoveFollowerButton({ ownerId, followerId }: { ownerId: number; followerId: number }) {
  const queryClient = useQueryClient()
  const mutation = useMutation({
    mutationFn: () => removeFollower(followerId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: getListFollowersQueryKey(ownerId) })
      notifications.show({ message: 'Подписчик удалён' })
    },
    onError: () => notifications.show({ message: 'Не удалось удалить подписчика', color: 'red' }),
  })
  return (
    <Button
      size="xs"
      variant="light"
      color="red"
      loading={mutation.isPending}
      onClick={() => mutation.mutate()}
    >
      Убрать
    </Button>
  )
}
