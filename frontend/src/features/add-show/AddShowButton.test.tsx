import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { AddShowButton } from './AddShowButton'

const base = 'http://localhost:8080'

describe('AddShowButton', () => {
  it('posts the show and shows a success notification', async () => {
    let body: unknown
    server.use(
      http.post(`${base}/me/shows`, async ({ request }) => {
        body = await request.json()
        return HttpResponse.json({ id: 1, show_id: 10, status: 'watching', favorite: false }, { status: 201 })
      }),
    )
    renderWithProviders(<AddShowButton tmdbId={1399} />)

    await userEvent.click(screen.getByRole('button', { name: 'Добавить' }))

    await waitFor(() => expect(body).toEqual({ tmdb_id: 1399 }))
    expect(await screen.findByText('Добавлено в «Мои сериалы»')).toBeInTheDocument()
  })

  it('notifies when the show is already tracked (409)', async () => {
    server.use(http.post(`${base}/me/shows`, () => new HttpResponse(null, { status: 409 })))
    renderWithProviders(<AddShowButton tmdbId={1} />)

    await userEvent.click(screen.getByRole('button', { name: 'Добавить' }))
    expect(await screen.findByText('Сериал уже добавлен')).toBeInTheDocument()
  })
})
