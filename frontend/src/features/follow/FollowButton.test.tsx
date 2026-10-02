import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { FollowButton } from './FollowButton'

const base = 'http://localhost:8080'

describe('FollowButton', () => {
  it('shows «Подписаться» for state none and follows', async () => {
    let hit = false
    server.use(
      http.post(`${base}/me/follows/7`, () => {
        hit = true
        return HttpResponse.json({ status: 'pending' })
      }),
    )
    renderWithProviders(<FollowButton userId={7} state="none" />)
    const btn = screen.getByRole('button', { name: 'Подписаться' })
    await userEvent.click(btn)
    await waitFor(() => expect(hit).toBe(true))
  })

  it('shows «Запрос отправлен» for pending and cancels via unfollow', async () => {
    let hit = false
    server.use(
      http.delete(`${base}/me/follows/7`, () => {
        hit = true
        return new HttpResponse(null, { status: 204 })
      }),
    )
    renderWithProviders(<FollowButton userId={7} state="pending" />)
    await userEvent.click(screen.getByRole('button', { name: 'Запрос отправлен' }))
    await waitFor(() => expect(hit).toBe(true))
  })

  it('shows «Вы подписаны» for accepted and unfollows', async () => {
    let hit = false
    server.use(
      http.delete(`${base}/me/follows/7`, () => {
        hit = true
        return new HttpResponse(null, { status: 204 })
      }),
    )
    renderWithProviders(<FollowButton userId={7} state="accepted" />)
    await userEvent.click(screen.getByRole('button', { name: 'Вы подписаны' }))
    await waitFor(() => expect(hit).toBe(true))
  })
})
