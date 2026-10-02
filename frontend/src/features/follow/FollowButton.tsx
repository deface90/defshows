import { Button } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { followUser, unfollowUser } from '@/shared/api/social/endpoints'
import { getGetUserProfileQueryKey } from '@/shared/api/users/endpoints'
import type { UserProfileIsFollowing } from '@/shared/api/users/model'

/**
 * FollowButton renders the viewer's follow relationship to a user and toggles it.
 * Four states: none → «Подписаться», pending → «Запрос отправлен» (click cancels),
 * accepted → «Вы подписаны» (click unfollows).
 */
export function FollowButton({ userId, state }: { userId: number; state: UserProfileIsFollowing }) {
  const queryClient = useQueryClient()
  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: getGetUserProfileQueryKey(userId) })

  const follow = useMutation({
    mutationFn: () => followUser(userId),
    onSuccess: invalidate,
    onError: () => notifications.show({ message: 'Не удалось подписаться', color: 'red' }),
  })
  const unfollow = useMutation({
    mutationFn: () => unfollowUser(userId),
    onSuccess: invalidate,
    onError: () => notifications.show({ message: 'Не удалось отписаться', color: 'red' }),
  })
  const busy = follow.isPending || unfollow.isPending

  if (state === 'accepted') {
    return (
      <Button size="xs" variant="light" loading={busy} onClick={() => unfollow.mutate()}>
        Вы подписаны
      </Button>
    )
  }
  if (state === 'pending') {
    return (
      <Button size="xs" variant="default" loading={busy} onClick={() => unfollow.mutate()}>
        Запрос отправлен
      </Button>
    )
  }
  return (
    <Button size="xs" loading={busy} onClick={() => follow.mutate()}>
      Подписаться
    </Button>
  )
}
