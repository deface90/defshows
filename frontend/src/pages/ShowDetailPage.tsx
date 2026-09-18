import { WatchShowButton } from '@/features/mark-watched/WatchShowButton'
import { Anchor, Badge, Box, Card, Group, Progress, Stack, Tabs, Text, Title } from '@mantine/core'
import { Link, useParams } from 'react-router-dom'
import { AddShowButton } from '@/features/add-show/AddShowButton'
import { ContinueWatching } from '@/features/mark-watched/ContinueWatching'
import { NotesList } from '@/features/notes/NotesList'
import { LinksEditor } from '@/features/show-links/LinksEditor'
import { StatusSelect } from '@/features/show-status/StatusSelect'
import { RecapView } from '@/entities/recap/RecapView'
import { SeasonAccordion } from '@/entities/episode/SeasonAccordion'
import { AiringStatusBadge } from '@/entities/show/AiringStatusBadge'
import { NextEpisodeInfo } from '@/entities/show/NextEpisodeInfo'
import { Poster } from '@/entities/show/Poster'
import { RatingBadges } from '@/entities/show/RatingBadges'
import { useGetShow } from '@/shared/api/shows/endpoints'
import type { Episode, Show } from '@/shared/api/shows/model'
import { useGetTracked } from '@/shared/api/tracking/endpoints'
import { isAxiosError } from 'axios'
import { EmptyState, ErrorState, LoadingState } from '@/shared/ui/states'

const eyebrow = { textTransform: 'uppercase', letterSpacing: '0.09em' } as const

export function ShowDetailPage() {
  const { id } = useParams()
  const showId = Number(id)
  const showQuery = useGetShow(showId, { query: { enabled: Number.isFinite(showId) } })
  if (showQuery.isLoading) return <LoadingState />
  if (showQuery.isError || !showQuery.data) return <ErrorState message="Сериал не найден" />
  return <ShowDetailContent show={showQuery.data} />
}

