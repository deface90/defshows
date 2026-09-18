import { useEffect, useState } from 'react'
import { Checkbox } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
  getGetTrackedQueryKey,
  getListTrackedQueryKey,
  unwatchEpisode,
  watchEpisode,
} from '@/shared/api/tracking/endpoints'

export function WatchToggle({
  showId,
  episodeId,
  watched,
  disabled = false,
}: {
  showId: number
  episodeId: number
  watched: boolean
  disabled?: boolean
}) {
  const queryClient = useQueryClient()
  const [checked, setChecked] = useState(watched)
  useEffect(() => setChecked(watched), [watched])

  const mutation = useMutation({
    mutationFn: (next: boolean) =>
      next ? watchEpisode(showId, episodeId) : unwatchEpisode(showId, episodeId),
    onMutate: (next) => {
      const previous = checked
      setChecked(next)
      return { previous }
    },
    onError: (_error, _next, context) => {
      if (context) setChecked(context.previous)
      notifications.show({ message: 'Не удалось обновить отметку', color: 'red' })
    },
    onSuccess: () => {
      return Promise.all([
        queryClient.invalidateQueries({ queryKey: getGetTrackedQueryKey(showId) }),
        queryClient.invalidateQueries({ queryKey: getListTrackedQueryKey() }),
      ])
    },
  })

  return (
    <Checkbox
      checked={checked}
      disabled={disabled || mutation.isPending}
      onChange={(e) => mutation.mutate(e.currentTarget.checked)}
      aria-label="Просмотрено"
    />
  )
}
