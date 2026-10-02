import { Button } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { blockUser, getListBlocksQueryKey, unblockUser } from '@/shared/api/social/endpoints'
import { getGetUserProfileQueryKey, getListUsersQueryKey } from '@/shared/api/users/endpoints'

/**
 * useBlockMutations wires the block/unblock mutations for a user, invalidating the
 * profile, directory-search and blocked-list queries on success. Blocking tears down
 * follow edges server-side, so a refetch is enough to hide the blocked user's
 * collection/feed — `onChange` lets callers (e.g. the activity feed, which keeps its
 * own local state) refresh too.
 */
export function useBlockMutations(userId: number, onChange?: () => void) {
  const queryClient = useQueryClient()
  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: getGetUserProfileQueryKey(userId) })
    queryClient.invalidateQueries({ queryKey: getListUsersQueryKey() })
    queryClient.invalidateQueries({ queryKey: getListBlocksQueryKey() })
    onChange?.()
  }

  const block = useMutation({
    mutationFn: () => blockUser(userId),
    onSuccess: () => {
      notifications.show({ message: 'Пользователь заблокирован' })
      invalidate()
    },
    onError: () => notifications.show({ message: 'Не удалось заблокировать', color: 'red' }),
  })
  const unblock = useMutation({
    mutationFn: () => unblockUser(userId),
    onSuccess: () => {
      notifications.show({ message: 'Пользователь разблокирован' })
      invalidate()
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
