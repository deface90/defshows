import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { RatingControl } from './RatingControl'

const base = 'http://localhost:8080'

describe('RatingControl', () => {
  it('shows the current rating', () => {
    renderWithProviders(<RatingControl showId={10} rating={7} />)
    expect(screen.getByRole('combobox', { name: 'Моя оценка' })).toHaveValue('7')
  })

  it('sends the chosen rating', async () => {
    let body: unknown
    server.use(
      http.patch(`${base}/me/shows/10`, async ({ request }) => {
        body = await request.json()
        return HttpResponse.json({ id: 1, show_id: 10, status: 'watching', favorite: false, rating: 9 })
      }),
    )
    renderWithProviders(<RatingControl showId={10} rating={null} />)
    await userEvent.click(screen.getByRole('combobox', { name: 'Моя оценка' }))
    await userEvent.click(await screen.findByRole('option', { name: '9' }))
    await waitFor(() => expect(body).toEqual({ rating: 9 }))
  })

  it('clears the rating', async () => {
    let body: unknown
    server.use(
      http.patch(`${base}/me/shows/10`, async ({ request }) => {
        body = await request.json()
        return HttpResponse.json({ id: 1, show_id: 10, status: 'watching', favorite: false })
      }),
    )
    renderWithProviders(<RatingControl showId={10} rating={5} />)
    await userEvent.click(screen.getByRole('combobox', { name: 'Моя оценка' }))
    await userEvent.click(await screen.findByRole('option', { name: 'Без оценки' }))
    await waitFor(() => expect(body).toEqual({ clear_rating: true }))
  })
})
