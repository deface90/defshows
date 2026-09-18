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
