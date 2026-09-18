import { createBrowserRouter } from 'react-router-dom'
import { AdminDubbingPage } from '@/pages/admin/AdminDubbingPage'
import { LoginPage } from '@/pages/LoginPage'
import { OAuthCallbackPage } from '@/pages/OAuthCallbackPage'
import { MyShowsPage } from '@/pages/MyShowsPage'
import { NotFoundPage } from '@/pages/NotFoundPage'
import { NotificationsPage } from '@/pages/NotificationsPage'
import { RegisterPage } from '@/pages/RegisterPage'
import { SearchPage } from '@/pages/SearchPage'
import { SettingsPage } from '@/pages/SettingsPage'
import { ShowDetailPage } from '@/pages/ShowDetailPage'
import { UserProfilePage } from '@/pages/UserProfilePage'
import { UsersPage } from '@/pages/UsersPage'
import { Layout } from '@/shared/ui/Layout'
import { RequireAuth, RequireRole } from './guards'

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  { path: '/register', element: <RegisterPage /> },
  { path: '/auth/callback', element: <OAuthCallbackPage /> },
  {
    element: <RequireAuth />,
    children: [
      {
        path: '/',
        element: <Layout />,
        children: [
          { index: true, element: <MyShowsPage /> },
          { path: 'search', element: <SearchPage /> },
          { path: 'shows/:id', element: <ShowDetailPage /> },
          { path: 'users', element: <UsersPage /> },
          { path: 'users/:id', element: <UserProfilePage /> },
          { path: 'notifications', element: <NotificationsPage /> },
          { path: 'settings', element: <SettingsPage /> },
          {
            element: <RequireRole role="admin" />,
            children: [{ path: 'admin', element: <AdminDubbingPage /> }],
          },
          { path: '*', element: <NotFoundPage /> },
        ],
      },
    ],
  },
])
