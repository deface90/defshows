import { Text } from '@mantine/core'

export function NextEpisodeInfo({ date }: { date?: string | null }) {
  if (!date) return null
  return (
    <Text size="sm" c="brand">
      Следующий эпизод: {date}
    </Text>
  )
}
