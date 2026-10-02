import { Badge, Card, Group, Image, Stack, Text } from '@mantine/core'
import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import type { FeedCard } from '@/shared/api/social/model'

const pad = (n: number) => String(n).padStart(2, '0')
const code = (season: number, episode: number) => `S${pad(season)}E${pad(episode)}`

/**
 * feedCardLabel renders the short Russian action phrase for a feed card. Grouped
 * watched episodes collapse to a range, e.g. «Серии S02E01–E05 · 5 серий».
 */
export function feedCardLabel(card: FeedCard): string {
  switch (card.type) {
    case 'watched_episode': {
      const eps = [...(card.episodes ?? [])].sort(
        (a, b) => a.season_number - b.season_number || a.episode_number - b.episode_number,
      )
      if (eps.length === 0) return 'Просмотр серий'
      if (eps.length === 1) return `Серия ${code(eps[0].season_number, eps[0].episode_number)}`
      const first = eps[0]
      const last = eps[eps.length - 1]
      // Within one season, abbreviate the end to just the episode, e.g. S02E01–E05.
      const end =
        first.season_number === last.season_number
          ? `E${pad(last.episode_number)}`
          : code(last.season_number, last.episode_number)
      return `Серии ${code(first.season_number, first.episode_number)}–${end} · ${card.count} серий`
    }
    case 'finished_season':
      return card.season_number != null ? `Сезон ${card.season_number} завершён` : 'Сезон завершён'
    case 'finished_show':
      return 'Сериал завершён'
    case 'added_show':
      return 'Добавлен в коллекцию'
    case 'rated_show':
      return card.rating != null ? `Оценка ${card.rating}/10` : 'Оценка снята'
    default:
      return ''
  }
}

/**
 * FeedCardView renders a single activity event; actor is shown only on the home feed.
 * `renderActorMenu` is an optional page-supplied slot (FSD: entities can't import
 * features) that mounts moderation controls next to the actor, e.g. Block/Report.
 */
export function FeedCardView({
  card,
  renderActorMenu,
}: {
  card: FeedCard
  renderActorMenu?: (actorId: number) => ReactNode
}) {
  const date = new Date(card.created_at).toLocaleDateString('ru-RU', {
    day: 'numeric',
    month: 'short',
  })
  return (
    <Card withBorder padding="sm">
      <Group wrap="nowrap" align="flex-start">
        {card.show.poster_url && (
          <Image src={card.show.poster_url} alt="" w={40} h={60} radius="sm" fit="cover" />
        )}
        <Stack gap={2} style={{ minWidth: 0, flex: 1 }}>
          {card.actor && (
            <Group gap={4} wrap="nowrap" justify="space-between">
              <Text size="xs" c="dimmed" truncate>
                {card.actor.display_name}
              </Text>
              {renderActorMenu?.(card.actor.id)}
            </Group>
          )}
          <Text component={Link} to={`/catalog/${card.show.tmdb_id}`} fw={600} truncate style={{ color: 'inherit' }}>
            {card.show.title}
          </Text>
          <Group gap="xs" wrap="nowrap">
            <Badge variant="light" color="orange">
              {feedCardLabel(card)}
            </Badge>
            <Text size="xs" c="dimmed">
              {date}
            </Text>
          </Group>
        </Stack>
      </Group>
    </Card>
  )
}
