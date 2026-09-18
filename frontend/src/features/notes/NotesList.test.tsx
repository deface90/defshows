import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import type { Note } from '@/shared/api/notes/model'
import { NotesList } from './NotesList'

const base = 'http://localhost:8080'

function noteHandlers(initial: Note[]) {
  const notes = [...initial]
  return [
    http.get(`${base}/me/notes`, () => HttpResponse.json({ notes })),
    http.post(`${base}/me/notes`, async ({ request }) => {
      const body = (await request.json()) as Note
      const created: Note = { ...body, id: 99, created_at: '2026-09-16T00:00:00Z' }
      notes.push(created)
      return HttpResponse.json(created, { status: 201 })
    }),
    http.delete(`${base}/me/notes/:id`, ({ params }) => {
      const idx = notes.findIndex((n) => n.id === Number(params.id))
      if (idx >= 0) notes.splice(idx, 1)
      return new HttpResponse(null, { status: 204 })
    }),
  ]
}

describe('NotesList', () => {
  it('renders existing notes', async () => {
    server.use(
      ...noteHandlers([
        { id: 1, show_id: 5, scope: 'show', body: 'Люблю этот сериал', created_at: '2026-09-01' },
      ]),
    )
    renderWithProviders(<NotesList showId={5} />)
    expect(await screen.findByText('Люблю этот сериал')).toBeInTheDocument()
  })

  it('shows an empty state when there are no notes', async () => {
    server.use(...noteHandlers([]))
    renderWithProviders(<NotesList showId={5} />)
    expect(await screen.findByText('Заметок пока нет')).toBeInTheDocument()
  })

  it('creates a note and refreshes the list', async () => {
    let posted: unknown
    server.use(...noteHandlers([]))
    server.use(
      http.post(`${base}/me/notes`, async ({ request }) => {
        posted = await request.json()
        return HttpResponse.json(
          { id: 99, show_id: 5, scope: 'show', body: 'Новая заметка', created_at: '2026-09-16' },
          { status: 201 },
        )
      }),
      http.get(`${base}/me/notes`, () =>
        HttpResponse.json({
          notes: posted
            ? [{ id: 99, show_id: 5, scope: 'show', body: 'Новая заметка', created_at: '2026-09-16' }]
            : [],
        }),
      ),
    )
    renderWithProviders(<NotesList showId={5} />)

    await userEvent.type(screen.getByLabelText('Текст заметки'), 'Новая заметка')
    await userEvent.click(screen.getByRole('button', { name: 'Добавить заметку' }))

    await waitFor(() => expect(posted).toMatchObject({ show_id: 5, scope: 'show', body: 'Новая заметка' }))
    expect(await screen.findByText('Новая заметка')).toBeInTheDocument()
  })

  it('deletes a note', async () => {
    let deleted = false
    server.use(
      http.get(`${base}/me/notes`, () =>
        HttpResponse.json({
          notes: deleted
            ? []
            : [{ id: 7, show_id: 5, scope: 'show', body: 'Удалить меня', created_at: '2026-09-01' }],
        }),
      ),
      http.delete(`${base}/me/notes/7`, () => {
        deleted = true
        return new HttpResponse(null, { status: 204 })
      }),
    )
    renderWithProviders(<NotesList showId={5} />)

    const card = (await screen.findByText('Удалить меня')).closest('.mantine-Card-root') as HTMLElement
    await userEvent.click(within(card).getByLabelText('Удалить заметку'))

    await waitFor(() => expect(deleted).toBe(true))
    await waitFor(() => expect(screen.queryByText('Удалить меня')).not.toBeInTheDocument())
  })
})
