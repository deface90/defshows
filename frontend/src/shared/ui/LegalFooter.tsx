import { Anchor, Box, Container, Group } from '@mantine/core'
import { Link } from 'react-router-dom'
import { legalOperator } from '@/shared/legal/operator'

export function LegalFooter() {
  if (!legalOperator) return null

  return (
    <Box component="footer" mt="auto" py="lg" style={{ borderTop: '1px solid var(--mantine-color-default-border)' }}>
      <Container size="lg">
        <Group justify="space-between" gap="sm">
          <Group gap="md">
            <Anchor component={Link} to="/legal" size="xs">Реквизиты</Anchor>
            <Anchor component={Link} to="/privacy" size="xs">Политика обработки персональных данных</Anchor>
          </Group>
        </Group>
      </Container>
    </Box>
  )
}
