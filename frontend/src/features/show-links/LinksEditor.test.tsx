import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { LinksEditor } from './LinksEditor'

const base = 'http://localhost:8080'

describe('LinksEditor', () => {
  it('shows all empty rows and adds plain text inline', async () => {
    let links: { id: number; kind: string; url: string }[] = []
    server.use(
      http.get(`${base}/me/shows/10/links`, () => HttpResponse.json({ links })),
      http.post(`${base}/me/shows/10/links`, async ({ request }) => {
        const body = await request.json() as { kind: string; url: string }
        links = [{ id: 1, ...body }]
        return HttpResponse.json(links[0], { status: 201 })
      }),
    )
    renderWithProviders(<LinksEditor showId={10} />)
    await screen.findByRole('button', { name: 'Редактировать: Скачать' })
    expect(screen.getAllByText('нет')).toHaveLength(6)
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument()
    expect(screen.queryByRole('combobox')).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Редактировать: Скачать' }))
    await userEvent.type(screen.getByRole('textbox', { name: 'Скачать' }), 'На домашнем диске{Enter}')
    expect(await screen.findByText('На домашнем диске')).toBeInTheDocument()
    expect(screen.queryByRole('link')).not.toBeInTheDocument()
    expect(links[0]).toMatchObject({ kind: 'download', url: 'На домашнем диске' })
  })

  it('updates a URL in place, cancels edits, and clears a row', async () => {
    let links = [{ id: 7, kind: 'wiki', url: 'https://example.com/old' }]
    server.use(
      http.get(`${base}/me/shows/10/links`, () => HttpResponse.json({ links })),
      http.patch(`${base}/me/shows/10/links/7`, async ({ request }) => {
        const body = await request.json() as { url: string }
        links[0].url = body.url
        return new HttpResponse(null, { status: 204 })
      }),
      http.delete(`${base}/me/shows/10/links/7`, () => {
        links = []
        return new HttpResponse(null, { status: 204 })
      }),
    )
    renderWithProviders(<LinksEditor showId={10} dubbing="LostFilm" />)
    expect(await screen.findByRole('link')).toHaveAttribute('href', 'https://example.com/old')
    await userEvent.click(screen.getByRole('button', { name: 'Редактировать: Wiki' }))
    const input = screen.getByRole('textbox', { name: 'Wiki' })
    await userEvent.clear(input)
    await userEvent.type(input, 'https://example.com/new{Enter}')
    await waitFor(() => expect(screen.getByRole('link')).toHaveAttribute('href', 'https://example.com/new'))
    expect(links[0].id).toBe(7)
    await userEvent.click(screen.getByRole('button', { name: 'Редактировать: Wiki' }))
    await userEvent.type(screen.getByRole('textbox', { name: 'Wiki' }), 'draft{Escape}')
    expect(screen.getByRole('link')).toHaveAttribute('href', 'https://example.com/new')
    await userEvent.click(screen.getByRole('button', { name: 'Редактировать: Wiki' }))
    await userEvent.clear(screen.getByRole('textbox', { name: 'Wiki' }))
    await userEvent.click(screen.getByRole('button', { name: 'Сохранить: Wiki' }))
    await waitFor(() => expect(screen.queryByRole('textbox')).not.toBeInTheDocument())
    expect(screen.queryByRole('link')).not.toBeInTheDocument()
    expect(links).toHaveLength(0)
  })

  it('keeps unsafe URLs as text and preserves the draft on save failure', async () => {
    server.use(
      http.get(`${base}/me/shows/10/links`, () => HttpResponse.json({ links: [
        { id: 8, kind: 'streaming', url: 'javascript:alert(1)' },
      ] })),
      http.patch(`${base}/me/shows/10/links/8`, () => new HttpResponse(null, { status: 500 })),
    )
    renderWithProviders(<LinksEditor showId={10} />)
    expect(await screen.findByText('javascript:alert(1)')).toBeInTheDocument()
    expect(screen.queryByRole('link')).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Редактировать: Смотреть' }))
    await userEvent.clear(screen.getByRole('textbox', { name: 'Смотреть' }))
    await userEvent.type(screen.getByRole('textbox', { name: 'Смотреть' }), 'new value{Enter}')
    expect(await screen.findByText('Не удалось сохранить: Смотреть')).toBeInTheDocument()
    expect(screen.getByRole('textbox', { name: 'Смотреть' })).toHaveValue('new value')
  })

  it('edits dubbing without opening a separate form', async () => {
    let body: unknown
    server.use(
      http.get(`${base}/me/shows/10/links`, () => HttpResponse.json({ links: [] })),
      http.patch(`${base}/me/shows/10`, async ({ request }) => {
        body = await request.json()
        return HttpResponse.json({ show_id: 10, preferred_dubbing: 'LostFilm' })
      }),
    )
    renderWithProviders(<LinksEditor showId={10} />)
    await userEvent.click(screen.getByRole('button', { name: 'Редактировать: Озвучка' }))
    await userEvent.type(screen.getByRole('textbox', { name: 'Озвучка' }), 'LostFilm{Enter}')
    await waitFor(() => expect(body).toEqual({ preferred_dubbing: 'LostFilm' }))
    await waitFor(() => expect(screen.queryByRole('textbox')).not.toBeInTheDocument())
  })
})
