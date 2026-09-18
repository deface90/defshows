import { Checkbox } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
  getGetTrackedQueryKey,
  getListTrackedQueryKey,
  unwatchEpisode,
  watchEpisode,
} from '@/shared/api/tracking/endpoints'
import type { Episode } from '@/shared/api/shows/model'

/**
 * Marks an entire season watched/unwatched in one click. There is no batch
 * endpoint, so we fan out over the per-episode watch/unwatch calls for the
 * episodes that actually need to change, then invalidate the tracked query.
 */
export function SeasonWatchToggle({
  showId,
  episodes,
  watchedIds,
}: {
  showId: number
  episodes: Episode[]
  watchedIds: Set<number>
}) {
  const queryClient = useQueryClient()
  const watchedCount = episodes.reduce((n, ep) => (watchedIds.has(ep.id) ? n + 1 : n), 0)
  const allWatched = episodes.length > 0 && watchedCount === episodes.length
  const someWatched = watchedCount > 0 && !allWatched

  const mutation = useMutation({
    mutationFn: async (next: boolean) => {
      const targets = episodes.filter((ep) => watchedIds.has(ep.id) !== next)
      const results = await Promise.allSettled(
        targets.map((ep) =>
          next ? watchEpisode(showId, ep.id) : unwatchEpisode(showId, ep.id),
        ),
      )
      const failed = results.find((result) => result.status === 'rejected')
      if (failed?.status === 'rejected') throw failed.reason
    },
    onSuccess: () => {
      return Promise.all([
        queryClient.invalidateQueries({ queryKey: getGetTrackedQueryKey(showId) }),
        queryClient.invalidateQueries({ queryKey: getListTrackedQueryKey() }),
      ])
    },
    onError: () => {
      void queryClient.invalidateQueries({ queryKey: getGetTrackedQueryKey(showId) })
      notifications.show({ message: 'Не удалось обновить сезон', color: 'red' })
    },
  })

  return (
    <Checkbox
      checked={allWatched}
      indeterminate={someWatched}
      disabled={mutation.isPending || episodes.length === 0}
      onChange={(e) => mutation.mutate(e.currentTarget.checked)}
      onClick={(e) => e.stopPropagation()}
      aria-label="Отметить сезон просмотренным"
    />
  )
}
