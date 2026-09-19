import { screen } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { UnwatchedPage } from './UnwatchedPage'

const base = 'http://localhost:8080'

const tracked = [
  {
    user_show: { id: 1, show_id: 10, status: 'watching', favorite: false },
    show: { id: 10, tmdb_id: 1399, title: 'Game of Thrones', airing_status: 'ended' },
    progress: { watched: 8, total: 10, unwatched: 2, watched_episode_ids: [] },
  },
  {
    user_show: { id: 2, show_id: 20, status: 'watching', favorite: false },
    show: { id: 20, tmdb_id: 66732, title: 'Stranger Things', airing_status: 'airing' },
    progress: { watched: 10, total: 10, unwatched: 0, watched_episode_ids: [] },
  },
]

describe('UnwatchedPage', () => {
  it('lists only shows with aired-unwatched episodes and a per-show counter', async () => {
    server.use(http.get(`${base}/me/shows`, () => HttpResponse.json({ tracked })))
    renderWithProviders(<UnwatchedPage />)

    expect(await screen.findByText('Game of Thrones')).toBeInTheDocument()
    // The fully-watched show is filtered out.
    expect(screen.queryByText('Stranger Things')).not.toBeInTheDocument()
    // Per-show unwatched counter badge.
    expect(screen.getByText('🔴 2 к просмотру')).toBeInTheDocument()
  })

  it('shows an empty state when everything is watched', async () => {
    server.use(
      http.get(`${base}/me/shows`, () =>
        HttpResponse.json({ tracked: [tracked[1]] }),
      ),
    )
    renderWithProviders(<UnwatchedPage />)
    expect(await screen.findByText('Всё просмотрено')).toBeInTheDocument()
  })
})
