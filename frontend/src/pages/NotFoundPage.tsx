import { Button, Center, Stack, Text, Title } from '@mantine/core'
import { Link } from 'react-router-dom'

/** NotFoundPage is the catch-all route for unknown URLs. */
export function NotFoundPage() {
  return (
    <Center py={80}>
      <Stack align="center" gap="sm">
        <Title order={1}>404</Title>
        <Text c="dimmed">Такой страницы нет.</Text>
        <Button component={Link} to="/">
          На главную
        </Button>
      </Stack>
    </Center>
  )
}
