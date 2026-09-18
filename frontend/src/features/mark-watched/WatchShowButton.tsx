import { Button } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
  getGetTrackedQueryKey,
  getListTrackedQueryKey,
  watchShow,
} from '@/shared/api/tracking/endpoints'

export function WatchShowButton({ showId, completed }: { showId: number; completed: boolean }) {
  const queryClient = useQueryClient()
  const mutation = useMutation({
    mutationFn: () => watchShow(showId),
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: getGetTrackedQueryKey(showId) }),
        queryClient.invalidateQueries({ queryKey: getListTrackedQueryKey() }),
      ])
      notifications.show({ message: 'Все эпизоды отмечены просмотренными', color: 'teal' })
    },
    onError: () => {
      notifications.show({ message: 'Не удалось отметить сериал просмотренным', color: 'red' })
    },
  })

  return (
    <Button
      variant="light"
      size="sm"
      loading={mutation.isPending}
      disabled={completed}
      onClick={() => mutation.mutate()}
    >
      {completed ? 'Весь сериал просмотрен ✓' : 'Отметить весь сериал просмотренным'}
    </Button>
  )
}
