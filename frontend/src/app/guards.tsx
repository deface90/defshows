import { Navigate, Outlet } from 'react-router-dom'
import { useAuthStore } from '@/shared/auth/authStore'

/** RequireAuth renders child routes only for authenticated users. */
export function RequireAuth() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  return isAuthenticated ? <Outlet /> : <Navigate to="/login" replace />
}

/** RequireRole renders child routes only for users with the given role. */
export function RequireRole({ role }: { role: string }) {
  const user = useAuthStore((s) => s.user)
  if (!user) return <Navigate to="/login" replace />
  return user.role === role ? <Outlet /> : <Navigate to="/" replace />
}
