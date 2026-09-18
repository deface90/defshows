import { SimpleGrid, Stack, TextInput, Title } from '@mantine/core'
import { useDebouncedValue } from '@mantine/hooks'
import { useState } from 'react'
import { AddShowButton } from '@/features/add-show/AddShowButton'
import { ShowCard } from '@/entities/show/ShowCard'
import { useSearchShows } from '@/shared/api/shows/endpoints'
import { EmptyState, ErrorState, LoadingState } from '@/shared/ui/states'

export function SearchPage() {
  const [q, setQ] = useState('')
  const [debounced] = useDebouncedValue(q, 400)
  const enabled = debounced.trim().length >= 3

  const query = useSearchShows({ q: debounced }, { query: { enabled } })

  return (
    <Stack>
      <Title order={3}>Поиск сериалов</Title>
      <TextInput
        placeholder="Название сериала…"
        value={q}
        onChange={(e) => setQ(e.currentTarget.value)}
        aria-label="Поиск сериалов"
      />

      {!enabled && <EmptyState title="Введите минимум 3 символа" />}
      {enabled && query.isLoading && <LoadingState />}
      {enabled && query.isError && <ErrorState message="Не удалось выполнить поиск" />}
      {enabled && query.isSuccess && query.data.results.length === 0 && (
        <EmptyState title="Ничего не найдено" description={`По запросу «${debounced}» нет результатов`} />
      )}
      {enabled && query.isSuccess && query.data.results.length > 0 && (
        <SimpleGrid cols={{ base: 2, sm: 3, md: 4, lg: 5 }}>
          {query.data.results.map((show) => (
            <ShowCard key={show.tmdb_id} show={show} action={<AddShowButton tmdbId={show.tmdb_id} />} />
          ))}
        </SimpleGrid>
      )}
    </Stack>
  )
}
