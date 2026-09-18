import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import type { DubbingStudio } from '@/shared/api/admin/model'
import { AdminDubbingPage } from './AdminDubbingPage'

const base = 'http://localhost:8080'

describe('AdminDubbingPage', () => {
  it('lists studios and creates a new one', async () => {
    let posted: unknown
    const studios: DubbingStudio[] = [{ id: 1, name: 'LostFilm', site_url: 'https://lostfilm.tv', active: true }]
    server.use(
      http.get(`${base}/admin/dubbing-studios`, () => HttpResponse.json({ studios })),
      http.post(`${base}/admin/dubbing-studios`, async ({ request }) => {
        posted = await request.json()
        const created = { id: 2, name: 'HDrezka', active: true }
        studios.push(created)
        return HttpResponse.json(created, { status: 201 })
      }),
    )
    renderWithProviders(<AdminDubbingPage />)

    expect(await screen.findByText('LostFilm')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Добавить' }))
    await userEvent.type(await screen.findByLabelText('Название'), 'HDrezka')
    await userEvent.click(screen.getByRole('button', { name: 'Создать' }))

    await waitFor(() => expect(posted).toMatchObject({ name: 'HDrezka', active: true }))
    expect(await screen.findByText('Сохранено')).toBeInTheDocument()
  })

  it('rejects an empty name', async () => {
    server.use(http.get(`${base}/admin/dubbing-studios`, () => HttpResponse.json({ studios: [] })))
    renderWithProviders(<AdminDubbingPage />)

    await userEvent.click(await screen.findByRole('button', { name: 'Добавить' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Создать' }))

    expect(await screen.findByText('Введите название')).toBeInTheDocument()
  })
})
