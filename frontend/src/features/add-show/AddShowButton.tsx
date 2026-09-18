import { Link } from 'react-router-dom'
import { useState } from 'react'
import { Button } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { isAxiosError } from 'axios'
import { addShow } from '@/shared/api/tracking/endpoints'
import { getGetTrackedQueryKey, getListTrackedQueryKey } from '@/shared/api/tracking/endpoints'

export function AddShowButton({ tmdbId, addedShowId, disabled = false }: { tmdbId: number; addedShowId?: number; disabled?: boolean }) {
  const [added, setAdded] = useState<number>()
  const [alreadyAdded, setAlreadyAdded] = useState(false)
  const queryClient = useQueryClient()

  const mutation = useMutation({
    mutationFn: () => addShow({ tmdb_id: tmdbId }),
    onSuccess: (show) => {
      setAdded(show.show_id)
      queryClient.invalidateQueries({ queryKey: getGetTrackedQueryKey(show.show_id) })
      queryClient.invalidateQueries({ queryKey: getListTrackedQueryKey() })
      notifications.show({ message: 'Добавлено в «Мои сериалы»', color: 'teal' })
    },
    onError: (err) => {
      if (isAxiosError(err) && err.response?.status === 409) {
        setAlreadyAdded(true)
        queryClient.invalidateQueries({ queryKey: getListTrackedQueryKey() })
        notifications.show({ message: 'Сериал уже добавлен', color: 'yellow' })
      } else {
        notifications.show({ message: 'Не удалось добавить сериал', color: 'red' })
      }
    },
  })

  const existingId = addedShowId ?? added
  if (existingId) return <Button component={Link} to={`/shows/${existingId}`} size="xs" variant="light">В моих сериалах</Button>
  if (alreadyAdded) return <Button size="xs" disabled>Уже добавлен</Button>

  return (
    <Button disabled={disabled} size="xs" loading={mutation.isPending} onClick={() => mutation.mutate()}>
      Добавить
    </Button>
  )
}
