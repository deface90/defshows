import { Button } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { blockUser, getListBlocksQueryKey, unblockUser } from '@/shared/api/social/endpoints'
import {
  getGetUserProfileQueryKey,
  getListUserShowsQueryKey,
  getListUsersQueryKey,
} from '@/shared/api/users/endpoints'

/**
 * useBlockMutations wires the block/unblock mutations for a user. Blocking makes the
 * blocked user's profile (404) and collection inaccessible, so those cached entries are
 * *removed* (not just invalidated) — invalidate would refetch into a 404 while React
 * Query keeps the previous data, leaving a stale profile rendered next to the error.
 * Unblocking restores access, so it invalidates (refetch). The directory-search and
 * blocked-list queries are invalidated in both cases. `onChange` lets callers (e.g. the
 * activity feed, which keeps its own local state) refresh too.
 */
export function useBlockMutations(userId: number, onChange?: () => void) {
  const queryClient = useQueryClient()
  const refreshLists = () => {
    queryClient.invalidateQueries({ queryKey: getListUsersQueryKey() })
    queryClient.invalidateQueries({ queryKey: getListBlocksQueryKey() })
    onChange?.()
  }

  const block = useMutation({
    mutationFn: () => blockUser(userId),
    onSuccess: () => {
      notifications.show({ message: 'Пользователь заблокирован' })
      // Profile + collection are no longer viewable — drop the cache so no stale data
      // renders alongside the resulting 404.
      queryClient.removeQueries({ queryKey: getGetUserProfileQueryKey(userId) })
      queryClient.removeQueries({ queryKey: getListUserShowsQueryKey(userId) })
      refreshLists()
    },
    onError: () => notifications.show({ message: 'Не удалось заблокировать', color: 'red' }),
  })
  const unblock = useMutation({
    mutationFn: () => unblockUser(userId),
    onSuccess: () => {
      notifications.show({ message: 'Пользователь разблокирован' })
      queryClient.invalidateQueries({ queryKey: getGetUserProfileQueryKey(userId) })
      refreshLists()
    },
    onError: () => notifications.show({ message: 'Не удалось разблокировать', color: 'red' }),
  })

  return { block, unblock, busy: block.isPending || unblock.isPending }
}

/**
 * BlockButton is a standalone unblock control used in the Settings blocked-list; the
 * profile/feed overflow menus call {@link useBlockMutations} directly so Block sits
 * alongside Report in a single menu.
 */
export function BlockButton({ userId, onChange }: { userId: number; onChange?: () => void }) {
  const { unblock, busy } = useBlockMutations(userId, onChange)
  return (
    <Button size="xs" variant="light" loading={busy} onClick={() => unblock.mutate()}>
      Разблокировать
    </Button>
  )
}
