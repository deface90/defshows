import { Anchor, Button, Card, Divider, Group, Stack, Tabs, Text, Title } from '@mantine/core'
import { useCallback } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router-dom'
import { ActivityFeed } from '@/entities/activity/ActivityFeed'
import { MyShowRow } from '@/entities/show/MyShowRow'
import { AddShowButton } from '@/features/add-show/AddShowButton'
import { FollowButton } from '@/features/follow/FollowButton'
import { ModerationMenu } from '@/features/report-user/ModerationMenu'
import { getProfileFeed } from '@/shared/api/social/endpoints'
import { useListTracked } from '@/shared/api/tracking/endpoints'
import { useGetUserProfile, useListUserShows } from '@/shared/api/users/endpoints'
import { useAuthStore } from '@/shared/auth/authStore'
import { useDocumentTitle } from '@/shared/lib/useDocumentTitle'
import { EmptyState, ErrorState, LoadingState } from '@/shared/ui/states'

/**
 * UserProfilePage shows another user's tracked shows (read-only). Private
 * profiles are hidden unless you're their owner.
 */
export function UserProfilePage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const location = useLocation()
  const from = location.state?.directoryFrom
  const directoryPath = typeof from === 'string' && /^\/users(?:\?.*)?$/.test(from) ? from : '/users'
  const userId = Number(id)
  const me = useAuthStore((s) => s.user)
  const isSelf = me?.id === userId

  const profileQuery = useGetUserProfile(userId, { query: { enabled: Number.isFinite(userId) } })
  const profile = profileQuery.data
  useDocumentTitle(profile ? `${profile.display_name} — профиль` : 'Профиль')
  const canView = !!profile && (profile.is_public || isSelf || profile.is_following === 'accepted')

  const showsQuery = useListUserShows(userId, {
    query: { enabled: canView, retry: false },
  })
  const shows = showsQuery.data?.tracked ?? []
  const canAdd = !!me && !isSelf && canView
  const trackedQuery = useListTracked(undefined, { query: { enabled: canAdd, retry: false } })
  const fetchActivity = useCallback(
    (cursor?: string) => getProfileFeed(userId, { cursor, limit: 20 }),
    [userId],
  )

  return (
    <Stack gap="lg">
      <Anchor component={Link} to={directoryPath} size="sm" c="dimmed">
        ← Пользователи
      </Anchor>

      {profileQuery.isLoading && <LoadingState />}
      {profileQuery.isError && <ErrorState message="Пользователь не найден" />}

      {profile && (
        <>
          <Group justify="space-between" align="flex-start" wrap="nowrap">
            <div>
              <Title order={2}>{profile.display_name}</Title>
              <Text c="dimmed" mt={4}>
                {profile.shows_count} сериалов в коллекции
              </Text>
              <Group gap="md" mt={6}>
                <Text component={Link} to="/follows" size="sm" c="dimmed">
                  {profile.followers_count} подписчиков
                </Text>
                <Text component={Link} to="/follows" size="sm" c="dimmed">
                  {profile.following_count} подписок
                </Text>
              </Group>
            </div>
            {!isSelf && (
              <Group gap="xs" wrap="nowrap">
                <FollowButton userId={userId} state={profile.is_following} />
                {/* Blocking hides this profile (server returns 404); navigate away rather
                    than leaving the now-stale profile + collection rendered on screen. */}
                <ModerationMenu
                  userId={userId}
                  onBlocked={() => navigate(directoryPath, { replace: true })}
                />
              </Group>
            )}
          </Group>

          {!canView ? (
            <Card withBorder padding="xl">
              <EmptyState
                title="Профиль скрыт"
                description="Этот пользователь сделал свою коллекцию приватной."
              />
            </Card>
          ) : (
            <Tabs defaultValue="collection" keepMounted={false}>
              <Tabs.List>
                <Tabs.Tab value="collection">Коллекция</Tabs.Tab>
                <Tabs.Tab value="activity">Активность</Tabs.Tab>
              </Tabs.List>

              <Tabs.Panel value="collection" pt="md">
                <Card withBorder padding={0}>
                  {showsQuery.isLoading && <LoadingState />}
                  {showsQuery.isError && <ErrorState message="Не удалось загрузить сериалы" />}
                  {showsQuery.isSuccess && shows.length === 0 && (
                    <EmptyState title="Пусто" description="У пользователя пока нет сериалов." />
                  )}
                  {shows.map((t, i) => (
                    <div key={t.user_show.id}>
                      {i > 0 && <Divider />}
                      <MyShowRow
                        tracked={t}
                        readOnly
                        action={canAdd ? (trackedQuery.isError ? (
                          <Stack gap={4}>
                            <Text size="xs" c="red">Не удалось проверить твою коллекцию</Text>
                            <Button size="xs" variant="light" onClick={() => trackedQuery.refetch()}>Повторить проверку</Button>
                          </Stack>
                        ) : (
                          <AddShowButton
                            tmdbId={t.show.tmdb_id}
                            label={trackedQuery.isSuccess ? 'Добавить к себе' : 'Проверяем коллекцию…'}
                            disabled={!trackedQuery.isSuccess}
                            addedShowId={trackedQuery.data?.tracked.find((item) => item.show.tmdb_id === t.show.tmdb_id)?.show.id}
                          />
                        )) : undefined}
                      />
                    </div>
                  ))}
                </Card>
              </Tabs.Panel>

              <Tabs.Panel value="activity" pt="md">
                <ActivityFeed
                  fetchPage={fetchActivity}
                  emptyText="У пользователя пока нет активности."
                />
              </Tabs.Panel>
            </Tabs>
          )}
        </>
      )}
    </Stack>
  )
}
