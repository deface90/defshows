import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { PrefsForm } from './PrefsForm'

const base = 'http://localhost:8080'

const prefs = {
  episode_release: true,
  season_start: false,
  weekly_digest: false,
  channel: 'telegram',
  lead_time_hours: 24,
}

describe('PrefsForm', () => {
  it('loads current prefs and submits both prefs and settings', async () => {
    let prefsBody: unknown
    let settingsBody: unknown
    server.use(
      http.get(`${base}/me/notifications/prefs`, () => HttpResponse.json(prefs)),
      http.get(`${base}/me/settings`, () => HttpResponse.json({ timezone: 'Europe/Moscow' })),
      http.patch(`${base}/me/notifications/prefs`, async ({ request }) => {
        prefsBody = await request.json()
        return HttpResponse.json({ ...prefs, weekly_digest: true })
      }),
      http.patch(`${base}/me/settings`, async ({ request }) => {
        settingsBody = await request.json()
        return HttpResponse.json({ timezone: 'Europe/Berlin' })
      }),
    )
    renderWithProviders(<PrefsForm />)

    // Wait for the loaded value to hydrate the form.
    await waitFor(() => expect((screen.getByLabelText('Таймзона (IANA)') as HTMLInputElement).value).toBe('Europe/Moscow'))

    await userEvent.click(screen.getByLabelText('Недельный дайджест'))
    const tz = screen.getByLabelText('Таймзона (IANA)')
    await userEvent.clear(tz)
    await userEvent.type(tz, 'Europe/Berlin')
    await userEvent.click(screen.getByRole('button', { name: 'Сохранить' }))

    await waitFor(() => expect(prefsBody).toMatchObject({ weekly_digest: true, lead_time_hours: 24 }))
    expect(settingsBody).toEqual({ timezone: 'Europe/Berlin' })
    expect(await screen.findByText('Настройки сохранены')).toBeInTheDocument()
  })
})
