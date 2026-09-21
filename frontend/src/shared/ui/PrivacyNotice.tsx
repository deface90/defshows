import { Anchor, Text } from '@mantine/core'
import { legalOperator } from '@/shared/legal/operator'

export function PrivacyNotice() {
  if (!legalOperator) return null

  return (
    <Text size="xs" c="dimmed">
      О том, как сервис использует ваши данные, — в{' '}
      <Anchor href="/privacy" target="_blank" rel="noopener noreferrer" size="xs">
        политике обработки персональных данных
      </Anchor>.
    </Text>
  )
}
