import { Link } from 'react-router-dom'
import { Anchor, Box, Card, Group, Image, Stack, Text } from '@mantine/core'
import type { ReactNode } from 'react'
import type { ShowSummary } from '@/shared/api/shows/model'
import { RatingBadge } from './RatingBadge'

function year(date?: string | null): string {
  return date ? date.slice(0, 4) : ''
}

export function ShowCard({ show, action, to }: { show: ShowSummary; action?: ReactNode; to?: string }) {
  return (
    <Card withBorder padding="sm" h="100%">
      <Stack gap="xs" h="100%">
        <Box style={{ position: 'relative' }}>
          <Box component={Link} to={to ?? "#"} tabIndex={to ? 0 : -1} style={{ pointerEvents: to ? undefined : "none" }} aria-label={show.title}>
          <Image
            src={show.poster_url || undefined}
            h={220}
            radius="sm"
            fallbackSrc="https://placehold.co/300x450?text=No+Poster"
            alt={show.title}
          />
          </Box>
          <Box style={{ position: 'absolute', top: 6, right: 6 }}>
            <RatingBadge average={show.vote_average} votes={show.vote_count} size="xs" />
          </Box>
        </Box>
        <Stack gap={2} style={{ flex: 1 }}>
          <Text fw={600} lineClamp={2}>
            {to ? <Anchor component={Link} to={to} inherit>{show.title}</Anchor> : show.title}
          </Text>
          {year(show.first_air_date) && (
            <Text c="dimmed" size="sm">
              {year(show.first_air_date)}
            </Text>
          )}
        </Stack>
        {action && <Group justify="flex-end">{action}</Group>}
      </Stack>
    </Card>
  )
}
