import { screen } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { Route, Routes } from 'react-router-dom'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { UserProfilePage } from './UserProfilePage'

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
})
