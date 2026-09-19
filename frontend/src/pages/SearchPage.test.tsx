import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { SearchPage } from './SearchPage'

const base = 'http://localhost:8080'

describe('SearchPage', () => {
  it('searches and renders results with an add button', async () => {
    server.use(
      http.get(`${base}/me/shows`, () => HttpResponse.json({ tracked: [] })),
      http.get(`${base}/shows/search`, () =>
        HttpResponse.json({
          results: [{ tmdb_id: 1399, title: 'Game of Thrones', poster_url: 'https://image.tmdb.org/t/p/w500/poster.jpg', first_air_date: '2011-04-17' }],
        }),
      ),
    )
    renderWithProviders(<SearchPage />)

    await userEvent.type(screen.getByLabelText('Поиск сериалов'), 'thrones')

    expect(await screen.findByText('Game of Thrones', {}, { timeout: 3000 })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Добавить' })).toBeInTheDocument()
    expect(screen.getByRole('img', { name: 'Game of Thrones' })).toHaveAttribute('src', `${base}/images/tmdb/w500/poster.jpg`)
    expect(screen.getAllByRole('link', { name: 'Game of Thrones' })[0]).toHaveAttribute('href', '/catalog/1399')
  })
})


it('shows a library link for an already added search result', async () => {
  server.use(
    http.get(`${base}/me/shows`, () => HttpResponse.json({
      tracked: [{
        user_show: { id: 5, show_id: 10, status: 'watching', favorite: false },
        show: { id: 10, tmdb_id: 1399, title: 'Game of Thrones', airing_status: 'ended' },
        progress: { watched: 0, total: 2, watched_episode_ids: [] },
      }],
    })),
    http.get(`${base}/shows/search`, () => HttpResponse.json({
      results: [{ tmdb_id: 1399, title: 'Game of Thrones' }],
    })),
  )
  renderWithProviders(<SearchPage />)
  await userEvent.type(screen.getByLabelText('Поиск сериалов'), 'thrones')
  expect(await screen.findByRole('link', { name: 'В моих сериалах' })).toHaveAttribute('href', '/shows/10')
  expect(screen.queryByRole('button', { name: 'Добавить' })).not.toBeInTheDocument()
})
