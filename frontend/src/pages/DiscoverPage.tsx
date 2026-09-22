import { Button, Card, Group, MultiSelect, NumberInput, Pagination, SegmentedControl, Select, SimpleGrid, Stack, Text, TextInput, Title } from '@mantine/core'
import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { ShowCard } from '@/entities/show/ShowCard'
import { AddShowButton } from '@/features/add-show/AddShowButton'
import { useDiscoverShows, useGetDiscoveryFilters, useTrendingShows } from '@/shared/api/shows/endpoints'
import type { DiscoverShowsParams } from '@/shared/api/shows/model'
import { useListTracked } from '@/shared/api/tracking/endpoints'
import { useDocumentTitle } from '@/shared/lib/useDocumentTitle'
import { EmptyState, ErrorState, LoadingState } from '@/shared/ui/states'

const sortOptions = [
  { value: 'popularity.desc', label: 'По популярности' },
  { value: 'vote_average.desc', label: 'По рейтингу' },
  { value: 'first_air_date.desc', label: 'Сначала новые' },
]

function readParams(params: URLSearchParams): DiscoverShowsParams {
  const number = (key: string, fallback: number) => params.has(key) ? Number(params.get(key)) : fallback
  return {
    country: params.get('country') || undefined,
    genres: params.get('genres') || undefined,
    rating_min: number('rating_min', 0), rating_max: number('rating_max', 10),
    votes_min: number('votes_min', 100),
    date_from: params.get('date_from') || undefined, date_to: params.get('date_to') || undefined,
    sort: (params.get('sort') || 'popularity.desc') as DiscoverShowsParams['sort'],
    page: number('page', 1),
  }
}

function invalid(params: DiscoverShowsParams): boolean {
  return !Number.isFinite(params.rating_min) || !Number.isFinite(params.rating_max) ||
    params.rating_min! < 0 || params.rating_max! > 10 || params.rating_min! > params.rating_max! ||
    !Number.isSafeInteger(params.votes_min) || params.votes_min! < 0 ||
    !Number.isSafeInteger(params.page) || params.page! < 1 || params.page! > 500 ||
    (!!params.date_from && !!params.date_to && params.date_from > params.date_to)
}

function serialize(params: DiscoverShowsParams): URLSearchParams {
  const result = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== '') result.set(key, String(value))
  })
  return result
}

const regionNames = new Intl.DisplayNames(['ru'], { type: 'region' })

