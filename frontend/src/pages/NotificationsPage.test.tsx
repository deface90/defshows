import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { NotificationsPage } from './NotificationsPage'

const base = 'http://localhost:8080'

describe('NotificationsPage', () => {
  it('renders the notification feed', async () => {
    server.use(
      http.get(`${base}/me/notifications`, () =>
        HttpResponse.json({
          notifications: [
            {
              id: 1,
              type: 'episode_released',
              status: 'sent',
              read: false,
              payload: 'Вышел новый эпизод «Foo»',
              created_at: '2026-09-16T10:00:00Z',
            },
          ],
        }),
      ),
    )
    renderWithProviders(<NotificationsPage />)

    expect(await screen.findByText('Вышел новый эпизод «Foo»')).toBeInTheDocument()
    expect(screen.getByText('Новый эпизод')).toBeInTheDocument()
  })

  it('shows an empty state when there are no notifications', async () => {
    server.use(http.get(`${base}/me/notifications`, () => HttpResponse.json({ notifications: [] })))
    renderWithProviders(<NotificationsPage />)

    expect(await screen.findByText('Уведомлений нет')).toBeInTheDocument()
  })

  it('marks a single notification read', async () => {
    let markedId: number | undefined
    let read = false
    server.use(
      http.get(`${base}/me/notifications`, () =>
        HttpResponse.json({
          notifications: [
            {
              id: 7,
              type: 'episode_released',
              status: 'sent',
              read,
              payload: 'Эпизод вышел',
              created_at: '2026-09-16T10:00:00Z',
            },
          ],
        }),
      ),
      http.post(`${base}/me/notifications/7/read`, () => {
        markedId = 7
        read = true
        return new HttpResponse(null, { status: 204 })
      }),
    )
    renderWithProviders(<NotificationsPage />)

    const card = (await screen.findByText('Эпизод вышел')).closest('.mantine-Card-root') as HTMLElement
    await userEvent.click(within(card).getByLabelText('Отметить прочитанным'))

    await waitFor(() => expect(markedId).toBe(7))
    // After invalidation the item is read → the mark control disappears.
    await waitFor(() =>
      expect(within(card).queryByLabelText('Отметить прочитанным')).not.toBeInTheDocument(),
    )
  })

  it('marks all notifications read', async () => {
    let allMarked = false
    server.use(
      http.get(`${base}/me/notifications`, () =>
        HttpResponse.json({
          notifications: [
            { id: 1, type: 'episode_released', status: 'sent', read: allMarked, payload: 'a', created_at: '2026-09-16' },
            { id: 2, type: 'season_upcoming', status: 'sent', read: allMarked, payload: 'b', created_at: '2026-09-16' },
          ],
        }),
      ),
      http.post(`${base}/me/notifications/read-all`, () => {
        allMarked = true
        return new HttpResponse(null, { status: 204 })
      }),
    )
    renderWithProviders(<NotificationsPage />)

    await userEvent.click(await screen.findByRole('button', { name: 'Прочитать всё' }))

    await waitFor(() => expect(allMarked).toBe(true))
    await waitFor(() =>
      expect(screen.queryByRole('button', { name: 'Прочитать всё' })).not.toBeInTheDocument(),
    )
  })
})
