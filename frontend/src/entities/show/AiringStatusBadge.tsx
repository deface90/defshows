import { Badge, type MantineColor } from '@mantine/core'

type Status = 'not_started' | 'airing' | 'between_seasons' | 'ended' | string

const map: Record<string, { label: string; color: MantineColor }> = {
  not_started: { label: '◷ Не начат', color: 'gray' },
  airing: { label: '▶ Идёт', color: 'green' },
  between_seasons: { label: 'Ⅱ Между сезонами', color: 'yellow' },
  ended: { label: '■ Завершён', color: 'red' },
}

export function AiringStatusBadge({ status }: { status: Status }) {
  const s = map[status] ?? { label: status, color: 'gray' as MantineColor }
  return (
    <Badge color={s.color} variant="light">
      {s.label}
    </Badge>
  )
}
