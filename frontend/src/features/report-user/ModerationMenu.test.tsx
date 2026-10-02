import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { ModerationMenu } from './ModerationMenu'

const base = 'http://localhost:8080'

describe('ModerationMenu', () => {
  it('blocks the user from the menu and calls onBlocked', async () => {
    let blocked = false
    let onBlockedCalled = false
    server.use(
      http.post(`${base}/me/blocks/7`, () => {
        blocked = true
        return new HttpResponse(null, { status: 204 })
      }),
    )
    renderWithProviders(<ModerationMenu userId={7} onBlocked={() => (onBlockedCalled = true)} />)

    await userEvent.click(screen.getByRole('button', { name: 'Действия с пользователем' }))
    await userEvent.click(await screen.findByRole('menuitem', { name: 'Заблокировать' }))

    await waitFor(() => expect(blocked).toBe(true))
    await waitFor(() => expect(onBlockedCalled).toBe(true))
  })

  it('opens the report modal from the menu', async () => {
    renderWithProviders(<ModerationMenu userId={7} />)
    await userEvent.click(screen.getByRole('button', { name: 'Действия с пользователем' }))
    await userEvent.click(await screen.findByRole('menuitem', { name: 'Пожаловаться' }))
    expect(await screen.findByText('Пожаловаться на пользователя')).toBeInTheDocument()
  })
})
