import { Anchor, Card, Center, Divider, Stack, Text } from '@mantine/core'
import { Link } from 'react-router-dom'
import { LoginForm } from '@/features/auth/LoginForm'
// import { OAuthButtons } from '@/features/auth/OAuthButtons'

export function LoginPage() {
  return (
    <Center mih="100vh" p="md">
      <Card withBorder shadow="md" w={400} maw="100%" padding="xl">
        <Stack>
          <img
            src="/logo.png"
            alt="defShows"
            width={64}
            height={64}
            style={{ display: 'block' }}
          />
          <Text c="dimmed" size="sm">
            Вход в трекер сериалов
          </Text>
          <LoginForm />
          <Divider label="или" labelPosition="center" />
          {/*<OAuthButtons />*/}
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
