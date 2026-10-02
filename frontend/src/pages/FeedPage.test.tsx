import { screen } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { FeedPage } from './FeedPage'

const base = 'http://localhost:8080'

describe('FeedPage', () => {
  it('renders the home feed from /me/feed', async () => {
    server.use(
      http.get(`${base}/me/feed`, () =>
        HttpResponse.json({
          cards: [
            {
              type: 'added_show',
              count: 0,
              created_at: '2026-05-01T10:00:00Z',
              show: { id: 1, tmdb_id: 1399, title: 'Game of Thrones' },
              actor: { id: 5, display_name: 'Аня', is_public: true },
            },
          ],
          next_cursor: null,
        }),
      ),
    )
    renderWithProviders(<FeedPage />)
    expect(await screen.findByText('Game of Thrones')).toBeInTheDocument()
    expect(screen.getByText('Аня')).toBeInTheDocument()
    expect(screen.getByText('Добавлен в коллекцию')).toBeInTheDocument()
  })

  it('shows an empty state when the feed is empty', async () => {
    server.use(http.get(`${base}/me/feed`, () => HttpResponse.json({ cards: [], next_cursor: null })))
    renderWithProviders(<FeedPage />)
    expect(
      await screen.findByText('Подпишитесь на кого-нибудь, чтобы видеть их активность здесь.'),
    ).toBeInTheDocument()
  })
})
