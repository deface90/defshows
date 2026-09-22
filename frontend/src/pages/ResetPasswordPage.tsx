import { Anchor, Card, Center, Stack, Text } from '@mantine/core'
import { Link } from 'react-router-dom'
import { ResetPasswordForm } from '@/features/auth/ResetPasswordForm'
import { useDocumentTitle } from '@/shared/lib/useDocumentTitle'

export function ResetPasswordPage() {
  useDocumentTitle('Новый пароль')
  return (
    <Center mih="100vh" p="md">
      <Card withBorder shadow="md" w={400} maw="100%" padding="xl">
        <Stack>
          <img src="/logo.png" alt="defShows" width={64} height={64} style={{ display: 'block' }} />
          <Text c="dimmed" size="sm">
            Задайте новый пароль
          </Text>
          <ResetPasswordForm />
          <Text size="sm" ta="center">
            <Anchor component={Link} to="/login">
              Вернуться ко входу
            </Anchor>
          </Text>
        </Stack>
      </Card>
    </Center>
  )
}
