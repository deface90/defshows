import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { Route, Routes } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { useAuthStore } from '@/shared/auth/authStore'
import { CatalogShowPage } from './CatalogShowPage'

const base = 'http://localhost:8080'
const show = {
  id: 10, tmdb_id: 1399, title: 'Game of Thrones', airing_status: 'ended', overview: 'Overview',
  seasons: [{ id: 1, season_number: 1, name: 'Сезон 1', episodes: [
    { id: 100, season_number: 1, episode_number: 1, name: 'Winter Is Coming' },
  ] }],
}
function renderPage() {
  return renderWithProviders(<Routes><Route path="/catalog/:tmdbId" element={<CatalogShowPage />} /></Routes>, { route: '/catalog/1399' })
}

describe('CatalogShowPage', () => {
  // Adding is gated behind auth; sign in so the click posts instead of opening
  // the auth modal.
  beforeEach(() =>
    useAuthStore.setState({
      isAuthenticated: true,
      accessToken: 'token',
      user: { id: 1, display_name: 'U', role: 'user', timezone: 'UTC' },
    }),
  )
  afterEach(() => useAuthStore.getState().clear())

  it('shows read-only details and changes the action after adding', async () => {
    let added = false
    server.use(
      http.get(`${base}/shows/tmdb/1399`, () => HttpResponse.json(show)),
      http.get(`${base}/me/shows/10`, () => added ? HttpResponse.json({ user_show: { show_id: 10 }, show, progress: { watched: 0, total: 1, watched_episode_ids: [] } }) : new HttpResponse(null, { status: 404 })),
      http.post(`${base}/me/shows`, () => {
        added = true
        return HttpResponse.json({ id: 5, show_id: 10, status: 'watching', favorite: false }, { status: 201 })
      }),
    )
    renderPage()
    expect(await screen.findByText('Winter Is Coming')).toBeInTheDocument()
    expect(screen.getByText('Overview')).toBeInTheDocument()
    expect(screen.queryByRole('checkbox')).not.toBeInTheDocument()
    expect(screen.queryByRole('tab')).not.toBeInTheDocument()
    expect(added).toBe(false)
    await userEvent.click(await screen.findByRole('button', { name: 'Добавить' }))
    expect(await screen.findByRole('link', { name: /В моих сериалах/ })).toHaveAttribute('href', '/shows/10')
    expect(screen.queryByRole('checkbox')).not.toBeInTheDocument()
    expect(screen.queryByRole('tab')).not.toBeInTheDocument()
  })

  it('links to an existing tracked show without offering to add it', async () => {
    server.use(
      http.get(`${base}/shows/tmdb/1399`, () => HttpResponse.json(show)),
      http.get(`${base}/me/shows/10`, () => HttpResponse.json({ user_show: { show_id: 10 }, show, progress: { watched: 1, total: 1, watched_episode_ids: [100] } })),
    )
    renderPage()
    expect(await screen.findByRole('link', { name: /В моих сериалах/ })).toHaveAttribute('href', '/shows/10')
    expect(screen.queryByRole('button', { name: 'Добавить' })).not.toBeInTheDocument()
    expect(screen.queryByRole('checkbox')).not.toBeInTheDocument()
    expect(screen.queryByRole('tab')).not.toBeInTheDocument()
  })
})
