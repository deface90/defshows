import {
  ActionIcon,
  Box,
  Burger,
  Button,
  Container,
  Drawer,
  Group,
  Menu,
  Stack,
  useComputedColorScheme,
  useMantineColorScheme,
} from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { Link, NavLink, Outlet, useNavigate } from 'react-router-dom'
import { QueryErrorBoundary } from '@/app/QueryErrorBoundary'
import { logout } from '@/shared/api/auth/endpoints'
import { getRefreshToken, useAuthStore } from '@/shared/auth/authStore'
import { LegalFooter } from './LegalFooter'

// Public nav is shown to everyone; the catalog is browsable without an account.
const publicNavItems = [
  { to: '/search', label: 'Поиск' },
  { to: '/discover', label: 'Подбор' },
]

// Authenticated users additionally get their account-bound sections.
const userNavItems = [
  { to: '/my', label: 'Мои сериалы' },
  ...publicNavItems,
  { to: '/users', label: 'Пользователи' },
  { to: '/notifications', label: 'Уведомления' },
]

export function Layout() {
  const navigate = useNavigate()
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const user = useAuthStore((s) => s.user)
  const clear = useAuthStore((s) => s.clear)
  const { setColorScheme } = useMantineColorScheme()
  const computed = useComputedColorScheme('dark')
  const [drawerOpen, drawer] = useDisclosure(false)

  const navItems = isAuthenticated ? userNavItems : publicNavItems

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
    <Box style={{ display: 'flex', flexDirection: 'column', minHeight: '100dvh' }}>
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
          <Group h={56} justify="space-between" wrap="nowrap">
            <Group gap="xl" wrap="nowrap">
              <NavLink to="/" style={{ display: 'flex', alignItems: 'center' }} aria-label="defShows">
                <img src="/logo.png" alt="defShows" style={{ height: 32, display: 'block' }} />
              </NavLink>
              {/* Full inline nav on wider screens; collapses into the burger below md. */}
              <Group gap="lg" wrap="nowrap" visibleFrom="md">
                {navItems.map((item) => (
                  <NavLink
                    key={item.to}
                    to={item.to}
                    className={({ isActive }) => `app-navlink${isActive ? ' active' : ''}`}
                  >
                    {item.label}
                  </NavLink>
                ))}
              </Group>
            </Group>
            <Group gap="sm" wrap="nowrap">
              <ActionIcon
                variant="default"
                aria-label="Переключить тему"
                onClick={() => setColorScheme(computed === 'dark' ? 'light' : 'dark')}
              >
                {computed === 'dark' ? '🌞' : '🌙'}
              </ActionIcon>
              {isAuthenticated ? (
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
              ) : (
                <Group gap="xs" wrap="nowrap" visibleFrom="xs">
                  <Button component={Link} to="/login" variant="subtle" size="compact-sm">
                    Войти
                  </Button>
                  <Button component={Link} to="/register" color="orange" size="compact-sm">
                    Начать
                  </Button>
                </Group>
              )}
              <Burger
                opened={drawerOpen}
                onClick={drawer.toggle}
                hiddenFrom="md"
                size="sm"
                aria-label="Навигация"
              />
            </Group>
          </Group>
        </Container>
      </Box>

      <Drawer
        opened={drawerOpen}
        onClose={drawer.close}
        position="right"
        size="70%"
        title="Навигация"
        hiddenFrom="md"
      >
        <Stack gap="xs">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              onClick={drawer.close}
              className={({ isActive }) => `app-drawer-link${isActive ? ' active' : ''}`}
            >
              {item.label}
            </NavLink>
          ))}
          {!isAuthenticated && (
            <Group gap="xs" mt="sm">
              <Button component={Link} to="/login" variant="default" onClick={drawer.close}>
                Войти
              </Button>
              <Button component={Link} to="/register" color="orange" onClick={drawer.close}>
                Начать
              </Button>
            </Group>
          )}
        </Stack>
      </Drawer>

      <Container size="lg" py="md" w="100%">
        <QueryErrorBoundary>
          <Outlet />
        </QueryErrorBoundary>
      </Container>

      <LegalFooter />
    </Box>
  )
}