export function DiscoverPage() {
  useDocumentTitle('Подбор сериалов')
  const [searchParams, setSearchParams] = useSearchParams()
  const applied = readParams(searchParams)
  const [draft, setDraft] = useState(applied)
  const [genreMode, setGenreMode] = useState(applied.genres?.includes(',') ? 'all' : 'any')
  useEffect(() => {
    const next = readParams(searchParams)
    setDraft(next)
    setGenreMode(next.genres?.includes(',') ? 'all' : 'any')
  }, [searchParams])
  const references = useGetDiscoveryFilters({ query: { staleTime: 24 * 60 * 60 * 1000, retry: false } })
  // Only run a search once the user has submitted the form (or opened a shared
  // URL that already carries filters); an empty URL means "waiting for input".
  const hasQuery = searchParams.toString().length > 0
  const results = useDiscoverShows(applied, { query: { enabled: hasQuery && !invalid(applied), retry: false } })
  const trending = useTrendingShows({ query: { enabled: !hasQuery, retry: false } })
  const tracked = useListTracked(undefined, { query: { retry: false } })
  const genres = draft.genres?.split(/[,|]/).filter(Boolean) ?? []
  const countries = (references.data?.countries ?? []).map((country) => ({
    value: country.code, label: regionNames.of(country.code) || country.name,
  })).sort((a, b) => a.label.localeCompare(b.label, 'ru'))
  const set = <K extends keyof DiscoverShowsParams>(key: K, value: DiscoverShowsParams[K]) => setDraft((prev) => ({ ...prev, [key]: value }))
  const numberValue = (value: string | number, fallback: number) => value === '' ? fallback : Number(value)

  return (
    <Stack gap="lg">
      <Title order={3}>Подбор сериала</Title>
      <Card withBorder padding="lg">
        <Stack component="form" onSubmit={(event) => {
          event.preventDefault()
          if (!invalid({ ...draft, page: 1 })) setSearchParams(serialize({ ...draft, page: 1 }))
        }}>
          {references.isError && <Group><Text c="red" size="sm">Не удалось загрузить страны и жанры</Text><Button variant="subtle" size="xs" onClick={() => references.refetch()}>Повторить</Button></Group>}
          <SimpleGrid cols={{ base: 1, sm: 2 }}>
            <Select label="Страна производства" placeholder="Любая страна" searchable clearable data={countries} value={draft.country ?? null} onChange={(value) => set('country', value || undefined)} disabled={!references.isSuccess} />
            <MultiSelect label="Жанры" placeholder="Любые жанры" searchable clearable data={(references.data?.genres ?? []).map((genre) => ({ value: String(genre.id), label: genre.name }))} value={genres} onChange={(values) => set('genres', values.join(genreMode === 'all' ? ',' : '|') || undefined)} disabled={!references.isSuccess} />
          </SimpleGrid>
          <Group><Text size="sm">Совпадение жанров</Text><SegmentedControl value={genreMode} data={[{ value: 'any', label: 'Любой из выбранных' }, { value: 'all', label: 'Все выбранные' }]} onChange={(value) => { setGenreMode(value); set('genres', genres.join(value === 'all' ? ',' : '|') || undefined) }} /></Group>
          <SimpleGrid cols={{ base: 1, sm: 3 }}>
            <NumberInput label="Рейтинг TMDB от" min={0} max={10} step={0.1} decimalScale={1} value={draft.rating_min} onChange={(value) => set('rating_min', numberValue(value, 0))} />
            <NumberInput label="Рейтинг TMDB до" min={0} max={10} step={0.1} decimalScale={1} value={draft.rating_max} onChange={(value) => set('rating_max', numberValue(value, 10))} />
            <NumberInput label="Минимум голосов" min={0} allowDecimal={false} value={draft.votes_min} onChange={(value) => set('votes_min', numberValue(value, 0))} />
          </SimpleGrid>
          <SimpleGrid cols={{ base: 1, sm: 3 }}>
            <TextInput type="date" label="Премьера с" value={draft.date_from ?? ''} onChange={(event) => set('date_from', event.currentTarget.value || undefined)} />
            <TextInput type="date" label="Премьера по" value={draft.date_to ?? ''} onChange={(event) => set('date_to', event.currentTarget.value || undefined)} />
            <Select label="Сортировка" data={sortOptions} value={draft.sort} onChange={(value) => set('sort', value as DiscoverShowsParams['sort'])} allowDeselect={false} />
          </SimpleGrid>
          {invalid({ ...draft, page: 1 }) && <Text c="red" size="sm">Проверьте диапазоны рейтинга, дат и количество голосов</Text>}
          <Group>
            <Button type="submit" disabled={invalid({ ...draft, page: 1 })} loading={results.isFetching}>Подобрать</Button>
            <Button variant="default" onClick={() => { setDraft(readParams(new URLSearchParams())); setGenreMode('any'); setSearchParams(new URLSearchParams()) }}>Сбросить</Button>
          </Group>
        </Stack>
      </Card>
      {!hasQuery && <>
        <div>
          <Title order={4}>Сейчас в тренде</Title>
          <Text c="dimmed" size="sm">Задайте фильтры выше, чтобы сузить подборку</Text>
        </div>
        {trending.isLoading && <LoadingState />}
        {trending.isError && <Stack><ErrorState message="Не удалось загрузить тренды" /><Button variant="subtle" onClick={() => trending.refetch()}>Повторить</Button></Stack>}
        {trending.isSuccess && (trending.data.results.length === 0 ? <EmptyState title="Сейчас нет данных о трендах" /> : (
          <SimpleGrid cols={{ base: 2, sm: 3, md: 4, lg: 5 }}>
            {trending.data.results.map((show) => <ShowCard
              key={show.tmdb_id} show={show}
              to={`/catalog/${show.tmdb_id}?from=${encodeURIComponent('/discover')}`}
              action={<AddShowButton tmdbId={show.tmdb_id} disabled={!tracked.isSuccess} addedShowId={tracked.data?.tracked.find((item) => item.show.tmdb_id === show.tmdb_id)?.show.id} />}
            />)}
          </SimpleGrid>
        ))}
      </>}
      {hasQuery && invalid(applied) && <ErrorState message="Некорректные параметры в ссылке. Измените фильтры или сбросьте их." />}
      {results.isLoading && <LoadingState />}
      {results.isError && <Stack><ErrorState message="Не удалось подобрать сериалы" /><Button variant="subtle" onClick={() => results.refetch()}>Повторить подбор</Button></Stack>}
      {results.isSuccess && !invalid(applied) && <>
        <Text c="dimmed" size="sm">Найдено: {results.data.total_results}</Text>
        {results.data.results.length === 0 ? <EmptyState title="Сериалы не найдены" description="Попробуйте расширить условия подбора" /> : (
          <SimpleGrid cols={{ base: 2, sm: 3, md: 4, lg: 5 }}>
            {results.data.results.map((show) => <ShowCard
              key={show.tmdb_id} show={show}
              to={`/catalog/${show.tmdb_id}?from=${encodeURIComponent(`/discover?${searchParams.toString()}`)}`}
              action={<AddShowButton tmdbId={show.tmdb_id} disabled={!tracked.isSuccess} addedShowId={tracked.data?.tracked.find((item) => item.show.tmdb_id === show.tmdb_id)?.show.id} />}
            />)}
          </SimpleGrid>
        )}
        {results.data.total_pages > 1 && <Pagination total={Math.min(results.data.total_pages, 500)} value={applied.page} onChange={(page) => setSearchParams(serialize({ ...applied, page }))} />}
      </>}
    </Stack>
  )
}
