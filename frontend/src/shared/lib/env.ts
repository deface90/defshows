// Runtime configuration. In production, public/env.template.js sets
// window.__ENV.API_BASE_URL at container start; in dev it stays an unsubstituted
// "${API_BASE_URL}" placeholder, so we fall back to the Vite env / default.

declare global {
  interface Window {
    __ENV?: { API_BASE_URL?: string }
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

export {}
