import { Button, Card, Divider, Group, Stack, Text, Title } from '@mantine/core'
import { useMemo } from 'react'
import { Link } from 'react-router-dom'
import { MyShowRow } from '@/entities/show/MyShowRow'
import { useListTracked } from '@/shared/api/tracking/endpoints'
import { useDocumentTitle } from '@/shared/lib/useDocumentTitle'
import { EmptyState, ErrorState, LoadingState } from '@/shared/ui/states'

/**
 * UnwatchedPage lists tracked shows that have already-aired episodes the user
 * hasn't marked watched, most-behind first, with a per-show counter.
 */
export function UnwatchedPage() {
  useDocumentTitle('Непросмотренные')
  const query = useListTracked()

  const shows = useMemo(() => {
    const list = (query.data?.tracked ?? []).filter((t) => t.progress.unwatched > 0)
    return list.sort((a, b) => b.progress.unwatched - a.progress.unwatched)
  }, [query.data])

  const totalEpisodes = useMemo(
    () => shows.reduce((sum, t) => sum + t.progress.unwatched, 0),
    [shows],
  )

  return (
    <Stack gap="lg">
      <Group justify="space-between" align="flex-start" wrap="wrap">
        <div>
          <Title order={2}>Непросмотренные</Title>
          <Text c="dimmed" mt={4}>
            Сериалы с вышедшими, но не отмеченными эпизодами
            {shows.length > 0 ? ` · ${totalEpisodes} эпизодов к просмотру` : ''}
          </Text>
        </div>
        <Button component={Link} to="/" variant="default">
          Все сериалы
        </Button>
      </Group>

      <Card withBorder padding={0}>
        {query.isLoading && <LoadingState />}
        {query.isError && <ErrorState message="Не удалось загрузить список" />}
        {query.isSuccess && shows.length === 0 && (
          <EmptyState
            title="Всё просмотрено"
            description="Нет сериалов с вышедшими непросмотренными эпизодами."
          />
        )}
        {shows.map((t, i) => (
          <div key={t.user_show.id}>
            {i > 0 && <Divider />}
            <MyShowRow tracked={t} showUnwatched />
          </div>
        ))}
      </Card>
    </Stack>
  )
}
