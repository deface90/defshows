import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { ForgotPasswordForm } from './ForgotPasswordForm'

const base = 'http://localhost:8080'

describe('ForgotPasswordForm', () => {
  it('shows a neutral success message after requesting a reset', async () => {
    server.use(http.post(`${base}/auth/password/forgot`, () => new HttpResponse(null, { status: 204 })))
    renderWithProviders(<ForgotPasswordForm />)

    await userEvent.type(screen.getByLabelText('Email'), 'user@example.com')
    await userEvent.click(screen.getByRole('button', { name: 'Отправить ссылку' }))

    expect(await screen.findByText(/мы отправили на его почту ссылку/i)).toBeInTheDocument()
  })

  it('shows an error when the request fails', async () => {
    server.use(http.post(`${base}/auth/password/forgot`, () => new HttpResponse(null, { status: 500 })))
    renderWithProviders(<ForgotPasswordForm />)

    await userEvent.type(screen.getByLabelText('Email'), 'user@example.com')
    await userEvent.click(screen.getByRole('button', { name: 'Отправить ссылку' }))

    expect(await screen.findByText(/не удалось отправить письмо/i)).toBeInTheDocument()
  })
})
