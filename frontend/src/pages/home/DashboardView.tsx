import { Badge, Box, Button, Card, Divider, Group, SimpleGrid, Stack, Text, Title } from '@mantine/core'
import { useMemo } from 'react'
import { Link } from 'react-router-dom'
import { AddShowButton } from '@/features/add-show/AddShowButton'
import { MyShowRow } from '@/entities/show/MyShowRow'
import { Poster } from '@/entities/show/Poster'
import { useGetRecommendations, useListTracked } from '@/shared/api/tracking/endpoints'
import type { ShowRef, TrackedShow } from '@/shared/api/tracking/model'
import { useAuthStore } from '@/shared/auth/authStore'
import { useDocumentTitle } from '@/shared/lib/useDocumentTitle'
import { EmptyState, ErrorState, LoadingState } from '@/shared/ui/states'

// How many rows each dashboard section previews before linking to the full page.
const PREVIEW = 5

function StatCard({ label, value }: { label: string; value: number }) {
  return (
    <Card withBorder padding="md" radius="md">
      <Text fz={28} fw={700} lh={1}>
        {value}
      </Text>
      <Text c="dimmed" fz="sm" mt={4}>
        {label}
      </Text>
    </Card>
  )
}

/**
 * DashboardView is the authenticated home at `/`: a personal summary built from
 * the user's tracked list — quick stats, what to continue watching, and the
 * unwatched backlog. Taste analysis + recommendations arrive in Phase 4.
 */
export function DashboardView() {
  useDocumentTitle('Главная')
  const user = useAuthStore((s) => s.user)
  const query = useListTracked()

  const all = useMemo(() => query.data?.tracked ?? [], [query.data])

  const stats = useMemo(
    () => ({
      total: all.length,
      watching: all.filter((t) => t.user_show.status === 'watching').length,
      backlog: all.reduce((sum, t) => sum + t.progress.unwatched, 0),
      completed: all.filter((t) => t.user_show.status === 'completed').length,
    }),
    [all],
  )

  // "Продолжить смотреть": actively-watched shows, least-progressed first.
  const continueWatching = useMemo(() => {
    const pct = (t: (typeof all)[number]) =>
      t.progress.total > 0 ? t.progress.watched / t.progress.total : 0
    return all
      .filter((t) => t.user_show.status === 'watching')
      .sort((a, b) => pct(a) - pct(b))
      .slice(0, PREVIEW)
  }, [all])

  // "Непросмотренные": aired-but-unwatched backlog, most-behind first.
  const unwatched = useMemo(
    () =>
      all
        .filter((t) => t.progress.unwatched > 0)
        .sort((a, b) => b.progress.unwatched - a.progress.unwatched)
        .slice(0, PREVIEW),
    [all],
  )

  const greeting = user?.display_name || user?.email || 'снова привет'

  return (
    <Stack gap="lg">
      <div>
        <Title order={2}>Привет, {greeting} 👋</Title>
        <Text c="dimmed" mt={4}>
          Ваша сводка по просмотру
        </Text>
      </div>

      {query.isLoading && <LoadingState />}
      {query.isError && <ErrorState message="Не удалось загрузить сводку" />}

      {query.isSuccess && all.length === 0 && (
        <Card withBorder padding={0}>
          <EmptyState
            title="Начните свою коллекцию"
            description={
              <>
                Найдите сериал в{' '}
                <Text span component={Link} to="/search" c="brand">
                  поиске
                </Text>{' '}
                или загляните в{' '}
                <Text span component={Link} to="/discover" c="brand">
                  подборки
                </Text>
                .
              </>
            }
          />
        </Card>
      )}

      {query.isSuccess && all.length > 0 && (
        <>
          <SimpleGrid cols={{ base: 2, sm: 4 }} spacing="md">
            <StatCard label="Всего сериалов" value={stats.total} />
            <StatCard label="Смотрю" value={stats.watching} />
            <StatCard label="Эпизодов к просмотру" value={stats.backlog} />
            <StatCard label="Завершено" value={stats.completed} />
          </SimpleGrid>

          <DashboardSection
            title="Продолжить смотреть"
            to="/my"
            linkLabel="Все сериалы"
            rows={continueWatching}
            emptyText="Нет сериалов в статусе «Смотрю»."
          />

          <DashboardSection
            title="Непросмотренные серии"
            to="/unwatched"
            linkLabel="Все непросмотренные"
            rows={unwatched}
            emptyText="Всё вышедшее просмотрено — красота!"
          />

          <RecommendationsBlock />
        </>
      )}
    </Stack>
  )
}

/**
 * RecommendationsBlock shows the user's taste profile (top genres/languages) and
 * catalog picks derived from it. Data is cached server-side; the block hides
 * itself while there's not enough signal or on a (non-critical) error.
 */
function RecommendationsBlock() {
  const { data, isLoading, isError } = useGetRecommendations()

  if (isLoading) return <LoadingState label="Подбираем рекомендации…" />
  if (isError || !data) return null

  const { taste, recommendations } = data
  const hasTaste = taste.genres.length > 0 || taste.languages.length > 0
  if (!hasTaste && recommendations.length === 0) return null

  return (
    <Stack gap="sm">
      <Title order={3} fz="h4">
        Для вас
      </Title>
      {hasTaste && (
        <Group gap="xs">
          {taste.genres.map((g) => (
            <Badge key={g.id} variant="light" color="orange">
              {g.name}
            </Badge>
          ))}
          {taste.languages.map((l) => (
            <Badge key={l.code} variant="default">
              {l.code.toUpperCase()}
            </Badge>
          ))}
        </Group>
      )}
      {recommendations.length > 0 ? (
        <SimpleGrid cols={{ base: 2, xs: 3, sm: 4, md: 6 }} spacing="md">
          {recommendations.map((show) => (
            <RecommendationCard key={show.id} show={show} />
          ))}
        </SimpleGrid>
      ) : (
        <Text c="dimmed" size="sm">
          Пока недостаточно данных для рекомендаций — добавьте ещё пару сериалов.
        </Text>
      )}
    </Stack>
  )
}

function RecommendationCard({ show }: { show: ShowRef }) {
  return (
    <Stack gap={6} align="center">
      <Box
        component={Link}
        to={`/catalog/${show.tmdb_id}`}
        aria-label={show.title}
        style={{ display: 'block' }}
      >
        <Poster
          src={show.poster_url || undefined}
          title={show.title}
          w={120}
          voteAverage={show.vote_average}
          voteCount={show.vote_count}
        />
      </Box>
      <Text size="sm" fw={500} ta="center" lineClamp={2} w={120}>
        {show.title}
      </Text>
      <AddShowButton tmdbId={show.tmdb_id} />
    </Stack>
  )
}

function DashboardSection({
  title,
  to,
  linkLabel,
  rows,
  emptyText,
}: {
  title: string
  to: string
  linkLabel: string
  rows: TrackedShow[]
  emptyText: string
}) {
  return (
    <Stack gap="sm">
      <Group justify="space-between" align="center">
        <Title order={3} fz="h4">
          {title}
        </Title>
        <Button component={Link} to={to} variant="subtle" size="compact-sm">
          {linkLabel} →
        </Button>
      </Group>
      <Card withBorder padding={0}>
        {rows.length === 0 ? (
          <Text c="dimmed" size="sm" ta="center" py="xl">
            {emptyText}
          </Text>
        ) : (
          rows.map((t, i) => (
            <div key={t.user_show.id}>
              {i > 0 && <Divider />}
              <MyShowRow tracked={t} showUnwatched />
            </div>
          ))
        )}
      </Card>
    </Stack>
  )
}
