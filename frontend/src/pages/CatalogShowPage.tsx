import { useParams, useSearchParams } from 'react-router-dom'
import { useGetShowByTMDB } from '@/shared/api/shows/endpoints'
import { ErrorState, LoadingState } from '@/shared/ui/states'
import { ShowDetailContent } from './ShowDetailPage'

export function CatalogShowPage() {
  const [params] = useSearchParams()
  const from = params.get('from') ?? ''
  const backTo = /^\/discover(?:\?|$)/.test(from) ? from : '/search'
  const { tmdbId } = useParams()
  const id = Number(tmdbId)
  const valid = Number.isSafeInteger(id) && id > 0
  const query = useGetShowByTMDB(id, { query: { enabled: valid, retry: false } })
  if (!valid) return <ErrorState message="Сериал не найден" />
  if (query.isLoading) return <LoadingState />
  if (query.isError || !query.data) return <ErrorState message="Не удалось загрузить сериал" />
  return <ShowDetailContent show={query.data} preview backTo={backTo} />
}
