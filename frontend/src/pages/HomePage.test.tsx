import { screen } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { afterEach, describe, expect, it } from 'vitest'
import { useAuthStore } from '@/shared/auth/authStore'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { HomePage } from './HomePage'

const base = 'http://localhost:8080'

function login() {
  useAuthStore.setState({
    isAuthenticated: true,
    accessToken: 'token',
    user: { id: 1, display_name: 'Дефейс', role: 'user', timezone: 'UTC' },
  })
}

describe('HomePage (smart root)', () => {
  afterEach(() => useAuthStore.getState().clear())

  it('shows the marketing landing to guests', () => {
    renderWithProviders(<HomePage />)
    expect(screen.getByRole('link', { name: 'Смотреть каталог' })).toBeInTheDocument()
    // No dashboard greeting for guests.
    expect(screen.queryByText(/Привет,/)).not.toBeInTheDocument()
  })

  it('shows the personal dashboard to authenticated users', async () => {
    login()
    server.use(
      http.get(`${base}/me/shows`, () =>
        HttpResponse.json({
          tracked: [
            {
              user_show: { id: 1, show_id: 10, status: 'watching', favorite: false },
              show: { id: 10, tmdb_id: 1399, title: 'Game of Thrones', airing_status: 'airing' },
              progress: { watched: 2, total: 10, unwatched: 3 },
            },
          ],
        }),
      ),
      http.get(`${base}/me/recommendations`, () =>
        HttpResponse.json({
          taste: { genres: [{ id: 1, name: 'Драма', weight: 3 }], languages: [{ code: 'ko', weight: 2 }] },
          recommendations: [{ id: 20, tmdb_id: 2000, title: 'Recommended Show', airing_status: 'ended' }],
        }),
      ),
    )
    renderWithProviders(<HomePage />)

    expect(await screen.findByText(/Привет, Дефейс/)).toBeInTheDocument()
    // Show surfaces in both the "continue watching" and "unwatched" sections.
    expect(await screen.findAllByText('Game of Thrones')).not.toHaveLength(0)
    expect(screen.getByText('Продолжить смотреть')).toBeInTheDocument()
    expect(screen.getByText('Непросмотренные серии')).toBeInTheDocument()

    // Taste + recommendations block.
    expect(await screen.findByText('Для вас')).toBeInTheDocument()
    expect(screen.getByText('Драма')).toBeInTheDocument()
    expect(screen.getByText('Recommended Show')).toBeInTheDocument()
  })
})
