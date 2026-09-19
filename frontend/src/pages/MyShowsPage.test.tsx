import { screen } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { MyShowsPage } from './MyShowsPage'

const base = 'http://localhost:8080'

describe('MyShowsPage', () => {
  it('renders tracked shows with progress', async () => {
    server.use(
      http.get(`${base}/me/shows`, () =>
        HttpResponse.json({
          tracked: [
            {
              user_show: { id: 1, show_id: 10, status: 'watching', favorite: false },
              show: { id: 10, tmdb_id: 1399, title: 'Game of Thrones', airing_status: 'ended' },
              progress: { watched: 1, total: 10 },
            },
          ],
        }),
      ),
    )
    renderWithProviders(<MyShowsPage />)

    expect(await screen.findByText('Game of Thrones')).toBeInTheDocument()
    expect(screen.getByText('1 / 10 эпизодов')).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /Смотрю/ })).toHaveAttribute('aria-selected', 'true')
  })

  it('shows an empty state when nothing is tracked', async () => {
    server.use(http.get(`${base}/me/shows`, () => HttpResponse.json({ tracked: [] })))
    renderWithProviders(<MyShowsPage />)
    expect(await screen.findByText('Пока пусто')).toBeInTheDocument()
  })
})
