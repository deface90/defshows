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

  it('shows the empty state when there is no activity', async () => {
    const fetchPage = vi.fn().mockResolvedValue({ cards: [], next_cursor: undefined })
    renderWithProviders(<ActivityFeed fetchPage={fetchPage} emptyText="нет активности" />)
    expect(await screen.findByText('нет активности')).toBeInTheDocument()
  })
})
