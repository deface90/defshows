import { Anchor, Box, Container, Group } from '@mantine/core'
import { Link } from 'react-router-dom'
import { useAnalyticsConsent } from '@/shared/lib/analyticsConsent'
import { getAppStoreUrl, getMetrikaId } from '@/shared/lib/env'
import { legalOperator } from '@/shared/legal/operator'

export function LegalFooter() {
  const appStoreUrl = getAppStoreUrl()
  const resetConsent = useAnalyticsConsent((s) => s.reset)

  if (!legalOperator && !appStoreUrl) return null

  return (
    <Box component="footer" mt="auto" py="lg" style={{ borderTop: '1px solid var(--mantine-color-default-border)' }}>
      <Container size="lg">
        <Group justify="space-between" gap="sm">
          {legalOperator ? (
            <Group gap="md">
              <Anchor component={Link} to="/legal" size="xs">Реквизиты</Anchor>
              <Anchor component={Link} to="/privacy" size="xs">Политика обработки персональных данных</Anchor>
              {getMetrikaId() && (
                <Anchor component="button" type="button" size="xs" onClick={resetConsent}>Настройки cookie</Anchor>
              )}
            </Group>
          ) : (
            <span />
          )}
          {appStoreUrl && (
            <Anchor href={appStoreUrl} target="_blank" rel="noopener noreferrer" size="xs">
              defShows для iPhone — App Store
            </Anchor>
          )}
        </Group>
      </Container>
    </Box>
  )
}
