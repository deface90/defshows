import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { beforeEach, describe, expect, it } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { DiscoverPage } from './DiscoverPage'

const base = 'http://localhost:8080'
const queries: URLSearchParams[] = []

beforeEach(() => {
  queries.length = 0
  server.use(
    http.get(`${base}/shows/discover/filters`, () => HttpResponse.json({
      genres: [{ id: 18, name: 'Драма' }, { id: 35, name: 'Комедия' }],
      countries: [{ code: 'US', name: 'United States' }, { code: 'GB', name: 'United Kingdom' }],
    })),
    http.get(`${base}/me/shows`, () => HttpResponse.json({ tracked: [
      { show: { id: 10, tmdb_id: 1399 } },
    ] })),
    http.get(`${base}/shows/discover`, ({ request }) => {
      const params = new URL(request.url).searchParams
      queries.push(params)
      return HttpResponse.json({ page: Number(params.get('page')), total_pages: 2, total_results: 21, results: [
        { tmdb_id: 1399, title: 'Game of Thrones' },
      ] })
    }),
  )
})

describe('DiscoverPage', () => {
  it('submits multiple genres and filters only when requested', async () => {
    const user = userEvent.setup()
    renderWithProviders(<DiscoverPage />, { route: '/discover' })
    await screen.findByText('Game of Thrones')
    await waitFor(() => expect(screen.getByRole('combobox', { name: 'Жанры' })).toBeEnabled())
    expect(queries[0].get('votes_min')).toBe('100')
    expect(await screen.findByRole('link', { name: 'В моих сериалах' })).toHaveAttribute('href', '/shows/10')

    await user.click(screen.getByRole('combobox', { name: 'Жанры' }))
    await user.click(await screen.findByRole('option', { name: /Драма/ }))
    await user.click(screen.getByRole('combobox', { name: 'Жанры' }))
    await user.click(await screen.findByRole('option', { name: /Комедия/ }))
    await user.keyboard('{Escape}')
    await user.clear(screen.getByLabelText('Рейтинг TMDB от'))
    await user.type(screen.getByLabelText('Рейтинг TMDB от'), '7.3')
    expect(queries).toHaveLength(1)
    await user.click(screen.getByRole('button', { name: 'Подобрать' }))
    await waitFor(() => expect(queries.at(-1)?.get('genres')).toBe('18|35'))
    expect(queries.at(-1)?.get('rating_min')).toBe('7.3')
    expect(queries.at(-1)?.get('page')).toBe('1')
    await user.click(screen.getByLabelText('Все выбранные'))
    await user.click(screen.getByRole('button', { name: 'Подобрать' }))
    await waitFor(() => expect(queries.at(-1)?.get('genres')).toBe('18,35'))
    await user.click(screen.getByRole('button', { name: '2' }))
    await waitFor(() => expect(queries.at(-1)?.get('page')).toBe('2'))
    expect(queries.at(-1)?.get('genres')).toBe('18,35')
    expect(screen.getAllByRole('link', { name: 'Game of Thrones' })[0].getAttribute('href')).toContain('from=%2Fdiscover')
  })

  it('restores filters and page from the URL and resets unsaved edits', async () => {
    const user = userEvent.setup()
    renderWithProviders(<DiscoverPage />, { route: '/discover?country=US&genres=18,35&rating_min=8&votes_min=500&date_from=2000-01-01&date_to=2020-12-31&page=2' })
    await screen.findByText('Game of Thrones')
    expect(queries[0].get('country')).toBe('US')
    expect(queries[0].get('genres')).toBe('18,35')
    expect(queries[0].get('page')).toBe('2')
    expect(screen.getByLabelText('Премьера с')).toHaveValue('2000-01-01')
    await user.click(screen.getByRole('button', { name: 'Сбросить' }))
    await waitFor(() => expect(queries.at(-1)?.get('page')).toBe('1'))
    expect(screen.getByLabelText('Рейтинг TMDB от')).toHaveValue('0')
    await user.clear(screen.getByLabelText('Рейтинг TMDB от'))
    await user.type(screen.getByLabelText('Рейтинг TMDB от'), '9')
    await user.click(screen.getByRole('button', { name: 'Сбросить' }))
    expect(screen.getByLabelText('Рейтинг TMDB от')).toHaveValue('0')
  })

  it('rejects reversed ranges without sending an invalid request', async () => {
    renderWithProviders(<DiscoverPage />, { route: '/discover?rating_min=9&rating_max=3' })
    expect(await screen.findByText('Проверьте диапазоны рейтинга, дат и количество голосов')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Подобрать' })).toBeDisabled()
    expect(queries).toHaveLength(0)
  })

  it('shows an empty result state', async () => {
    server.use(http.get(`${base}/shows/discover`, () => HttpResponse.json({ page: 1, total_pages: 0, total_results: 0, results: [] })))
    renderWithProviders(<DiscoverPage />, { route: '/discover' })
    expect(await screen.findByText('Сериалы не найдены')).toBeInTheDocument()
  })
})
