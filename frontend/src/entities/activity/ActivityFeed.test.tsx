import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { FeedCard } from '@/shared/api/social/model'
import { renderWithProviders } from '@/test/render'
import { ActivityFeed } from './ActivityFeed'

const card = (over: Partial<FeedCard>): FeedCard => ({
  type: 'added_show',
  count: 0,
  created_at: '2026-05-01T10:00:00Z',
  show: { id: 1, tmdb_id: 1399, title: 'Game of Thrones' },
  ...over,
})

describe('ActivityFeed', () => {
  it('renders a card per event type, grouping watched episodes into a range', async () => {
    const cards: FeedCard[] = [
      card({
        type: 'watched_episode',
        count: 3,
        episodes: [
          { season_number: 2, episode_number: 5 },
          { season_number: 2, episode_number: 1 },
          { season_number: 2, episode_number: 3 },
        ],
      }),
      card({ type: 'finished_season', season_number: 2 }),
      card({ type: 'finished_show' }),
      card({ type: 'added_show' }),
      card({ type: 'rated_show', rating: 8 }),
    ]
    const fetchPage = vi.fn().mockResolvedValue({ cards, next_cursor: undefined })

    renderWithProviders(<ActivityFeed fetchPage={fetchPage} emptyText="пусто" />)

    expect(await screen.findByText('Серии S02E01–E05 · 3 серий')).toBeInTheDocument()
    expect(screen.getByText('Сезон 2 завершён')).toBeInTheDocument()
    expect(screen.getByText('Сериал завершён')).toBeInTheDocument()
    expect(screen.getByText('Добавлен в коллекцию')).toBeInTheDocument()
    expect(screen.getByText('Оценка 8/10')).toBeInTheDocument()
    // No next cursor → no "load more".
    expect(screen.queryByRole('button', { name: 'Показать ещё' })).not.toBeInTheDocument()
  })

  it('loads more using the cursor from the previous page', async () => {
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce({
        cards: [card({ type: 'added_show', show: { id: 1, tmdb_id: 1, title: 'First' } })],
        next_cursor: 'cursor-2',
      })
      .mockResolvedValueOnce({
        cards: [card({ type: 'added_show', show: { id: 2, tmdb_id: 2, title: 'Second' } })],
        next_cursor: undefined,
      })

    renderWithProviders(<ActivityFeed fetchPage={fetchPage} emptyText="пусто" />)

    expect(await screen.findByText('First')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Показать ещё' }))

    expect(await screen.findByText('Second')).toBeInTheDocument()
    expect(fetchPage).toHaveBeenCalledWith('cursor-2')
    // First page remains (append, not replace); the button is gone at the end.
    expect(screen.getByText('First')).toBeInTheDocument()
    await waitFor(() =>
      expect(screen.queryByRole('button', { name: 'Показать ещё' })).not.toBeInTheDocument(),
    )
  })

  it('discards an in-flight loadMore when a reload happens first (block race)', async () => {
    // page 1 resolves immediately; loadMore (page 2) is deferred so we can trigger a
    // reload before it settles. The reloaded page 3 must win — stale page-2 cards dropped.
    let releaseLoadMore: (v: { cards: FeedCard[]; next_cursor?: string }) => void = () => {}
    const loadMorePromise = new Promise<{ cards: FeedCard[]; next_cursor?: string }>((res) => {
      releaseLoadMore = res
    })
    const actor = { id: 42, display_name: 'Actor', is_public: true }
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce({
        cards: [card({ type: 'added_show', actor, show: { id: 1, tmdb_id: 1, title: 'Page1' } })],
        next_cursor: 'cursor-2',
      })
      .mockReturnValueOnce(loadMorePromise) // deferred page 2
      .mockResolvedValueOnce({
        cards: [card({ type: 'added_show', actor, show: { id: 3, tmdb_id: 3, title: 'Reloaded' } })],
        // Reloaded feed has another page: the load-more button must come back ENABLED
        // (loadingMore must have been reset by the reload, not left stuck true).
        next_cursor: 'cursor-reloaded-2',
      })
      .mockResolvedValue({ cards: [], next_cursor: undefined })

    renderWithProviders(
      <ActivityFeed
        fetchPage={fetchPage}
        emptyText="пусто"
        renderActorMenu={(_actorId, reload) => (
          <button type="button" onClick={reload}>
            reload
          </button>
        )}
      />,
    )

    expect(await screen.findByText('Page1')).toBeInTheDocument()
    // Start loadMore (page 2, deferred).
    await userEvent.click(screen.getByRole('button', { name: 'Показать ещё' }))
    // Trigger a reload (simulating block) before page 2 settles.
    await userEvent.click(screen.getAllByRole('button', { name: 'reload' })[0])
    expect(await screen.findByText('Reloaded')).toBeInTheDocument()

    // Now release the stale page-2 response — it must be discarded.
    releaseLoadMore({
      cards: [card({ type: 'added_show', show: { id: 2, tmdb_id: 2, title: 'StalePage2' } })],
      next_cursor: 'stale-cursor',
    })
    await waitFor(() => expect(screen.getByText('Reloaded')).toBeInTheDocument())
    expect(screen.queryByText('StalePage2')).not.toBeInTheDocument()
    expect(screen.queryByText('Page1')).not.toBeInTheDocument()

    // The reloaded feed has another page, so "Показать ещё" must be present and ENABLED —
    // i.e. loadingMore was reset by the reload (not left stuck true by the discarded loadMore).
    const loadMoreBtn = await screen.findByRole('button', { name: 'Показать ещё' })
    expect(loadMoreBtn).not.toBeDisabled()
    // And it works: clicking it fetches the reloaded feed's next page.
    await userEvent.click(loadMoreBtn)
    expect(fetchPage).toHaveBeenLastCalledWith('cursor-reloaded-2')
  })

  it('shows the empty state when there is no activity', async () => {
    const fetchPage = vi.fn().mockResolvedValue({ cards: [], next_cursor: undefined })
    renderWithProviders(<ActivityFeed fetchPage={fetchPage} emptyText="нет активности" />)
    expect(await screen.findByText('нет активности')).toBeInTheDocument()
  })
})
