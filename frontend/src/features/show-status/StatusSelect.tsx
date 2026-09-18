import { Select } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { getGetTrackedQueryKey, getListTrackedQueryKey, updateShow } from '@/shared/api/tracking/endpoints'
import type { UserShowStatus } from '@/shared/api/tracking/model'
import { STATUS_OPTIONS } from '@/entities/show/status'

export function StatusSelect({ showId, status }: { showId: number; status: UserShowStatus }) {
  const queryClient = useQueryClient()
  const mutation = useMutation({
    mutationFn: (next: string) => updateShow(showId, { status: next as UserShowStatus }),
    onSuccess: () =>
      Promise.all([
        // The detail page reads this single-show query; invalidate it too so the
        // select reflects the new status instead of the stale cached value.
        queryClient.invalidateQueries({ queryKey: getGetTrackedQueryKey(showId) }),
        queryClient.invalidateQueries({ queryKey: getListTrackedQueryKey() }),
      ]),
    onError: () => notifications.show({ message: 'Не удалось изменить статус', color: 'red' }),
  })

  return (
    <Select
      size="xs"
      w={150}
      data={STATUS_OPTIONS}
      value={status}
      allowDeselect={false}
      disabled={mutation.isPending}
      onChange={(v) => v && mutation.mutate(v)}
      aria-label="Статус"
    />
  )
}
