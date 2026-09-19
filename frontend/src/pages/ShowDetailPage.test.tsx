import userEvent from '@testing-library/user-event'
import { screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { Route, Routes } from 'react-router-dom'
import { describe, expect, it } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { ShowDetailPage } from './ShowDetailPage'

const base = 'http://localhost:8080'

const show = {
  id: 10,
  tmdb_id: 1399,
  title: 'Game of Thrones',
  airing_status: 'ended',
  ratings: [{ source: 'imdb', value: '9.2/10' }],
  seasons: [
    {
      id: 1,
      season_number: 1,
      name: 'Сезон 1',
      episodes: [
        { id: 100, season_number: 1, episode_number: 1, name: 'Winter Is Coming', air_date: '2011-04-17' },
        { id: 101, season_number: 1, episode_number: 2, name: 'The Kingsroad', air_date: '2011-04-24' },
      ],
    },
  ],
}

function renderDetail() {
  return renderWithProviders(
    <Routes>
      <Route path="/shows/:id" element={<ShowDetailPage />} />
    </Routes>,
    { route: '/shows/10' },
  )
}

describe('ShowDetailPage', () => {
  it('renders detail, ratings and per-episode watched state when tracked', async () => {
    server.use(
      http.get(`${base}/shows/10`, () => HttpResponse.json(show)),
      http.get(`${base}/me/shows/10`, () =>
        HttpResponse.json({
          user_show: { id: 5, show_id: 10, status: 'watching', favorite: false },
          show: { id: 10, tmdb_id: 1399, title: 'Game of Thrones', airing_status: 'ended' },
          progress: { watched: 1, total: 2, watched_episode_ids: [100], next_unwatched_episode_id: 101 },
        }),
      ),
    )
    renderDetail()

    expect(await screen.findByText('Game of Thrones')).toBeInTheDocument()
    expect(screen.getByText('IMDb: 9.2/10')).toBeInTheDocument()
    expect(await screen.findByText('Winter Is Coming')).toBeInTheDocument()

    // Episode 100 (before next unwatched 101) is checked; 101 is not.
    const boxes = await screen.findAllByRole('checkbox', { name: 'Просмотрено' })
    expect(boxes[0]).toBeChecked()
    expect(boxes[1]).not.toBeChecked()

    // Season toggle is indeterminate: 1 of 2 episodes watched.
    const seasonBox = screen.getByRole('checkbox', { name: 'Отметить сезон просмотренным' })
    expect(seasonBox).not.toBeChecked()
    expect(seasonBox).toHaveProperty('indeterminate', true)
  })

  it('shows an Add button when not tracked', async () => {
    server.use(
      http.get(`${base}/shows/10`, () => HttpResponse.json(show)),
      http.get(`${base}/me/shows/10`, () => new HttpResponse(null, { status: 404 })),
    )
    renderDetail()

    expect(await screen.findByText('Game of Thrones')).toBeInTheDocument()
    expect(await screen.findByRole('button', { name: 'Добавить' })).toBeInTheDocument()
  })
})

it('keeps a later season checked independently of the first and updates episode checkboxes', async () => {
  const user = userEvent.setup()
  const watched = new Set<number>()
  const detailed = { ...show, seasons: [show.seasons[0], {
    id: 2, season_number: 2, name: 'Сезон 2',
    episodes: [{ id: 200, season_number: 2, episode_number: 1, name: 'Later episode', air_date: '2012-04-01' }],
  }] }
  server.use(
    http.get(`${base}/shows/10`, () => HttpResponse.json(detailed)),
    http.get(`${base}/me/shows/10`, () => HttpResponse.json({
      user_show: { id: 5, show_id: 10, status: 'watching', favorite: false },
      show,
      progress: { watched: watched.size, total: 3, watched_episode_ids: [...watched], next_unwatched_episode_id: 100 },
    })),
    http.post(`${base}/me/shows/10/episodes/200/watch`, () => {
      watched.add(200)
      return new HttpResponse(null, { status: 204 })
    }),
    http.delete(`${base}/me/shows/10/episodes/200/watch`, () => {
      watched.delete(200)
      return new HttpResponse(null, { status: 204 })
    }),
  )
  renderDetail()
  const seasons = await screen.findAllByRole('checkbox', { name: 'Отметить сезон просмотренным' })
  await user.click(screen.getByRole('button', { name: /Сезон 2/ }))
  await user.click(seasons[1])
  await waitFor(() => expect(seasons[1]).toBeChecked())
  expect(seasons[0]).not.toBeChecked()
  const episodes = screen.getAllByRole('checkbox', { name: 'Просмотрено' })
  expect(episodes[0]).not.toBeChecked()
  expect(episodes[2]).toBeChecked()
  await user.click(seasons[1])
  await waitFor(() => expect(episodes[2]).not.toBeChecked())
  expect(seasons[1]).not.toBeChecked()
})

it('marks the entire show with one write request and refreshes all checkboxes', async () => {
  let completed = false
  let requests = 0
  server.use(
    http.get(`${base}/shows/10`, () => HttpResponse.json(show)),
    http.get(`${base}/me/shows/10`, () => HttpResponse.json({
      user_show: { id: 5, show_id: 10, status: completed ? 'completed' : 'watching', favorite: false },
      show,
      progress: { watched: completed ? 2 : 0, total: 2, watched_episode_ids: completed ? [100, 101] : [], next_unwatched_episode_id: completed ? null : 100 },
    })),
    http.post(`${base}/me/shows/10/watch`, () => {
      requests++
      completed = true
      return new HttpResponse(null, { status: 204 })
    }),
  )
  renderDetail()
  await userEvent.click(await screen.findByRole('button', { name: 'Отметить весь сериал просмотренным' }))
  await waitFor(() => expect(screen.getByRole('button', { name: 'Весь сериал просмотрен ✓' })).toBeDisabled())
  expect(requests).toBe(1)
  for (const box of screen.getAllByRole('checkbox')) expect(box).toBeChecked()
})

it('keeps watch state unchanged when bulk marking fails', async () => {
  server.use(
    http.get(`${base}/shows/10`, () => HttpResponse.json(show)),
    http.get(`${base}/me/shows/10`, () => HttpResponse.json({
      user_show: { id: 5, show_id: 10, status: 'watching', favorite: false },
      show, progress: { watched: 0, total: 2, watched_episode_ids: [], next_unwatched_episode_id: 100 },
    })),
    http.post(`${base}/me/shows/10/watch`, () => new HttpResponse(null, { status: 500 })),
  )
  renderDetail()
  await userEvent.click(await screen.findByRole('button', { name: 'Отметить весь сериал просмотренным' }))
  expect(await screen.findByText('Не удалось отметить сериал просмотренным')).toBeInTheDocument()
  for (const box of screen.getAllByRole('checkbox')) expect(box).not.toBeChecked()
  expect(screen.getByRole('button', { name: 'Отметить весь сериал просмотренным' })).toBeEnabled()
})


it('shows automatic references and removes personal links and dubbing controls', async () => {
  const imdb = 'https://www.imdb.com/title/tt0944947/'
  const wiki = 'https://ru.wikipedia.org/wiki/Game_of_Thrones'
  let personalLinkRequests = 0
  server.use(
    http.get(`${base}/shows/10`, () => HttpResponse.json({ ...show, imdb_url: imdb, wikipedia_url: wiki })),
    http.get(`${base}/me/shows/10`, () => HttpResponse.json({
      user_show: { id: 5, show_id: 10, status: 'watching', favorite: false, preferred_dubbing: 'LostFilm' },
      show, progress: { watched: 0, total: 2, watched_episode_ids: [] },
    })),
    http.get(`${base}/me/shows/10/links`, () => {
      personalLinkRequests++
      return HttpResponse.json({ links: [] })
    }),
  )
  renderDetail()
  expect(await screen.findByRole('link', { name: 'IMDb ↗' })).toHaveAttribute('href', imdb)
  expect(screen.getByRole('link', { name: 'Wikipedia ↗' })).toHaveAttribute('href', wiki)
  expect(await screen.findByRole('tab', { name: 'Заметки' })).toBeInTheDocument()
  expect(screen.queryByRole('tab', { name: 'Ссылки' })).not.toBeInTheDocument()
  expect(screen.queryByText('Озвучка')).not.toBeInTheDocument()
  expect(personalLinkRequests).toBe(0)
})

it('omits reference links when the provider has none', async () => {
  server.use(
    http.get(`${base}/shows/10`, () => HttpResponse.json(show)),
    http.get(`${base}/me/shows/10`, () => new HttpResponse(null, { status: 404 })),
  )
  renderDetail()
  await screen.findByText('Game of Thrones')
  expect(screen.queryByRole('link', { name: 'IMDb ↗' })).not.toBeInTheDocument()
  expect(screen.queryByRole('link', { name: 'Wikipedia ↗' })).not.toBeInTheDocument()
})
