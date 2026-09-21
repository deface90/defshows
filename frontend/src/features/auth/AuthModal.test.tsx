import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useAuthStore } from '@/shared/auth/authStore'
import { useRequireAuth } from '@/shared/auth/useRequireAuth'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { AuthModal } from './AuthModal'

const base = 'http://localhost:8080'

function Harness({ onRun }: { onRun: () => void }) {
  const requireAuth = useRequireAuth()
  return (
    <>
      <button type="button" onClick={() => requireAuth(onRun)}>
        Триггер
      </button>
      <AuthModal />
    </>
  )
}

describe('useRequireAuth + AuthModal', () => {
  afterEach(() => useAuthStore.getState().clear())

  it('runs the action immediately for authenticated users', async () => {
    useAuthStore.setState({
      isAuthenticated: true,
      accessToken: 'token',
      user: { id: 1, display_name: 'U', role: 'user', timezone: 'UTC' },
    })
    const action = vi.fn()
    renderWithProviders(<Harness onRun={action} />)

    await userEvent.click(screen.getByRole('button', { name: 'Триггер' }))

    expect(action).toHaveBeenCalledTimes(1)
    expect(screen.queryByText('Нужен аккаунт')).not.toBeInTheDocument()
  })

  it('opens the modal for guests and replays the action after login', async () => {
    server.use(
      http.post(`${base}/auth/login`, () =>
        HttpResponse.json({
          tokens: { access_token: 'a', refresh_token: 'r' },
          user: { id: 1, display_name: 'U', role: 'user', timezone: 'UTC' },
        }),
      ),
    )
    const action = vi.fn()
    renderWithProviders(<Harness onRun={action} />)

    await userEvent.click(screen.getByRole('button', { name: 'Триггер' }))

    // Modal opens; the action is deferred, not yet run.
    expect(await screen.findByText('Нужен аккаунт')).toBeInTheDocument()
    expect(action).not.toHaveBeenCalled()

    await userEvent.type(screen.getByLabelText('Email'), 'u@example.com')
    await userEvent.type(screen.getByLabelText('Пароль'), 'password123')
    await userEvent.click(screen.getByRole('button', { name: 'Войти' }))

    // On success the deferred action runs and the modal closes.
    await waitFor(() => expect(action).toHaveBeenCalledTimes(1))
    await waitFor(() => expect(screen.queryByText('Нужен аккаунт')).not.toBeInTheDocument())
  })
})
