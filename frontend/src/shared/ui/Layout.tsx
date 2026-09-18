import {
  ActionIcon,
  Box,
  Container,
  Group,
  Menu,
  useComputedColorScheme,
  useMantineColorScheme,
} from '@mantine/core'
import { NavLink, Outlet, useNavigate } from 'react-router-dom'
import { QueryErrorBoundary } from '@/app/QueryErrorBoundary'
import { logout } from '@/shared/api/auth/endpoints'
import { getRefreshToken, useAuthStore } from '@/shared/auth/authStore'

const navItems = [
  { to: '/', label: 'Мои сериалы', end: true },
  { to: '/search', label: 'Поиск' },
  { to: '/discover', label: 'Подбор' },
  { to: '/users', label: 'Пользователи' },
  { to: '/notifications', label: 'Уведомления' },
]

export function Layout() {
  const navigate = useNavigate()
  const user = useAuthStore((s) => s.user)
  const clear = useAuthStore((s) => s.clear)
  const { setColorScheme } = useMantineColorScheme()
  const computed = useComputedColorScheme('dark')

  const onLogout = async () => {
    const rt = getRefreshToken()
    if (rt) {
      try {
        await logout({ refresh_token: rt })
      } catch {
        // best effort
      }
    }
    clear()
    navigate('/login', { replace: true })
  }

  return (
    <Box>
      <Box
        component="header"
        style={{
          position: 'sticky',
          top: 0,
          zIndex: 20,
          borderBottom: '1px solid var(--mantine-color-default-border)',
          backdropFilter: 'blur(8px)',
        }}
      >
        <Container size="lg">
          <Group h={56} justify="space-between">
            <Group gap="xl">
              <NavLink to="/" style={{ display: 'flex', alignItems: 'center' }} aria-label="defShows">
                <img src="/logo.png" alt="defShows" style={{ height: 32, display: 'block' }} />
              </NavLink>
              <Group gap="xl">
                {navItems.map((item) => (
                  <NavLink
                    key={item.to}
                    to={item.to}
                    end={item.end}
                    className={({ isActive }) => `app-navlink${isActive ? ' active' : ''}`}
                  >
                    {item.label}
                  </NavLink>
                ))}
              </Group>
            </Group>
            <Group>
              <ActionIcon
                variant="default"
                aria-label="Переключить тему"
                onClick={() => setColorScheme(computed === 'dark' ? 'light' : 'dark')}
              >
                {computed === 'dark' ? '🌞' : '🌙'}
              </ActionIcon>
              <Menu withinPortal>
                <Menu.Target>
                  <ActionIcon variant="default" aria-label="Меню пользователя">
                    👤
                  </ActionIcon>
                </Menu.Target>
                <Menu.Dropdown>
                  <Menu.Label>{user?.email ?? user?.display_name ?? 'Аккаунт'}</Menu.Label>
                  {user?.role === 'admin' && (
                    <Menu.Item onClick={() => navigate('/admin')}>Админка</Menu.Item>
                  )}
                  <Menu.Item onClick={() => navigate('/settings')}>Настройки</Menu.Item>
                  <Menu.Item color="red" onClick={onLogout}>
                    Выйти
                  </Menu.Item>
                </Menu.Dropdown>
              </Menu>
            </Group>
          </Group>
        </Container>
      </Box>
      <Container size="lg" py="md">
        <QueryErrorBoundary>
          <Outlet />
        </QueryErrorBoundary>
      </Container>
    </Box>
  )
}
