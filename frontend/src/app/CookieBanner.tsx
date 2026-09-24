import { Anchor, Button, Group, Paper, Text } from '@mantine/core'
import { Link } from 'react-router-dom'
import { useAnalyticsConsent } from '@/shared/lib/analyticsConsent'
import { getMetrikaId } from '@/shared/lib/env'

/**
 * CookieBanner asks for consent to web analytics. It is shown only while
 * Metrika is configured and the visitor has not decided yet.
 */
export function CookieBanner() {
  const { consent, grant, deny } = useAnalyticsConsent()
  if (!getMetrikaId() || consent !== null) return null

  return (
    <Paper
      role="dialog"
      aria-label="Согласие на cookie"
      withBorder
      shadow="md"
      p="md"
      radius="md"
      style={{
        position: 'fixed',
        left: 16,
        right: 16,
        bottom: 16,
        maxWidth: 560,
        marginInline: 'auto',
        zIndex: 300,
      }}
    >
      <Text size="sm">
        Мы используем cookie и Яндекс Метрику, чтобы собирать обезличенную статистику посещений.
        Сервис работает и без неё. Подробнее — в{' '}
        <Anchor component={Link} to="/privacy" size="sm">
          политике обработки персональных данных
        </Anchor>
        .
      </Text>
      <Group justify="flex-end" gap="sm" mt="sm">
        <Button variant="default" size="xs" onClick={deny}>
          Отклонить
        </Button>
        <Button size="xs" onClick={grant}>
          Принять
        </Button>
      </Group>
    </Paper>
  )
}
