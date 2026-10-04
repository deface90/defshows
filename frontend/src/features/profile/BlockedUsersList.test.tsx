import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { BlockedUsersList } from './BlockedUsersList'

const base = 'http://localhost:8080'

describe('BlockedUsersList', () => {
  it('renders blocked users and unblocks one', async () => {
    let unblocked: string | undefined
    let listCalls = 0
    server.use(
      http.get(`${base}/me/blocks`, () => {
        listCalls += 1
        return HttpResponse.json(
          listCalls === 1
            ? { users: [{ id: 5, display_name: 'Тролль', is_public: true }], total: 1 }
            : { users: [], total: 0 },
        )
      }),
      http.delete(`${base}/me/blocks/5`, ({ request }) => {
        unblocked = request.method
        return new HttpResponse(null, { status: 204 })
      }),
    )
    renderWithProviders(<BlockedUsersList />)

    expect(await screen.findByText('Тролль')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Разблокировать' }))

    await waitFor(() => expect(unblocked).toBe('DELETE'))
    // list refetch after invalidation yields the empty state
    expect(await screen.findByText('Вы никого не заблокировали.')).toBeInTheDocument()
  })

  it('shows an empty state when nobody is blocked', async () => {
    server.use(http.get(`${base}/me/blocks`, () => HttpResponse.json({ users: [], total: 0 })))
    renderWithProviders(<BlockedUsersList />)
    expect(await screen.findByText('Вы никого не заблокировали.')).toBeInTheDocument()
  })

  it('paginates to older blocked users via the page control', async () => {
    const requestedPages: string[] = []
    server.use(
      http.get(`${base}/me/blocks`, ({ request }) => {
        const page = new URL(request.url).searchParams.get('page') ?? '1'
        requestedPages.push(page)
        // 25 total → 2 pages of 20.
        if (page === '2') {
          return HttpResponse.json({
            users: [{ id: 99, display_name: 'Старый блок', is_public: true }],
            total: 25,
          })
        }
        return HttpResponse.json({
          users: Array.from({ length: 20 }, (_, i) => ({
            id: i + 1,
            display_name: `Блок ${i + 1}`,
            is_public: true,
          })),
          total: 25,
        })
      }),
    )
    renderWithProviders(<BlockedUsersList />)

    expect(await screen.findByText('Блок 1')).toBeInTheDocument()
    // Second Pagination page button.
    await userEvent.click(screen.getByRole('button', { name: '2' }))
    expect(await screen.findByText('Старый блок')).toBeInTheDocument()
    await waitFor(() => expect(requestedPages).toContain('2'))
  })
})
