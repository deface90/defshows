import { Select } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { getListTrackedQueryKey, updateShow } from '@/shared/api/tracking/endpoints'
import type { UserShowStatus } from '@/shared/api/tracking/model'
import { STATUS_OPTIONS } from '@/entities/show/status'

export function StatusSelect({ showId, status }: { showId: number; status: UserShowStatus }) {
  const queryClient = useQueryClient()
  const mutation = useMutation({
    mutationFn: (next: string) => updateShow(showId, { status: next as UserShowStatus }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: getListTrackedQueryKey() }),
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
