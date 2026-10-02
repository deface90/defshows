import { Select } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { getGetTrackedQueryKey, getListTrackedQueryKey, updateShow } from '@/shared/api/tracking/endpoints'

const CLEAR = '__clear__'
// 1..10, highest first so the common high scores are nearest the top.
const OPTIONS = [
  { value: CLEAR, label: 'Без оценки' },
  ...Array.from({ length: 10 }, (_, i) => 10 - i).map((n) => ({ value: String(n), label: String(n) })),
]

/**
 * RatingControl lets the owner set or clear their own 1..10 score for a show.
 * It is intended for the show-detail hero only (not list rows).
 */
export function RatingControl({ showId, rating }: { showId: number; rating?: number | null }) {
  const queryClient = useQueryClient()
  const mutation = useMutation({
    mutationFn: (next: string) =>
      updateShow(showId, next === CLEAR ? { clear_rating: true } : { rating: Number(next) }),
    onSuccess: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: getGetTrackedQueryKey(showId) }),
        queryClient.invalidateQueries({ queryKey: getListTrackedQueryKey() }),
      ]),
    onError: () => notifications.show({ message: 'Не удалось изменить оценку', color: 'red' }),
  })

  return (
    <Select
      size="xs"
      w={150}
      data={OPTIONS}
      value={rating != null ? String(rating) : CLEAR}
      allowDeselect={false}
      disabled={mutation.isPending}
      onChange={(v) => v && mutation.mutate(v)}
      aria-label="Моя оценка"
    />
  )
}
