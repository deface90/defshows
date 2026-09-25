import { useEffect, useRef } from 'react'
import { useLocation } from 'react-router-dom'
import { useAnalyticsConsent } from '@/shared/lib/analyticsConsent'
import { getMetrikaId } from '@/shared/lib/env'
import { hit, initMetrika } from '@/shared/lib/metrika'

/**
 * MetrikaTracker loads Yandex Metrika when a counter id is configured and the
 * visitor has accepted analytics, then reports a pageview on every route
 * change. Renders nothing.
 */
export function MetrikaTracker() {
  const location = useLocation()
  const consent = useAnalyticsConsent((s) => s.consent)
  const previousUrl = useRef(document.referrer)

  useEffect(() => {
    const id = getMetrikaId()
    if (!id || consent !== 'granted') return
    initMetrika(id)
    const url = window.location.href
    // Let the page set its document title before the hit is recorded.
    const timer = window.setTimeout(() => {
      hit(id, url, previousUrl.current)
      previousUrl.current = url
    }, 0)
    return () => window.clearTimeout(timer)
  }, [location.pathname, location.search, consent])

  return null
}
