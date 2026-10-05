import { Button } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient, type QueryKey } from '@tanstack/react-query'
import { followUser, unfollowUser } from '@/shared/api/social/endpoints'
import { getGetUserProfileQueryKey } from '@/shared/api/users/endpoints'
import type { UserProfileIsFollowing } from '@/shared/api/users/model'

/**
 * FollowButton renders the viewer's follow relationship to a user and toggles it.
 * Four states: none → «Подписаться», pending → «Запрос отправлен» (click cancels),
 * accepted → «Вы подписаны» (click unfollows).
 *
 * The user profile query is always invalidated on change; `extraInvalidateKeys` lets a
 * host like the user directory also refresh its own list (whose rows carry is_following).
 */
export function FollowButton({
  userId,
  state,
  extraInvalidateKeys,
}: {
  userId: number
  state: UserProfileIsFollowing
  extraInvalidateKeys?: QueryKey[]
}) {
  const queryClient = useQueryClient()
  const invalidate = () => {
    for (const queryKey of [getGetUserProfileQueryKey(userId), ...(extraInvalidateKeys ?? [])]) {
      queryClient.invalidateQueries({ queryKey })
    }
  }

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
