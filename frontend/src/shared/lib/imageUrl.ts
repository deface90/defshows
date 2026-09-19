import { getApiBaseUrl } from './env'

/** Route TMDB images through our API; mirrored/local images keep their URL. */
export function imageUrl(src?: string | null): string | undefined {
  if (!src) return undefined
  try {
    const url = new URL(src)
    if (url.hostname === 'image.tmdb.org' && /^https?:$/.test(url.protocol)) {
      const path = url.pathname.match(/^\/t\/p\/([^/]+)\/([^/]+)$/)
      if (path) return `${getApiBaseUrl().replace(/\/$/, '')}/images/tmdb/${path[1]}/${path[2]}`
    }
  } catch {
    // Relative URLs already point to our own image storage.
  }
  return src
}
