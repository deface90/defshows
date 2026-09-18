import { Badge, type MantineColor } from '@mantine/core'

type Status = 'not_started' | 'airing' | 'between_seasons' | 'ended' | string

const map: Record<string, { label: string; color: MantineColor }> = {
  not_started: { label: 'Не начат', color: 'gray' },
  airing: { label: '🔴 Идёт', color: 'red' },
  between_seasons: { label: '⏸ Между сезонами', color: 'yellow' },
  ended: { label: '✅ Завершён', color: 'teal' },
}

export function AiringStatusBadge({ status }: { status: Status }) {
  const s = map[status] ?? { label: status, color: 'gray' as MantineColor }
  return (
    <Badge color={s.color} variant="light">
      {s.label}
    </Badge>
  )
}
