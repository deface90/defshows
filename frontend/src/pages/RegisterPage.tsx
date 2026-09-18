import { Anchor, Card, Center, Stack, Text, Title } from '@mantine/core'
import { Link } from 'react-router-dom'
import { RegisterForm } from '@/features/auth/RegisterForm'

export function RegisterPage() {
  return (
    <Center mih="100vh" p="md">
      <Card withBorder shadow="md" w={400} maw="100%" padding="xl">
        <Stack>
          <Title order={2} c="brand">
            defShows
          </Title>
          <Text c="dimmed" size="sm">
            Регистрация
          </Text>
          <RegisterForm />
          <Text size="sm" ta="center">
            Уже есть аккаунт?{' '}
            <Anchor component={Link} to="/login">
              Войти
            </Anchor>
          </Text>
        </Stack>
      </Card>
    </Center>
  )
}
