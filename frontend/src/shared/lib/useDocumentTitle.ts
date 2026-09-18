import { useEffect } from 'react'

export const APP_NAME = 'defShows'

/**
 * useDocumentTitle sets `document.title` to "<title> · defShows" (or just the
 * app name when title is empty/undefined — e.g. while a page's data loads).
 * Re-runs whenever the title changes, so dynamic titles (a show's name) update
 * once the data arrives.
 */
export function useDocumentTitle(title?: string | null): void {
  useEffect(() => {
    document.title = title ? `${title} · ${APP_NAME}` : APP_NAME
  }, [title])
}
