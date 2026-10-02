import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { afterEach, describe, expect, it } from 'vitest'
import { useAuthStore } from '@/shared/auth/authStore'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { FollowsPage } from './FollowsPage'

const base = 'http://localhost:8080'

function login() {
  useAuthStore.setState({
    isAuthenticated: true,
    accessToken: 'token',
    user: { id: 1, display_name: 'Дефейс', role: 'user', timezone: 'UTC' },
  })
}

describe('FollowsPage', () => {
  afterEach(() => useAuthStore.getState().clear())

  it('lists followers by default and switches to following', async () => {
    login()
    server.use(
      http.get(`${base}/me/follows/incoming`, () => HttpResponse.json({ users: [], total: 0 })),
      http.get(`${base}/users/1/followers`, () =>
        HttpResponse.json({ users: [{ id: 2, display_name: 'Алиса', is_public: true }], total: 1 }),
      ),
      http.get(`${base}/users/1/following`, () =>
        HttpResponse.json({ users: [{ id: 3, display_name: 'Боб', is_public: false }], total: 1 }),
      ),
    )
    renderWithProviders(<FollowsPage />)
    expect(await screen.findByText('Алиса')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('tab', { name: 'Я подписан' }))
    expect(await screen.findByText('Боб')).toBeInTheDocument()
  })

  it('removes a follower and refetches the list', async () => {
    login()
    let removed = false
    server.use(
      http.get(`${base}/me/follows/incoming`, () => HttpResponse.json({ users: [], total: 0 })),
      http.get(`${base}/users/1/followers`, () =>
        HttpResponse.json({
          users: removed ? [] : [{ id: 2, display_name: 'Алиса', is_public: true }],
          total: removed ? 0 : 1,
        }),
      ),
      http.delete(`${base}/me/followers/2`, () => {
        removed = true
        return new HttpResponse(null, { status: 204 })
      }),
    )
    renderWithProviders(<FollowsPage />)

    expect(await screen.findByText('Алиса')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Убрать' }))

    await waitFor(() => expect(removed).toBe(true))
    await waitFor(() => expect(screen.queryByText('Алиса')).not.toBeInTheDocument())
  })

  it('shows a pending badge and approves a request', async () => {
    login()
    let approved = false
    server.use(
      http.get(`${base}/me/follows/incoming`, () =>
        HttpResponse.json({
          users: approved ? [] : [{ id: 9, display_name: 'Проситель', is_public: true }],
          total: approved ? 0 : 1,
        }),
      ),
      http.get(`${base}/users/1/followers`, () => HttpResponse.json({ users: [], total: 0 })),
      http.post(`${base}/me/follows/incoming/9/approve`, () => {
        approved = true
        return new HttpResponse(null, { status: 204 })
      }),
    )
    renderWithProviders(<FollowsPage />)

    const requestsTab = await screen.findByRole('tab', { name: /Запросы/ })
    // Badge count surfaces on the tab once the incoming query resolves.
    expect(await within(requestsTab).findByText('1')).toBeInTheDocument()

    await userEvent.click(requestsTab)
    expect(await screen.findByText('Проситель')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Принять' }))
    await waitFor(() => expect(approved).toBe(true))
  })
})
