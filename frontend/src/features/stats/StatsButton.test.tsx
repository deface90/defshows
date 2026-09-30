import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { StatsButton } from './StatsButton'
import { formatWatchTime } from './formatWatchTime'

const base = 'http://localhost:8080'

describe('formatWatchTime', () => {
  it('formats minutes into days/hours/minutes', () => {
    expect(formatWatchTime(0)).toBe('0 мин')
    expect(formatWatchTime(45)).toBe('45 мин')
    expect(formatWatchTime(90)).toBe('1 ч 30 мин')
    expect(formatWatchTime(60 * 24)).toBe('1 д')
    expect(formatWatchTime(60 * 24 + 150)).toBe('1 д 2 ч')
  })
})

describe('StatsButton', () => {
  it('fetches and renders stats only after the modal opens', async () => {
    let calls = 0
    server.use(
      http.get(`${base}/me/stats`, () => {
        calls++
        return HttpResponse.json({
          shows_tracked: 12,
          shows_completed: 4,
          seasons_watched: 9,
          episodes_watched: 120,
          minutes_watched: 3600,
        })
      }),
    )
    renderWithProviders(<StatsButton />)

    // Deferred: nothing is requested until the button is clicked.
    expect(calls).toBe(0)

    await userEvent.click(screen.getByRole('button', { name: /Статистика/ }))

    expect(await screen.findByText('Добавлено сериалов')).toBeInTheDocument()
    expect(screen.getByText('120')).toBeInTheDocument()
    // 3600 minutes = 2 days 12 hours.
    expect(screen.getByText('2 д 12 ч')).toBeInTheDocument()
    expect(calls).toBe(1)
  })
})
