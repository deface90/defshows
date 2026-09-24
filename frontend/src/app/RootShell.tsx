import { Outlet } from 'react-router-dom'
import { AuthModal } from '@/features/auth/AuthModal'
import { CookieBanner } from './CookieBanner'
import { MetrikaTracker } from './MetrikaTracker'

/**
 * RootShell wraps every route so the global AuthModal is always mounted inside
 * the router context, keeping the current page in place while a guest signs in.
 * MetrikaTracker and CookieBanner live here too, since they need the router.
 */
export function RootShell() {
  return (
    <>
      <Outlet />
      <AuthModal />
      <MetrikaTracker />
      <CookieBanner />
    </>
  )
}
