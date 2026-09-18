import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { WatchToggle } from './WatchToggle'

const base = 'http://localhost:8080'

describe('WatchToggle', () => {
  it('optimistically checks and persists on success', async () => {
    server.use(http.post(`${base}/me/shows/10/episodes/100/watch`, () => new HttpResponse(null, { status: 204 })))
    renderWithProviders(<WatchToggle showId={10} episodeId={100} watched={false} />)

    const box = screen.getByRole('checkbox')
    expect(box).not.toBeChecked()
    await userEvent.click(box)
    expect(box).toBeChecked()
  })

  it('rolls back on error', async () => {
    server.use(http.post(`${base}/me/shows/10/episodes/100/watch`, () => new HttpResponse(null, { status: 500 })))
    renderWithProviders(<WatchToggle showId={10} episodeId={100} watched={false} />)

    const box = screen.getByRole('checkbox')
    await userEvent.click(box)
    await waitFor(() => expect(box).not.toBeChecked())
  })
})
