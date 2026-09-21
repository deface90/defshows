import { DiscoverPage } from '@/pages/DiscoverPage'
import { CatalogShowPage } from '@/pages/CatalogShowPage'
import { createBrowserRouter } from 'react-router-dom'
import { AdminDubbingPage } from '@/pages/admin/AdminDubbingPage'
import { LoginPage } from '@/pages/LoginPage'
import { OAuthCallbackPage } from '@/pages/OAuthCallbackPage'
import { HomePage } from '@/pages/HomePage'
import { LegalPage } from '@/pages/LegalPage'
import { PrivacyPage } from '@/pages/PrivacyPage'
import { MyShowsPage } from '@/pages/MyShowsPage'
import { NotFoundPage } from '@/pages/NotFoundPage'
import { NotificationsPage } from '@/pages/NotificationsPage'
import { RegisterPage } from '@/pages/RegisterPage'
import { SearchPage } from '@/pages/SearchPage'
import { SettingsPage } from '@/pages/SettingsPage'
import { ShowDetailPage } from '@/pages/ShowDetailPage'
import { UnwatchedPage } from '@/pages/UnwatchedPage'
import { UserProfilePage } from '@/pages/UserProfilePage'
import { UsersPage } from '@/pages/UsersPage'
import { Layout } from '@/shared/ui/Layout'
import { RootShell } from './RootShell'
import { RequireAuth, RequireRole } from './guards'

export const router = createBrowserRouter([
  {
    element: <RootShell />,
    children: [
      { path: '/login', element: <LoginPage /> },
      { path: '/register', element: <RegisterPage /> },
      { path: '/auth/callback', element: <OAuthCallbackPage /> },
      {
        path: '/',
        element: <Layout />,
        children: [
          // Public routes: guests and users. `/` is the smart home (landing vs
          // dashboard); the catalog is browsable without an account.
          { index: true, element: <HomePage /> },
          { path: 'search', element: <SearchPage /> },
          { path: 'discover', element: <DiscoverPage /> },
          { path: 'catalog/:tmdbId', element: <CatalogShowPage /> },
          // Legal pages must be publicly reachable (required for vc.ru etc.).
          { path: 'legal', element: <LegalPage /> },
          { path: 'privacy', element: <PrivacyPage /> },
          // Private routes: everything bound to a user's account.
          {
            element: <RequireAuth />,
            children: [
              { path: 'my', element: <MyShowsPage /> },
              { path: 'unwatched', element: <UnwatchedPage /> },
              { path: 'shows/:id', element: <ShowDetailPage /> },
              { path: 'users', element: <UsersPage /> },
              { path: 'users/:id', element: <UserProfilePage /> },
              { path: 'notifications', element: <NotificationsPage /> },
              { path: 'settings', element: <SettingsPage /> },
              {
                element: <RequireRole role="admin" />,
                children: [{ path: 'admin', element: <AdminDubbingPage /> }],
              },
            ],
          },
          { path: '*', element: <NotFoundPage /> },
        ],
      },
    ],
  },
])
