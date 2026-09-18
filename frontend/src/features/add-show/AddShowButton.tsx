import { Button } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { isAxiosError } from 'axios'
import { addShow } from '@/shared/api/tracking/endpoints'
import { getListTrackedQueryKey } from '@/shared/api/tracking/endpoints'

export function AddShowButton({ tmdbId }: { tmdbId: number }) {
  const queryClient = useQueryClient()

  const mutation = useMutation({
    mutationFn: () => addShow({ tmdb_id: tmdbId }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: getListTrackedQueryKey() })
      notifications.show({ message: 'Добавлено в «Мои сериалы»', color: 'teal' })
    },
    onError: (err) => {
      if (isAxiosError(err) && err.response?.status === 409) {
        notifications.show({ message: 'Сериал уже добавлен', color: 'yellow' })
      } else {
        notifications.show({ message: 'Не удалось добавить сериал', color: 'red' })
      }
    },
  })

  return (
    <Button size="xs" loading={mutation.isPending} onClick={() => mutation.mutate()}>
      Добавить
    </Button>
  )
}
