import { Modal, Tabs, Text } from '@mantine/core'
import { useAuthGate, type AuthTab } from '@/shared/auth/authGate'
import { LoginForm } from './LoginForm'
import { RegisterForm } from './RegisterForm'

/**
 * AuthModal is the global sign-in gate. It opens when a guest triggers an action
 * that needs an account (via useRequireAuth); on success it replays the pending
 * action in place through consumeIntent, so the guest never loses context.
 * Mounted once at the app root.
 */
export function AuthModal() {
  const isOpen = useAuthGate((s) => s.isOpen)
  const tab = useAuthGate((s) => s.tab)
  const setTab = useAuthGate((s) => s.setTab)
  const close = useAuthGate((s) => s.close)
  const consumeIntent = useAuthGate((s) => s.consumeIntent)

  return (
    <Modal opened={isOpen} onClose={close} centered title="Нужен аккаунт" radius="md">
      <Text c="dimmed" size="sm" mb="md">
        Войдите или зарегистрируйтесь, чтобы сохранить это действие и вести свой прогресс.
      </Text>
      <Tabs value={tab} onChange={(v) => v && setTab(v as AuthTab)} keepMounted={false}>
        <Tabs.List grow mb="md">
          <Tabs.Tab value="login">Вход</Tabs.Tab>
          <Tabs.Tab value="register">Регистрация</Tabs.Tab>
        </Tabs.List>
        <Tabs.Panel value="login">
          <LoginForm onAuthed={consumeIntent} />
        </Tabs.Panel>
        <Tabs.Panel value="register">
          <RegisterForm onAuthed={consumeIntent} />
        </Tabs.Panel>
      </Tabs>
    </Modal>
  )
}
