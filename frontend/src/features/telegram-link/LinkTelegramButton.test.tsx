import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { LinkTelegramButton } from './LinkTelegramButton'

const base = 'http://localhost:8080'

describe('LinkTelegramButton', () => {
  afterEach(() => vi.restoreAllMocks())

  it('shows linked status and opens the deep link', async () => {
    const open = vi.spyOn(window, 'open').mockImplementation(() => null)
    server.use(
      http.get(`${base}/me/telegram`, () => HttpResponse.json({ linked: true })),
      http.get(`${base}/me/telegram/link`, () => HttpResponse.json({ url: 'https://t.me/bot?start=abc' })),
    )
    renderWithProviders(<LinkTelegramButton />)

    expect(await screen.findByText('привязан')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Перепривязать Telegram' }))

    await waitFor(() =>
      expect(open).toHaveBeenCalledWith('https://t.me/bot?start=abc', '_blank', 'noopener,noreferrer'),
    )
  })

  it('shows unlinked status', async () => {
    server.use(http.get(`${base}/me/telegram`, () => HttpResponse.json({ linked: false })))
    renderWithProviders(<LinkTelegramButton />)

    expect(await screen.findByText('не привязан')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Привязать Telegram' })).toBeInTheDocument()
  })
})
