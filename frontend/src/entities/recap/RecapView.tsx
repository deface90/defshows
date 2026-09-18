import { Card, Stack, Text } from '@mantine/core'
import { useGetRecap } from '@/shared/api/notes/endpoints'
import { EmptyState, LoadingState } from '@/shared/ui/states'

/**
 * RecapView renders the public generated recap for a show. The recap is produced
 * asynchronously on the backend, so a 404 is an expected "not ready" state rather
 * than an error.
 */
export function RecapView({ showId }: { showId: number }) {
  const query = useGetRecap(
    showId,
    { scope: 'show' },
    { query: { enabled: Number.isFinite(showId), retry: false } },
  )

  if (query.isLoading) return <LoadingState label="Загружаем recap…" />
  if (query.isError || !query.data) {
    return (
      <EmptyState
        title="Recap пока недоступен"
        description="Пересказ ещё не сгенерирован — загляните позже."
      />
    )
  }

  const recap = query.data
  return (
    <Card withBorder padding="md">
      <Stack gap="xs">
        <Text c="dimmed" size="xs">
          Язык: {recap.language}
          {recap.generated_at ? ` · сгенерирован ${new Date(recap.generated_at).toLocaleDateString()}` : ''}
        </Text>
        <Text size="sm" style={{ whiteSpace: 'pre-wrap' }}>
          {recap.body}
        </Text>
      </Stack>
    </Card>
  )
}
