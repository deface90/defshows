import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { afterEach, describe, expect, it } from 'vitest'
import { useAuthStore } from '@/shared/auth/authStore'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { ChangePasswordForm } from './ChangePasswordForm'

const base = 'http://localhost:8080'

const authResponse = {
  user: { id: 1, email: 'user@example.com', display_name: '', role: 'user', timezone: 'UTC' },
  tokens: { access_token: 'acc2', refresh_token: 'ref2' },
}

describe('ChangePasswordForm', () => {
  afterEach(() => {
    useAuthStore.getState().clear()
    localStorage.clear()
  })

  it('applies the fresh session returned on success', async () => {
    server.use(http.post(`${base}/auth/password/change`, () => HttpResponse.json(authResponse)))
    renderWithProviders(<ChangePasswordForm />)

    await userEvent.type(screen.getByLabelText('Текущий пароль'), 'oldpass1')
    await userEvent.type(screen.getByLabelText('Новый пароль'), 'newpass1')
    await userEvent.type(screen.getByLabelText('Повторите новый пароль'), 'newpass1')
    await userEvent.click(screen.getByRole('button', { name: 'Сменить пароль' }))

    await waitFor(() => expect(useAuthStore.getState().accessToken).toBe('acc2'))
  })

  it('shows a specific error on a wrong current password (401)', async () => {
    server.use(http.post(`${base}/auth/password/change`, () => new HttpResponse(null, { status: 401 })))
    renderWithProviders(<ChangePasswordForm />)

    await userEvent.type(screen.getByLabelText('Текущий пароль'), 'wrong')
    await userEvent.type(screen.getByLabelText('Новый пароль'), 'newpass1')
    await userEvent.type(screen.getByLabelText('Повторите новый пароль'), 'newpass1')
    await userEvent.click(screen.getByRole('button', { name: 'Сменить пароль' }))

    expect(await screen.findByText('Неверный текущий пароль')).toBeInTheDocument()
  })
})
