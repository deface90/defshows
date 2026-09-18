import { Badge, Tooltip } from '@mantine/core'
import type { Progress } from '@/shared/api/tracking/model'

export function ProgressBadge({ progress }: { progress: Progress }) {
  const { watched, total } = progress
  const complete = total > 0 && watched >= total
  return (
    <Tooltip label={complete ? 'Всё просмотрено' : `Осталось ${Math.max(total - watched, 0)}`}>
      <Badge variant="light" color={complete ? 'teal' : 'brand'}>
        {watched}/{total}
      </Badge>
    </Tooltip>
  )
}
