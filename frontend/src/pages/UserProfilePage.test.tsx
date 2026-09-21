import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { Route, Routes } from 'react-router-dom'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { UserProfilePage } from './UserProfilePage'
import { useAuthStore } from '@/shared/auth/authStore'

const base = 'http://localhost:8080'

function renderAt(id: number) {
  return renderWithProviders(
    <Routes>
      <Route path="/users/:id" element={<UserProfilePage />} />
    </Routes>,
    { route: `/users/${id}` },
  )
}

describe('UserProfilePage', () => {
  beforeEach(() => {
    useAuthStore.setState({ user: { id: 3, display_name: 'Me', role: 'user', timezone: 'UTC' } })
    server.use(http.get(`${base}/me/shows`, () => HttpResponse.json({ tracked: [] })))
  })
  afterEach(() => useAuthStore.getState().clear())

  it('hides a private profile', async () => {
    server.use(
      http.get(`${base}/users/2`, () =>
        HttpResponse.json({ id: 2, display_name: 'Bob', is_public: false, shows_count: 5 }),
      ),
    )
    renderAt(2)
    expect(await screen.findByText('Профиль скрыт')).toBeInTheDocument()
  })

  it('shows a public profile with tracked shows', async () => {
    server.use(
      http.get(`${base}/users/1`, () =>
        HttpResponse.json({ id: 1, display_name: 'Alice', is_public: true, shows_count: 1 }),
      ),
      http.get(`${base}/users/1/shows`, () =>
        HttpResponse.json({
          tracked: [
            {
              user_show: { id: 7, show_id: 10, status: 'watching', favorite: false },
              show: { id: 10, tmdb_id: 1399, title: 'Game of Thrones', airing_status: 'ended' },
              progress: { watched: 2, total: 10 },
            },
          ],
        }),
      ),
    )
    renderAt(1)
    expect(await screen.findByText('Game of Thrones')).toBeInTheDocument()
    expect(screen.getByText('2 / 10 эпизодов')).toBeInTheDocument()
  })

  const tracked = {
    user_show: { id: 7, show_id: 10, status: 'watching', favorite: false },
    show: { id: 10, tmdb_id: 1399, title: 'Game of Thrones', airing_status: 'ended' },
    progress: { watched: 2, total: 10 },
  }

  function mockProfile() {
    server.use(
      http.get(`${base}/users/1`, () => HttpResponse.json({ id: 1, display_name: 'Alice', is_public: true, shows_count: 1 })),
      http.get(`${base}/users/1/shows`, () => HttpResponse.json({ tracked: [tracked] })),
    )
  }

  it('adds a show to my collection without copying the other user’s progress', async () => {
    mockProfile()
    let body: unknown
    server.use(http.post(`${base}/me/shows`, async ({ request }) => {
      body = await request.json()
      return HttpResponse.json({ id: 8, show_id: 10, status: 'planned', favorite: false }, { status: 201 })
    }))
    renderAt(1)
    const button = await screen.findByRole('button', { name: 'Добавить к себе' })
    await waitFor(() => expect(button).toBeEnabled())
    await userEvent.click(button)
    expect(await screen.findByRole('link', { name: 'В моих сериалах' })).toHaveAttribute('href', '/shows/10')
    expect(body).toEqual({ tmdb_id: 1399 })
    expect(screen.getByText('2 / 10 эпизодов')).toBeInTheDocument()
  })

  it('links to a show that is already in my collection', async () => {
    mockProfile()
    server.use(http.get(`${base}/me/shows`, () => HttpResponse.json({ tracked: [tracked] })))
    renderAt(1)
    expect(await screen.findByRole('link', { name: 'В моих сериалах' })).toHaveAttribute('href', '/shows/10')
    expect(screen.queryByRole('button', { name: 'Добавить к себе' })).not.toBeInTheDocument()
  })

  it('does not offer to add shows from my own profile', async () => {
    mockProfile()
    useAuthStore.setState({ user: { id: 1, display_name: 'Alice', role: 'user', timezone: 'UTC' } })
    renderAt(1)
    expect(await screen.findByText('Game of Thrones')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Добавить к себе' })).not.toBeInTheDocument()
  })
})
