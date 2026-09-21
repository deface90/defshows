import { Outlet } from 'react-router-dom'
import { AuthModal } from '@/features/auth/AuthModal'

/**
 * RootShell wraps every route so the global AuthModal is always mounted inside
 * the router context, keeping the current page in place while a guest signs in.
 */
export function RootShell() {
  return (
    <>
      <Outlet />
      <AuthModal />
    </>
  )
}
