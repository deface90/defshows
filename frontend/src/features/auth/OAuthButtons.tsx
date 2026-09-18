import { Button, Stack } from '@mantine/core'
import { getApiBaseUrl } from '@/shared/lib/env'

const providers = [
  { key: 'google', label: 'Войти через Google' },
  { key: 'yandex', label: 'Войти через Яндекс' },
  { key: 'vk', label: 'Войти через VK' },
]

export function OAuthButtons() {
  const start = (provider: string) => {
    // Full-page redirect to the backend, which redirects to the provider.
    window.location.href = `${getApiBaseUrl()}/auth/oauth/${provider}`
  }
  return (
    <Stack>
      {providers.map((p) => (
        <Button key={p.key} variant="default" onClick={() => start(p.key)}>
          {p.label}
        </Button>
      ))}
    </Stack>
  )
}
