import { create } from 'zustand'

// Analytics (Yandex Metrika) runs only after the visitor explicitly accepts it
// in CookieBanner. The decision is kept per browser in localStorage.

const CONSENT_KEY = 'defshows_analytics_consent'

export type AnalyticsConsent = 'granted' | 'denied' | null

function readConsent(): AnalyticsConsent {
  try {
    const value = localStorage.getItem(CONSENT_KEY)
    return value === 'granted' || value === 'denied' ? value : null
  } catch {
    return null
  }
}

function writeConsent(value: AnalyticsConsent): void {
  try {
    if (value) localStorage.setItem(CONSENT_KEY, value)
    else localStorage.removeItem(CONSENT_KEY)
  } catch {
    // Storage unavailable (private mode): the choice lasts for this page only.
  }
}

/** Removes the cookies Metrika sets on our domain (best effort). */
function clearMetrikaCookies(): void {
  for (const pair of document.cookie.split(';')) {
    const name = pair.split('=')[0].trim()
    if (!name.startsWith('_ym')) continue
    const expired = `${name}=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/`
    document.cookie = expired
    document.cookie = `${expired}; domain=${window.location.hostname}`
    document.cookie = `${expired}; domain=.${window.location.hostname}`
  }
}

interface ConsentState {
  consent: AnalyticsConsent
  grant: () => void
  deny: () => void
  /** reset forgets the decision so CookieBanner asks again. */
  reset: () => void
}

export const useAnalyticsConsent = create<ConsentState>((set) => ({
  consent: readConsent(),
  grant: () => {
    writeConsent('granted')
    set({ consent: 'granted' })
  },
  deny: () => {
    writeConsent('denied')
    set({ consent: 'denied' })
    clearMetrikaCookies()
    // A loaded Metrika tag cannot be unloaded; reload so it stops immediately.
    if (window.ym) window.location.reload()
  },
  reset: () => {
    writeConsent(null)
    set({ consent: null })
  },
}))
