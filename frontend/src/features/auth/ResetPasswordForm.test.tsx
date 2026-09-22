import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { ResetPasswordForm } from './ResetPasswordForm'

const base = 'http://localhost:8080'

describe('ResetPasswordForm', () => {
  it('sends the token and new password from the URL', async () => {
    let received: { token?: string; new_password?: string } = {}
    server.use(
      http.post(`${base}/auth/password/reset`, async ({ request }) => {
        received = (await request.json()) as typeof received
        return new HttpResponse(null, { status: 204 })
      }),
    )
    renderWithProviders(<ResetPasswordForm />, { route: '/reset-password?token=abc123' })

    await userEvent.type(screen.getByLabelText('Новый пароль'), 'newpass1')
    await userEvent.type(screen.getByLabelText('Повторите пароль'), 'newpass1')
    await userEvent.click(screen.getByRole('button', { name: 'Сохранить пароль' }))

    await waitFor(() => expect(received.token).toBe('abc123'))
    expect(received.new_password).toBe('newpass1')
  })

  it('shows an error when the token is rejected', async () => {
    server.use(http.post(`${base}/auth/password/reset`, () => new HttpResponse(null, { status: 400 })))
    renderWithProviders(<ResetPasswordForm />, { route: '/reset-password?token=bad' })

    await userEvent.type(screen.getByLabelText('Новый пароль'), 'newpass1')
    await userEvent.type(screen.getByLabelText('Повторите пароль'), 'newpass1')
    await userEvent.click(screen.getByRole('button', { name: 'Сохранить пароль' }))

    expect(await screen.findByText(/ссылка недействительна или устарела/i)).toBeInTheDocument()
  })

  it('warns when no token is present', () => {
    renderWithProviders(<ResetPasswordForm />, { route: '/reset-password' })
    expect(screen.getByText(/отсутствует токен сброса/i)).toBeInTheDocument()
  })
})