export function ShowDetailContent({ show, preview = false, backTo = '/search' }: { show: Show; preview?: boolean; backTo?: string }) {
  const trackedQuery = useGetTracked(show.id, { query: { retry: false } })
  const showId = show.id
  const trackedShow = trackedQuery.isSuccess ? trackedQuery.data : undefined
  const tracked = !!trackedShow && !preview
  const progress = preview ? undefined : trackedShow?.progress
  const seasons = show.seasons ?? []
  const watchedIds = new Set(progress?.watched_episode_ids ?? [])
  const notTracked = isAxiosError(trackedQuery.error) && trackedQuery.error.response?.status === 404

  let nextEp: Episode | null = null
  if (progress?.next_unwatched_episode_id != null) {
    nextEp =
      seasons.flatMap((s) => s.episodes).find((e) => e.id === progress.next_unwatched_episode_id) ??
      null
  }

  const hasOriginal = show.original_title && show.original_title !== show.title
  const year = show.first_air_date?.slice(0, 4)
  const overallPct =
    progress && progress.total > 0 ? Math.round((progress.watched / progress.total) * 100) : 0

  return (
    <Stack gap="lg">
      <Anchor component={Link} to={preview ? backTo : "/"} size="sm" c="dimmed">
        {preview ? (backTo.startsWith("/discover") ? "← Подбор сериалов" : "← Поиск сериалов") : "← Мои сериалы"}
      </Anchor>

      {/* Hero */}
      <Card withBorder padding="lg">
        <Group align="flex-start" wrap="nowrap" gap="xl">
          <Poster
            src={show.poster_url || undefined}
            title={show.title}
            w={150}
            voteAverage={show.vote_average}
            voteCount={show.vote_count}
          />
          <Stack gap="sm" style={{ flex: 1, minWidth: 0 }}>
            <div>
              <Title order={2}>{show.title}</Title>
              {hasOriginal && (
                <Text c="dimmed" fz="lg" mt={4}>
                  {show.original_title}
                </Text>
              )}
            </div>

            <Group gap="xs" align="center">
              <AiringStatusBadge status={show.airing_status} />
              {year && (
                <Text c="dimmed" size="sm">
                  {year}
                </Text>
              )}
              {(show.genres ?? []).map((g) => (
                <Badge key={g.id} variant="outline" color="gray">
                  {g.name}
                </Badge>
              ))}
            </Group>

            <NextEpisodeInfo date={show.next_episode_air_date} />
            <RatingBadges ratings={show.ratings} />

            {trackedShow && !preview ? (
              <Stack gap="sm" mt="xs">
                <Box maw={260}>
                  <Text size="xs" c="dimmed" mb={4}>
                    Мой статус
                  </Text>
                  <StatusSelect showId={show.id} status={trackedShow.user_show.status} />
                </Box>
                <Box>
                  <WatchShowButton
                    showId={show.id}
                    completed={trackedShow.user_show.status === 'completed' && !!progress && progress.watched === progress.total}
                  />
                </Box>
                {progress && (
                  <Box maw={560}>
                    <Group justify="space-between" mb={6}>
                      <Text size="sm" fw={600}>
                        {progress.watched} / {progress.total} эпизодов
                      </Text>
                      <Text size="sm" c="dimmed">
                        {overallPct}%
                      </Text>
                    </Group>
                    <Progress value={overallPct} size="md" radius="xl" />
                  </Box>
                )}
              </Stack>
            ) : (
              <Box mt="xs">
                {trackedShow ? (
                  <Anchor component={Link} to={`/shows/${show.id}`}>В моих сериалах →</Anchor>
                ) : notTracked ? (
                  <AddShowButton tmdbId={show.tmdb_id} />
                ) : trackedQuery.isError ? (
                  <Text c="red" size="sm">Не удалось проверить добавление сериала</Text>
                ) : <Text c="dimmed" size="sm">Проверяем список сериалов…</Text>}
              </Box>
            )}
          </Stack>
        </Group>
      </Card>

      {tracked && <ContinueWatching showId={show.id} next={nextEp} />}

      {/* About + tabs */}
      <Card withBorder padding="lg">
        <Stack gap="md">
          <div>
            <Text fw={800} fz={11} c="dimmed" style={eyebrow}>
              О сериале
            </Text>
            {show.overview ? (
              <Text size="sm" mt="xs" style={{ lineHeight: 1.7 }}>
                {show.overview}
              </Text>
            ) : (
              <Text size="sm" c="dimmed" mt="xs">
                Описание отсутствует
              </Text>
            )}
          </div>

          {!preview && <Tabs key={tracked ? 'tracked' : 'guest'} defaultValue={tracked ? 'links' : 'recap'}>
            <Tabs.List>
              {tracked && <Tabs.Tab value="links">Ссылки</Tabs.Tab>}
              <Tabs.Tab value="recap">Рекап</Tabs.Tab>
              {tracked && <Tabs.Tab value="notes">Заметки</Tabs.Tab>}
            </Tabs.List>

            {tracked && (
              <Tabs.Panel value="links" pt="md">
                <LinksEditor showId={show.id} dubbing={trackedShow?.user_show.preferred_dubbing} />
              </Tabs.Panel>
            )}

            <Tabs.Panel value="recap" pt="md">
              <RecapView showId={show.id} />
            </Tabs.Panel>

            {tracked && (
              <Tabs.Panel value="notes" pt="md">
                <NotesList showId={show.id} />
              </Tabs.Panel>
            )}
          </Tabs>}
        </Stack>
      </Card>

      {/* Seasons */}
      <Card withBorder padding="lg">
        <Group justify="space-between" mb="md">
          <div>
            <Text fw={800} fz={11} c="dimmed" style={eyebrow}>
              Эпизоды
            </Text>
            <Title order={4} mt={4}>
              Список сезонов
            </Title>
          </div>
          {progress && (
            <Text c="dimmed" size="sm">
              {progress.watched} / {progress.total} просмотрено
            </Text>
          )}
        </Group>
        {seasons.length === 0 ? (
          <EmptyState title="Нет эпизодов" />
        ) : (
          <SeasonAccordion
            seasons={seasons}
            showId={show.id}
            tracked={tracked}
            watchedIds={watchedIds}
          />
        )}
      </Card>
    </Stack>
  )
}
