import { Anchor, Card, Center, Divider, Stack, Text, Title } from '@mantine/core'
import { Link } from 'react-router-dom'
import { LoginForm } from '@/features/auth/LoginForm'
import { OAuthButtons } from '@/features/auth/OAuthButtons'

export function LoginPage() {
  return (
    <Center mih="100vh" p="md">
      <Card withBorder shadow="md" w={400} maw="100%" padding="xl">
        <Stack>
          <Title order={2} c="brand">
            defShows
          </Title>
          <Text c="dimmed" size="sm">
            Вход в трекер сериалов
          </Text>
          <LoginForm />
          <Divider label="или" labelPosition="center" />
          <OAuthButtons />
          <Text size="sm" ta="center">
            Нет аккаунта?{' '}
            <Anchor component={Link} to="/register">
              Регистрация
            </Anchor>
          </Text>
        </Stack>
      </Card>
    </Center>
  )
}
