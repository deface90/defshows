import { render, waitFor } from '@testing-library/react'
import { MemoryRouter, useNavigate } from 'react-router-dom'
import { afterEach, describe, expect, it } from 'vitest'
import { useAnalyticsConsent } from '@/shared/lib/analyticsConsent'
import { MetrikaTracker } from './MetrikaTracker'

function Nav({ to }: { to?: string }) {
  const navigate = useNavigate()
  return to ? <button onClick={() => navigate(to)}>go</button> : null
}

const calls = () => (window.ym?.a ?? []).map((args) => args[1])

describe('MetrikaTracker', () => {
  afterEach(() => {
    delete window.__ENV
    delete window.ym
    useAnalyticsConsent.setState({ consent: null })
    localStorage.clear()
    document.head.querySelectorAll('script[src*="mc.yandex.ru"]').forEach((s) => s.remove())
  })

  it('loads nothing without a configured counter', async () => {
    render(<MemoryRouter><MetrikaTracker /></MemoryRouter>)
    await new Promise((r) => setTimeout(r, 10))
    expect(window.ym).toBeUndefined()
    expect(document.head.querySelector('script[src*="mc.yandex.ru"]')).toBeNull()
  })

  it('loads nothing until the visitor consents', async () => {
    window.__ENV = { METRIKA_ID: '42' }
    render(<MemoryRouter><MetrikaTracker /></MemoryRouter>)
    await new Promise((r) => setTimeout(r, 10))
    expect(window.ym).toBeUndefined()
  })

  it('inits once and reports a hit per navigation', async () => {
    window.__ENV = { METRIKA_ID: '42' }
    useAnalyticsConsent.getState().grant()
    const { getByText } = render(
      <MemoryRouter initialEntries={['/']}>
        <MetrikaTracker />
        <Nav to="/search" />
      </MemoryRouter>,
    )
    await waitFor(() => expect(calls()).toEqual(['init', 'hit']))
    expect(document.head.querySelectorAll('script[src="https://mc.yandex.ru/metrika/tag.js?id=42"]')).toHaveLength(1)

    getByText('go').click()
    await waitFor(() => expect(calls()).toEqual(['init', 'hit', 'hit']))
  })
})
