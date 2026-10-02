import { Stack, Title } from '@mantine/core'
import { useCallback } from 'react'
import { ActivityFeed } from '@/entities/activity/ActivityFeed'
import { getHomeFeed } from '@/shared/api/social/endpoints'
import { useDocumentTitle } from '@/shared/lib/useDocumentTitle'

/** FeedPage is the home feed: the aggregated activity of everyone you follow. */
export function FeedPage() {
  useDocumentTitle('Лента')
  const fetchPage = useCallback(
    (cursor?: string) => getHomeFeed({ cursor, limit: 20 }),
    [],
  )
  return (
    <Stack gap="lg">
      <Title order={2}>Лента</Title>
      <ActivityFeed
        fetchPage={fetchPage}
        emptyText="Подпишитесь на кого-нибудь, чтобы видеть их активность здесь."
      />
    </Stack>
  )
}
