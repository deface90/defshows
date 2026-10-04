import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { DisplayNameForm } from './DisplayNameForm'

const base = 'http://localhost:8080'

function settings(display_name: string) {
  return http.get(`${base}/me/settings`, () =>
    HttpResponse.json({ timezone: 'UTC', is_public: true, display_name }),
  )
}

describe('DisplayNameForm', () => {
  it('saves the display name via updateSettings', async () => {
    let body: unknown
    server.use(
      settings('Старое'),
      http.patch(`${base}/me/settings`, async ({ request }) => {
        body = await request.json()
        return HttpResponse.json({ timezone: 'UTC', is_public: true, display_name: 'Новое' })
      }),
    )
    renderWithProviders(<DisplayNameForm />)

    const input = await screen.findByLabelText('Имя')
    await waitFor(() => expect(input).toHaveValue('Старое'))
    await userEvent.clear(input)
    await userEvent.type(input, 'Новое')
    await userEvent.click(screen.getByRole('button', { name: 'Сохранить' }))

    // is_public must NOT be sent — sending a cached value could revert a concurrent
    // visibility change.
    await waitFor(() => expect(body).toEqual({ timezone: 'UTC', display_name: 'Новое' }))
    expect(body).not.toHaveProperty('is_public')
  })

  it('allows a blank value to reset the name', async () => {
    let body: unknown
    server.use(
      settings('Имя'),
      http.patch(`${base}/me/settings`, async ({ request }) => {
        body = await request.json()
        return HttpResponse.json({ timezone: 'UTC', is_public: true, display_name: 'derived' })
      }),
    )
    renderWithProviders(<DisplayNameForm />)

    const input = await screen.findByLabelText('Имя')
    await waitFor(() => expect(input).toHaveValue('Имя'))
    await userEvent.clear(input)
    await userEvent.click(screen.getByRole('button', { name: 'Сохранить' }))

    await waitFor(() => expect(body).toEqual({ timezone: 'UTC', display_name: '' }))
  })

  it('rejects names longer than 50 chars without calling the API', async () => {
    let hit = false
    server.use(
      settings(''),
      http.patch(`${base}/me/settings`, () => {
        hit = true
        return HttpResponse.json({ timezone: 'UTC', is_public: true, display_name: '' })
      }),
    )
    renderWithProviders(<DisplayNameForm />)

    const input = await screen.findByLabelText('Имя')
    await userEvent.type(input, 'x'.repeat(51))
    await userEvent.click(screen.getByRole('button', { name: 'Сохранить' }))

    expect(await screen.findByText('Не более 50 символов')).toBeInTheDocument()
    expect(hit).toBe(false)
  })
})
