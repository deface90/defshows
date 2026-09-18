import { Alert, Center, Loader, Stack, Text } from '@mantine/core'
import type { ReactNode } from 'react'

/** LoadingState is a centered spinner. */
export function LoadingState({ label = 'Загрузка…' }: { label?: string }) {
  return (
    <Center py="xl">
      <Stack align="center" gap="xs">
        <Loader />
        <Text c="dimmed" size="sm">
          {label}
        </Text>
      </Stack>
    </Center>
  )
}

/** EmptyState communicates "nothing here yet". */
export function EmptyState({ title = 'Пусто', description }: { title?: string; description?: ReactNode }) {
  return (
    <Center py="xl">
      <Stack align="center" gap={4}>
        <Text fw={600}>{title}</Text>
        {description && (
          <Text c="dimmed" size="sm" ta="center">
            {description}
          </Text>
        )}
      </Stack>
    </Center>
  )
}

/** ErrorState shows a recoverable error message. */
export function ErrorState({ message = 'Что-то пошло не так' }: { message?: string }) {
  return (
    <Alert color="red" title="Ошибка" my="md">
      {message}
    </Alert>
  )
}
