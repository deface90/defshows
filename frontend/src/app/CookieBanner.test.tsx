import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it } from 'vitest'
import { useAnalyticsConsent } from '@/shared/lib/analyticsConsent'
import { renderWithProviders } from '@/test/render'
import { CookieBanner } from './CookieBanner'

describe('CookieBanner', () => {
  afterEach(() => {
    delete window.__ENV
    useAnalyticsConsent.setState({ consent: null })
    localStorage.clear()
  })

  it('is hidden when analytics is not configured', () => {
    renderWithProviders(<CookieBanner />)
    expect(screen.queryByRole('dialog', { name: 'Согласие на cookie' })).toBeNull()
  })

  it('stores acceptance and hides itself', async () => {
    window.__ENV = { METRIKA_ID: '42' }
    renderWithProviders(<CookieBanner />)
    await userEvent.click(screen.getByRole('button', { name: 'Принять' }))
    expect(useAnalyticsConsent.getState().consent).toBe('granted')
    expect(localStorage.getItem('defshows_analytics_consent')).toBe('granted')
    expect(screen.queryByRole('dialog', { name: 'Согласие на cookie' })).toBeNull()
  })

  it('stores refusal and hides itself', async () => {
    window.__ENV = { METRIKA_ID: '42' }
    renderWithProviders(<CookieBanner />)
    await userEvent.click(screen.getByRole('button', { name: 'Отклонить' }))
    expect(useAnalyticsConsent.getState().consent).toBe('denied')
    expect(screen.queryByRole('dialog', { name: 'Согласие на cookie' })).toBeNull()
  })

  it('asks again after the decision is reset', async () => {
    window.__ENV = { METRIKA_ID: '42' }
    useAnalyticsConsent.getState().deny()
    renderWithProviders(<CookieBanner />)
    expect(screen.queryByRole('dialog', { name: 'Согласие на cookie' })).toBeNull()
    useAnalyticsConsent.getState().reset()
    expect(await screen.findByRole('dialog', { name: 'Согласие на cookie' })).toBeInTheDocument()
  })
})
