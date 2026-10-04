import { Button, Stack } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useEffect, useRef, useState } from 'react'
import type { ReactNode } from 'react'
import type { FeedCard, FeedPage } from '@/shared/api/social/model'
import { EmptyState, ErrorState, LoadingState } from '@/shared/ui/states'
import { FeedCardView } from './FeedCard'

/**
 * ActivityFeed renders a cursor-paginated activity feed, reused by both the home feed
 * and a profile's activity tab. The parent supplies a stable `fetchPage` (memoized with
 * useCallback) that loads a page given an optional cursor.
 *
 * `renderActorMenu` is an optional page slot (FSD: this entity can't import features)
 * for moderation controls on a feed-card actor; it receives a `reload` callback so an
 * action like blocking can refresh the locally-held feed state.
 */
export function ActivityFeed({
  fetchPage,
  emptyText,
  renderActorMenu,
}: {
  fetchPage: (cursor?: string) => Promise<FeedPage>
  emptyText: string
  renderActorMenu?: (actorId: number, reload: () => void) => ReactNode
}) {
  const [cards, setCards] = useState<FeedCard[]>([])
  const [cursor, setCursor] = useState<string | undefined>()
  const [reloadKey, setReloadKey] = useState(0)
  const [phase, setPhase] = useState<'loading' | 'ready' | 'error'>('loading')
  const [loadingMore, setLoadingMore] = useState(false)
  // Generation counter: a reload (e.g. after blocking an actor) bumps it so an in-flight
  // loadMore started on a previous generation is discarded instead of appending stale
  // (pre-block) cards over the refreshed list.
  const generation = useRef(0)

  useEffect(() => {
    let active = true
    generation.current = reloadKey
    setPhase('loading')
    setCards([])
    setCursor(undefined)
    // Reset the load-more flag: a reload that lands mid-flight bumps the generation so the
    // stale loadMore's finally() skips its own reset — clear it here so "Показать ещё"
    // re-enables once the refreshed first page lands (otherwise it stays disabled forever).
    setLoadingMore(false)
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
  }, [fetchPage, reloadKey])

  const reload = () => setReloadKey((k) => k + 1)

  const loadMore = () => {
    if (!cursor) return
    const startGen = generation.current
    setLoadingMore(true)
    fetchPage(cursor)
      .then((page) => {
        // Discard if a reload happened while this request was in flight.
        if (generation.current !== startGen) return
        setCards((prev) => [...prev, ...page.cards])
        setCursor(page.next_cursor)
      })
      .catch(() => {
        if (generation.current !== startGen) return
        notifications.show({ message: 'Не удалось загрузить ещё', color: 'red' })
      })
      .finally(() => {
        if (generation.current === startGen) setLoadingMore(false)
      })
  }

  if (phase === 'loading') return <LoadingState />
  if (phase === 'error') return <ErrorState message="Не удалось загрузить ленту" />
  if (cards.length === 0) return <EmptyState title="Пока пусто" description={emptyText} />

  return (
    <Stack gap="sm">
      {cards.map((card, i) => (
        <FeedCardView
          key={`${card.type}-${card.created_at}-${card.show.id}-${i}`}
          card={card}
          renderActorMenu={
            renderActorMenu ? (actorId) => renderActorMenu(actorId, reload) : undefined
          }
        />
      ))}
      {cursor && (
        <Button variant="light" loading={loadingMore} onClick={loadMore}>
          Показать ещё
        </Button>
      )}
    </Stack>
  )
}
