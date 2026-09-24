// Yandex Metrika for the SPA: the counter is initialised with defer:true so it
// does not report the initial load itself; every route (including the first)
// is reported via usePageviews, keeping one hit per navigation. Webvisor is off.

type Ym = ((id: number, method: string, ...args: unknown[]) => void) & { a?: unknown[][]; l?: number }

declare global {
  interface Window {
    ym?: Ym
  }
}

const TAG_URL = 'https://mc.yandex.ru/metrika/tag.js'

/** initMetrika injects the official tag and initialises the counter once. */
export function initMetrika(id: number): void {
  if (window.ym) return
  // Queueing stub from the official snippet: calls made before tag.js loads are replayed.
  const ym: Ym = (...args) => {
    ym.a = ym.a || []
    ym.a.push(args)
  }
  ym.l = Date.now()
  window.ym = ym

  const script = document.createElement('script')
  script.async = true
  script.src = `${TAG_URL}?id=${id}`
  document.head.appendChild(script)

  ym(id, 'init', {
    defer: true,
    clickmap: true,
    trackLinks: true,
    accurateTrackBounce: true,
  })
}

/** hit reports a pageview for an SPA navigation. */
export function hit(id: number, url: string, referer?: string): void {
  window.ym?.(id, 'hit', url, { referer, title: document.title })
}
