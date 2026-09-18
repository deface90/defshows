import { Alert, Anchor, Card, Center, Loader, Stack, Text } from '@mantine/core'
import { useEffect, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { oauthExchange } from '@/shared/api/auth/endpoints'
import { applySession } from '@/shared/auth/session'

export function OAuthCallbackPage() {
  const [params] = useSearchParams()
  const navigate = useNavigate()
  const [error, setError] = useState('')

  useEffect(() => {
    const err = params.get('error')
    const code = params.get('code')
    if (err) {
      setError('Не удалось войти через провайдера. Попробуйте ещё раз.')
      return
    }
    if (!code) {
      setError('Отсутствует код авторизации.')
      return
    }
    oauthExchange({ code })
      .then((res) => {
        applySession(res)
        navigate('/', { replace: true })
      })
      .catch(() => setError('Ссылка авторизации недействительна или истекла.'))
  }, [params, navigate])

  return (
    <Center mih="100vh" p="md">
      <Card withBorder w={400} maw="100%" padding="xl">
        {error ? (
          <Stack>
            <Alert color="red">{error}</Alert>
            <Text size="sm" ta="center">
              <Anchor component={Link} to="/login">
                Вернуться ко входу
              </Anchor>
            </Text>
          </Stack>
        ) : (
          <Stack align="center">
            <Loader />
            <Text c="dimmed">Завершаем вход…</Text>
          </Stack>
        )}
      </Card>
    </Center>
  )
}
