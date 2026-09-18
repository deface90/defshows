import { ActionIcon, Button, Card, Center, Divider, Group, Select, Stack, Tabs, Text, TextInput, Title, Tooltip } from '@mantine/core'
import { useIntersection } from '@mantine/hooks'
import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { MyShowRow } from '@/entities/show/MyShowRow'
import { STATUS_OPTIONS } from '@/entities/show/status'
import { useListTracked } from '@/shared/api/tracking/endpoints'
import type { TrackedShow } from '@/shared/api/tracking/model'
import { useDocumentTitle } from '@/shared/lib/useDocumentTitle'
import { EmptyState, ErrorState, LoadingState } from '@/shared/ui/states'

const tabs = [{ value: 'all', label: 'Все' }, ...STATUS_OPTIONS]

const sortOptions = [
  { value: 'default', label: 'По умолчанию' },
  { value: 'title', label: 'По названию' },
  { value: 'progress', label: 'По прогрессу' },
]

// Filter by the show's broadcast (airing) status — distinct from the user's own
// watch status in the tabs above.
const airingOptions = [
  { value: '', label: 'Любой статус выхода' },
  { value: 'airing', label: '🔴 Идёт' },
  { value: 'between_seasons', label: '⏸ Между сезонами' },
  { value: 'not_started', label: 'Не начат' },
  { value: 'ended', label: '✅ Завершён' },
]

function pct(t: TrackedShow): number {
  return t.progress.total > 0 ? t.progress.watched / t.progress.total : 0
}

// How many rows to reveal per infinite-scroll step over the fully-loaded list.
const PAGE = 20

export function MyShowsPage() {
  const [status, setStatus] = useState('')
  const [airing, setAiring] = useState('')
  const [q, setQ] = useState('')
  const [sort, setSort] = useState('default')
  const [reversed, setReversed] = useState(false)
  useDocumentTitle('Мои сериалы')
  const query = useListTracked()

  const all = useMemo(() => query.data?.tracked ?? [], [query.data])

  const counts = useMemo(() => {
    const map: Record<string, number> = { all: all.length }
    for (const opt of STATUS_OPTIONS) {
      map[opt.value] = all.filter((t) => t.user_show.status === opt.value).length
    }
    return map
  }, [all])

  const visible = useMemo(() => {
    let list = status ? all.filter((t) => t.user_show.status === status) : all
    if (airing) list = list.filter((t) => t.show.airing_status === airing)
    const needle = q.trim().toLocaleLowerCase('ru')
    if (needle) {
      list = list.filter((t) =>
        `${t.show.title} ${t.show.original_title ?? ''}`.toLocaleLowerCase('ru').includes(needle),
      )
    }
    // Natural order per key; `reversed` flips it. Default keeps the server order
    // (recently added first).
    if (sort === 'title') {
      list = [...list].sort((a, b) => a.show.title.localeCompare(b.show.title, 'ru'))
    } else if (sort === 'progress') {
      list = [...list].sort((a, b) => pct(b) - pct(a))
    } else {
      list = [...list]
    }
    if (reversed) list.reverse()
    return list
  }, [all, status, airing, q, sort, reversed])

  // Infinite scroll: render a growing window over the (client-side) full list,
  // extending it whenever the sentinel row scrolls into view.
  const [limit, setLimit] = useState(PAGE)
  useEffect(() => {
    setLimit(PAGE)
  }, [status, airing, q, sort, reversed])
  const shown = visible.slice(0, limit)
  const hasMore = limit < visible.length
  const { ref: sentinelRef, entry } = useIntersection({ threshold: 0 })
  useEffect(() => {
    if (entry?.isIntersecting && hasMore) {
      setLimit((l) => l + PAGE)
    }
  }, [entry?.isIntersecting, hasMore])

  return (
    <Stack gap="lg">
      <Group justify="space-between" align="flex-start" wrap="wrap">
        <div>
          <Title order={2}>Мои сериалы</Title>
          <Text c="dimmed" mt={4}>
            Коллекция и прогресс просмотра
          </Text>
        </div>
        <Button component={Link} to="/search">
          + Найти сериал
        </Button>
      </Group>

      <Card withBorder padding={0}>
        <Tabs value={status || 'all'} onChange={(v) => setStatus(v === 'all' ? '' : (v ?? ''))}>
          <Tabs.List px="sm">
            {tabs.map((t) => (
              <Tabs.Tab key={t.value} value={t.value}>
                {t.label}{' '}
                <Text span c="dimmed" size="sm">
                  {counts[t.value] ?? 0}
                </Text>
              </Tabs.Tab>
            ))}
          </Tabs.List>
        </Tabs>

        <Group p="md" gap="sm" wrap="wrap" style={{ background: 'var(--mantine-color-body)' }}>
          <TextInput
            style={{ flex: 1, minWidth: 200 }}
            placeholder="Поиск по моим сериалам…"
            value={q}
            onChange={(e) => setQ(e.currentTarget.value)}
            aria-label="Поиск по моим сериалам"
          />
          <Select
            w={210}
            data={airingOptions}
            value={airing}
            onChange={(v) => setAiring(v ?? '')}
            allowDeselect={false}
            aria-label="Фильтр по статусу выхода"
          />
          <Group gap={6} wrap="nowrap">
            <Select
              w={190}
              data={sortOptions}
              value={sort}
              onChange={(v) => v && setSort(v)}
              allowDeselect={false}
              aria-label="Сортировка"
            />
            <Tooltip label={reversed ? 'Обратный порядок' : 'Прямой порядок'} withArrow>
              <ActionIcon
                variant="default"
                size="lg"
                onClick={() => setReversed((r) => !r)}
                aria-label="Направление сортировки"
              >
                {reversed ? '↑' : '↓'}
              </ActionIcon>
            </Tooltip>
          </Group>
        </Group>
        <Divider />

        {query.isLoading && <LoadingState />}
        {query.isError && <ErrorState message="Не удалось загрузить список" />}
        {query.isSuccess && all.length === 0 && (
          <EmptyState
            title="Пока пусто"
            description={
              <>
                Найдите сериал на странице{' '}
                <Text span component={Link} to="/search" c="brand">
                  поиска
                </Text>{' '}
                и добавьте его.
              </>
            }
          />
        )}
        {query.isSuccess && all.length > 0 && visible.length === 0 && (
          <EmptyState
            title="Ничего не найдено"
            description="Попробуйте изменить запрос или выбрать другой статус."
          />
        )}
        {shown.map((t, i) => (
          <div key={t.user_show.id}>
            {i > 0 && <Divider />}
            <MyShowRow tracked={t} />
          </div>
        ))}
        {hasMore && (
          <div ref={sentinelRef}>
            <Center p="md">
              <Text c="dimmed" size="sm">
                Показано {shown.length} из {visible.length}
              </Text>
            </Center>
          </div>
        )}
      </Card>
    </Stack>
  )
}
