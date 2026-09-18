import { Checkbox } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import {
  getGetTrackedQueryKey,
  unwatchEpisode,
  watchEpisode,
} from '@/shared/api/tracking/endpoints'

export function WatchToggle({
  showId,
  episodeId,
  watched,
}: {
  showId: number
  episodeId: number
  watched: boolean
}) {
  const queryClient = useQueryClient()
  const [checked, setChecked] = useState(watched)

  const mutation = useMutation({
    mutationFn: (next: boolean) =>
      next ? watchEpisode(showId, episodeId) : unwatchEpisode(showId, episodeId),
    onMutate: (next) => {
      const prev = checked
      setChecked(next) // optimistic
      return { prev }
    },
    onError: (_e, _next, ctx) => {
      if (ctx) setChecked(ctx.prev) // rollback
      notifications.show({ message: 'Не удалось обновить отметку', color: 'red' })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: getGetTrackedQueryKey(showId) })
    },
  })

  return (
    <Checkbox
      checked={checked}
      disabled={mutation.isPending}
      onChange={(e) => mutation.mutate(e.currentTarget.checked)}
      aria-label="Просмотрено"
    />
  )
}
