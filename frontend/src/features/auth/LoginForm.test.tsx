import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { afterEach, describe, expect, it } from 'vitest'
import { useAuthStore } from '@/shared/auth/authStore'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { LoginForm } from './LoginForm'

const base = 'http://localhost:8080'

const authResponse = {
  user: { id: 1, email: 'user@example.com', display_name: '', role: 'user', timezone: 'UTC' },
  tokens: { access_token: 'acc', refresh_token: 'ref' },
}

describe('LoginForm', () => {
  afterEach(() => {
    useAuthStore.getState().clear()
    localStorage.clear()
  })

  it('logs in and stores the session', async () => {
    server.use(http.post(`${base}/auth/login`, () => HttpResponse.json(authResponse)))
    renderWithProviders(<LoginForm />)

    await userEvent.type(screen.getByLabelText('Email'), 'user@example.com')
    await userEvent.type(screen.getByLabelText('Пароль'), 'pw12345')
    await userEvent.click(screen.getByRole('button', { name: 'Войти' }))

    await waitFor(() => expect(useAuthStore.getState().isAuthenticated).toBe(true))
    expect(useAuthStore.getState().accessToken).toBe('acc')
  })

  it('shows an error on 401', async () => {
    server.use(http.post(`${base}/auth/login`, () => new HttpResponse(null, { status: 401 })))
    renderWithProviders(<LoginForm />)

    await userEvent.type(screen.getByLabelText('Email'), 'user@example.com')
    await userEvent.type(screen.getByLabelText('Пароль'), 'wrong')
    await userEvent.click(screen.getByRole('button', { name: 'Войти' }))

    expect(await screen.findByText('Неверный email или пароль')).toBeInTheDocument()
    expect(useAuthStore.getState().isAuthenticated).toBe(false)
  })
})
