// Runtime configuration. In production, public/env.template.js sets
// window.__ENV.API_BASE_URL at container start; in dev it stays an unsubstituted
// "${API_BASE_URL}" placeholder, so we fall back to the Vite env / default.

declare global {
  interface Window {
    __ENV?: { API_BASE_URL?: string; APP_STORE_URL?: string; METRIKA_ID?: string }
  }
}

const DEFAULT_API_BASE_URL = 'http://localhost:8080'

/** getApiBaseUrl resolves the backend base URL. */
export function getApiBaseUrl(): string {
  const runtime = typeof window !== 'undefined' ? window.__ENV?.API_BASE_URL : undefined
  if (runtime && !runtime.startsWith('${')) {
    return runtime
  }
  const fromVite = import.meta.env.VITE_API_BASE_URL as string | undefined
  return fromVite ?? DEFAULT_API_BASE_URL
}

/**
 * getAppStoreUrl resolves the App Store link, or undefined while the iOS app
 * isn't published yet / the var isn't configured — callers should hide the
 * link in that case rather than show a dead one.
 */
export function getAppStoreUrl(): string | undefined {
  const runtime = typeof window !== 'undefined' ? window.__ENV?.APP_STORE_URL : undefined
  if (runtime && !runtime.startsWith('${')) {
    return runtime
  }
  return import.meta.env.VITE_APP_STORE_URL as string | undefined
}

/**
 * getMetrikaId resolves the Yandex Metrika counter id, or undefined when
 * analytics is not configured (dev, tests, staging) — then nothing is loaded.
 */
export function getMetrikaId(): number | undefined {
  const runtime = typeof window !== 'undefined' ? window.__ENV?.METRIKA_ID : undefined
  const raw = runtime && !runtime.startsWith('${') ? runtime : (import.meta.env.VITE_METRIKA_ID as string | undefined)
  const id = Number(raw)
  return raw && Number.isInteger(id) && id > 0 ? id : undefined
}

export {}
