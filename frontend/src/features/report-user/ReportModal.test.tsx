import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it, vi } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { ReportModal } from './ReportModal'

const base = 'http://localhost:8080'

describe('ReportModal', () => {
  it('submits the report body with reason + note', async () => {
    let body: unknown
    server.use(
      http.post(`${base}/me/reports`, async ({ request }) => {
        body = await request.json()
        return HttpResponse.json(
          { id: 1, target_user_id: 7, reason: 'spam', status: 'open' },
          { status: 201 },
        )
      }),
    )
    const onClose = vi.fn()
    renderWithProviders(<ReportModal userId={7} opened onClose={onClose} />)

    const select = await screen.findByPlaceholderText('Выберите причину')
    await userEvent.click(select)
    await userEvent.click(await screen.findByRole('option', { name: 'Спам' }))

    await userEvent.type(screen.getByLabelText('Комментарий (необязательно)'), 'bad user')
    await userEvent.click(screen.getByRole('button', { name: 'Отправить' }))

    await waitFor(() =>
      expect(body).toEqual({ target_user_id: 7, reason: 'spam', note: 'bad user' }),
    )
    // On success the modal closes (reset + onClose).
    await waitFor(() => expect(onClose).toHaveBeenCalled())
  })

  it('requires a reason before submitting', async () => {
    let hit = false
    server.use(
      http.post(`${base}/me/reports`, () => {
        hit = true
        return HttpResponse.json({}, { status: 201 })
      }),
    )
    renderWithProviders(<ReportModal userId={7} opened onClose={() => {}} />)

    await userEvent.click(await screen.findByRole('button', { name: 'Отправить' }))

    expect(await screen.findByText('Выберите причину')).toBeInTheDocument()
    expect(hit).toBe(false)
  })
})
