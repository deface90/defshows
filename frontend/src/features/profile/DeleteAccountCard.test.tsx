import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { afterEach, describe, expect, it } from 'vitest'
import { useAuthStore } from '@/shared/auth/authStore'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { DeleteAccountCard } from './DeleteAccountCard'

const base = 'http://localhost:8080'

describe('DeleteAccountCard', () => {
  afterEach(() => {
    useAuthStore.getState().clear()
    localStorage.clear()
  })

  it('deletes the account after confirmation and clears the session', async () => {
    let deleted = false
    server.use(
      http.delete(`${base}/auth/me`, () => {
        deleted = true
        return new HttpResponse(null, { status: 204 })
      }),
    )
    useAuthStore.getState().setSession('acc', 'ref', {
      id: 1, email: 'user@example.com', display_name: '', role: 'user', timezone: 'UTC',
    })
    renderWithProviders(<DeleteAccountCard />)

    await userEvent.click(screen.getByRole('button', { name: 'Удалить аккаунт' }))
    expect(deleted).toBe(false)
    await userEvent.click(await screen.findByRole('button', { name: 'Удалить навсегда' }))

    await waitFor(() => expect(useAuthStore.getState().accessToken).toBeNull())
    expect(deleted).toBe(true)
  })

  it('keeps the session and shows an error when deletion fails', async () => {
    server.use(http.delete(`${base}/auth/me`, () => new HttpResponse(null, { status: 500 })))
    useAuthStore.getState().setSession('acc', 'ref', {
      id: 1, email: 'user@example.com', display_name: '', role: 'user', timezone: 'UTC',
    })
    renderWithProviders(<DeleteAccountCard />)

    await userEvent.click(screen.getByRole('button', { name: 'Удалить аккаунт' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Удалить навсегда' }))

    expect(await screen.findByText('Не удалось удалить аккаунт')).toBeInTheDocument()
    expect(useAuthStore.getState().accessToken).toBe('acc')
  })
})
