import { screen } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { Route, Routes } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { ShowDetailPage } from './ShowDetailPage'

const base = 'http://localhost:8080'

const show = {
  id: 10,
  tmdb_id: 1399,
  title: 'Game of Thrones',
  airing_status: 'ended',
  ratings: [{ source: 'imdb', value: '9.2/10' }],
  seasons: [
    {
      id: 1,
      season_number: 1,
      name: 'Сезон 1',
      episodes: [
        { id: 100, season_number: 1, episode_number: 1, name: 'Winter Is Coming' },
        { id: 101, season_number: 1, episode_number: 2, name: 'The Kingsroad' },
      ],
    },
  ],
}

function renderDetail() {
  return renderWithProviders(
    <Routes>
      <Route path="/shows/:id" element={<ShowDetailPage />} />
    </Routes>,
    { route: '/shows/10' },
  )
}

describe('ShowDetailPage', () => {
  it('renders detail, ratings and per-episode watched state when tracked', async () => {
    server.use(
      http.get(`${base}/shows/10`, () => HttpResponse.json(show)),
      http.get(`${base}/me/shows/10`, () =>
        HttpResponse.json({
          user_show: { id: 5, show_id: 10, status: 'watching', favorite: false },
          show: { id: 10, tmdb_id: 1399, title: 'Game of Thrones', airing_status: 'ended' },
          progress: { watched: 1, total: 2, next_unwatched_episode_id: 101 },
        }),
      ),
    )
    renderDetail()

    expect(await screen.findByText('Game of Thrones')).toBeInTheDocument()
    expect(screen.getByText('IMDb: 9.2/10')).toBeInTheDocument()
    expect(await screen.findByText('Winter Is Coming')).toBeInTheDocument()

    // Episode 100 (before next unwatched 101) is checked; 101 is not.
    const boxes = await screen.findAllByRole('checkbox', { name: 'Просмотрено' })
    expect(boxes[0]).toBeChecked()
    expect(boxes[1]).not.toBeChecked()

    // Season toggle is indeterminate: 1 of 2 episodes watched.
    const seasonBox = screen.getByRole('checkbox', { name: 'Отметить сезон просмотренным' })
    expect(seasonBox).not.toBeChecked()
    expect(seasonBox).toHaveProperty('indeterminate', true)
  })

  it('shows an Add button when not tracked', async () => {
    server.use(
      http.get(`${base}/shows/10`, () => HttpResponse.json(show)),
      http.get(`${base}/me/shows/10`, () => new HttpResponse(null, { status: 404 })),
    )
    renderDetail()

    expect(await screen.findByText('Game of Thrones')).toBeInTheDocument()
    expect(await screen.findByRole('button', { name: 'Добавить' })).toBeInTheDocument()
  })
})
