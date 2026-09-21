import { Route, Routes } from 'react-router-dom'
import { UserProfilePage } from './UserProfilePage'
import userEvent from '@testing-library/user-event'
import { screen } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { UsersPage } from './UsersPage'

const base = 'http://localhost:8080'

describe('UsersPage', () => {
  it('lists users with visibility badges', async () => {
    server.use(
      http.get(`${base}/users`, () =>
        HttpResponse.json({
          total: 2, page: 1, page_size: 20,
          users: [
            { id: 1, display_name: 'Alice', is_public: true, shows_count: 3 },
            { id: 2, display_name: 'Bob', is_public: false, shows_count: 0 },
          ],
        }),
      ),
    )
    renderWithProviders(<UsersPage />)

    expect(await screen.findByText('Alice')).toBeInTheDocument()
    expect(screen.getByText('Bob')).toBeInTheDocument()
    expect(screen.getByText('Публичный')).toBeInTheDocument()
    expect(screen.getByText('Приватный')).toBeInTheDocument()
  })
})


it('loads another page from the API and resets the page when searching', async () => {
  const requests: string[] = []
  server.use(http.get(`${base}/users`, ({ request }) => {
    const url = new URL(request.url)
    requests.push(url.search)
    const q = url.searchParams.get('q')
    const page = Number(url.searchParams.get('page'))
    return HttpResponse.json({
      total: q ? 1 : 21, page, page_size: 20,
      users: [{ id: page, display_name: q ? 'Alice' : page === 2 ? 'Bob' : 'First user', is_public: true, shows_count: 0 }],
    })
  }))
  renderWithProviders(<UsersPage />)
  expect(await screen.findByText('First user')).toBeInTheDocument()
  await userEvent.click(screen.getByRole('button', { name: 'Страница 2' }))
  expect(await screen.findByText('Bob')).toBeInTheDocument()
  expect(requests.at(-1)).toContain('page=2')
  await userEvent.type(screen.getByLabelText('Поиск пользователей'), 'Alice{enter}')
  expect(await screen.findByText('Alice')).toBeInTheDocument()
  expect(requests.at(-1)).toContain('q=Alice')
  expect(requests.at(-1)).toContain('page=1')
  expect(screen.queryByRole('button', { name: 'Страница 2' })).not.toBeInTheDocument()
  await userEvent.click(screen.getByRole('button', { name: 'Сбросить' }))
  expect(await screen.findByText('First user')).toBeInTheDocument()
})

it('restores search and page from the URL', async () => {
  let search = ''
  server.use(http.get(`${base}/users`, ({ request }) => {
    search = new URL(request.url).search
    return HttpResponse.json({ users: [{ id: 9, display_name: 'Alice', is_public: true, shows_count: 0 }], total: 21, page: 2, page_size: 20 })
  }))
  renderWithProviders(<UsersPage />, { route: '/users?q=ali&page=2' })
  expect(await screen.findByText('Alice')).toBeInTheDocument()
  expect(screen.getByLabelText('Поиск пользователей')).toHaveValue('ali')
  expect(search).toContain('q=ali')
  expect(search).toContain('page=2')
})

it('shows an empty search result and allows retry after a network error', async () => {
  let failed = true
  server.use(http.get(`${base}/users`, () => failed
    ? new HttpResponse(null, { status: 500 })
    : HttpResponse.json({ users: [], total: 0, page: 1, page_size: 20 })))
  renderWithProviders(<UsersPage />, { route: '/users?q=missing' })
  expect(await screen.findByText('Не удалось загрузить список пользователей')).toBeInTheDocument()
  failed = false
  await userEvent.click(screen.getByRole('button', { name: 'Повторить' }))
  expect(await screen.findByText('Пользователи не найдены')).toBeInTheDocument()
})


it('returns from a profile to the same directory search and page', async () => {
  server.use(
    http.get(`${base}/users`, () => HttpResponse.json({ users: [{ id: 9, display_name: 'Alice', is_public: true, shows_count: 0 }], total: 21, page: 2, page_size: 20 })),
    http.get(`${base}/users/9`, () => HttpResponse.json({ id: 9, display_name: 'Alice', is_public: true, shows_count: 0 })),
    http.get(`${base}/users/9/shows`, () => HttpResponse.json({ tracked: [] })),
  )
  renderWithProviders(<Routes>
    <Route path="/users" element={<UsersPage />} />
    <Route path="/users/:id" element={<UserProfilePage />} />
  </Routes>, { route: '/users?q=ali&page=2' })
  await userEvent.click(await screen.findByRole('link', { name: /Alice/ }))
  const back = await screen.findByRole('link', { name: '← Пользователи' })
  expect(back).toHaveAttribute('href', '/users?q=ali&page=2')
  await userEvent.click(back)
  expect(await screen.findByLabelText('Поиск пользователей')).toHaveValue('ali')
})
