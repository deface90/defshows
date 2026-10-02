import { Button, Stack } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useEffect, useState } from 'react'
import type { FeedCard, FeedPage } from '@/shared/api/social/model'
import { EmptyState, ErrorState, LoadingState } from '@/shared/ui/states'
import { FeedCardView } from './FeedCard'

/**
 * ActivityFeed renders a cursor-paginated activity feed, reused by both the home feed
 * and a profile's activity tab. The parent supplies a stable `fetchPage` (memoized with
 * useCallback) that loads a page given an optional cursor.
 */
export function ActivityFeed({
  fetchPage,
  emptyText,
}: {
  fetchPage: (cursor?: string) => Promise<FeedPage>
  emptyText: string
}) {
  const [cards, setCards] = useState<FeedCard[]>([])
  const [cursor, setCursor] = useState<string | undefined>()
  const [phase, setPhase] = useState<'loading' | 'ready' | 'error'>('loading')
  const [loadingMore, setLoadingMore] = useState(false)

  useEffect(() => {
    let active = true
    setPhase('loading')
    setCards([])
    setCursor(undefined)
    fetchPage(undefined)
      .then((page) => {
        if (!active) return
        setCards(page.cards)
        setCursor(page.next_cursor)
        setPhase('ready')
      })
      .catch(() => {
        if (active) setPhase('error')
      })
    return () => {
      active = false
    }
  }, [fetchPage])

  const loadMore = () => {
    if (!cursor) return
    setLoadingMore(true)
    fetchPage(cursor)
      .then((page) => {
        setCards((prev) => [...prev, ...page.cards])
        setCursor(page.next_cursor)
      })
      .catch(() => notifications.show({ message: 'Не удалось загрузить ещё', color: 'red' }))
      .finally(() => setLoadingMore(false))
  }

  if (phase === 'loading') return <LoadingState />
  if (phase === 'error') return <ErrorState message="Не удалось загрузить ленту" />
  if (cards.length === 0) return <EmptyState title="Пока пусто" description={emptyText} />

  return (
    <Stack gap="sm">
      {cards.map((card, i) => (
        <FeedCardView key={`${card.type}-${card.created_at}-${card.show.id}-${i}`} card={card} />
      ))}
      {cursor && (
        <Button variant="light" loading={loadingMore} onClick={loadMore}>
          Показать ещё
        </Button>
      )}
    </Stack>
  )
}
