import { useAuthStore } from '@/shared/auth/authStore'
import { LandingView } from './home/LandingView'
import { DashboardView } from './home/DashboardView'

/**
 * HomePage is the smart root: guests see the marketing landing, authenticated
 * users see their personal dashboard. Both live at `/` so a shared link to the
 * home page works for everyone.
 */
export function HomePage() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  return isAuthenticated ? <DashboardView /> : <LandingView />
}
