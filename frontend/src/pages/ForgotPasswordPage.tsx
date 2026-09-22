import { Anchor, Card, Center, Stack, Text } from '@mantine/core'
import { Link } from 'react-router-dom'
import { ForgotPasswordForm } from '@/features/auth/ForgotPasswordForm'
import { useDocumentTitle } from '@/shared/lib/useDocumentTitle'

export function ForgotPasswordPage() {
  useDocumentTitle('Восстановление пароля')
  return (
    <Center mih="100vh" p="md">
      <Card withBorder shadow="md" w={400} maw="100%" padding="xl">
        <Stack>
          <img src="/logo.png" alt="defShows" width={64} height={64} style={{ display: 'block' }} />
          <Text c="dimmed" size="sm">
            Восстановление пароля
          </Text>
          <ForgotPasswordForm />
          <Text size="sm" ta="center">
            Вспомнили пароль?{' '}
            <Anchor component={Link} to="/login">
              Войти
            </Anchor>
          </Text>
        </Stack>
      </Card>
    </Center>
  )
}
