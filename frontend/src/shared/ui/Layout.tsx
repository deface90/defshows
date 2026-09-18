import {
  ActionIcon,
  Box,
  Burger,
  Container,
  Drawer,
  Group,
  Menu,
  Stack,
  useComputedColorScheme,
  useMantineColorScheme,
} from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
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
  const [drawerOpen, drawer] = useDisclosure(false)

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
                    end={item.end}
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
              end={item.end}
              onClick={drawer.close}
              className={({ isActive }) => `app-drawer-link${isActive ? ' active' : ''}`}
            >
              {item.label}
            </NavLink>
          ))}
        </Stack>
      </Drawer>

      <Container size="lg" py="md">
        <QueryErrorBoundary>
          <Outlet />
        </QueryErrorBoundary>
      </Container>
    </Box>
  )
}
