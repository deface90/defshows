import { Stack, Title } from '@mantine/core'
import { useCallback } from 'react'
import { ActivityFeed } from '@/entities/activity/ActivityFeed'
import { ModerationMenu } from '@/features/report-user/ModerationMenu'
import { getHomeFeed } from '@/shared/api/social/endpoints'
import { useDocumentTitle } from '@/shared/lib/useDocumentTitle'

/** FeedPage is the home feed: the aggregated activity of everyone you follow. */
export function FeedPage() {
  useDocumentTitle('Лента')
  const fetchPage = useCallback(
    (cursor?: string) => getHomeFeed({ cursor, limit: 20 }),
    [],
  )
  const renderActorMenu = useCallback(
    (actorId: number, reload: () => void) => (
      <ModerationMenu userId={actorId} onBlocked={reload} />
    ),
    [],
  )
  return (
    <Stack gap="lg">
      <Title order={2}>Лента</Title>
      <ActivityFeed
        fetchPage={fetchPage}
        emptyText="Подпишитесь на кого-нибудь, чтобы видеть их активность здесь."
        renderActorMenu={renderActorMenu}
      />
    </Stack>
  )
}
