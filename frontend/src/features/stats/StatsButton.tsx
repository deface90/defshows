import { Button, Group, Modal, SimpleGrid, Stack, Text } from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { useGetStats } from '@/shared/api/tracking/endpoints'
import type { Stats } from '@/shared/api/tracking/model'
import { ErrorState, LoadingState } from '@/shared/ui/states'
import { formatWatchTime } from './formatWatchTime'

function StatRow({ label, value }: { label: string; value: string | number }) {
  return (
    <Stack gap={2}>
      <Text fz={26} fw={700} lh={1}>
        {value}
      </Text>
      <Text c="dimmed" fz="sm">
        {label}
      </Text>
    </Stack>
  )
}

function StatsContent({ stats }: { stats: Stats }) {
  return (
    <SimpleGrid cols={{ base: 2, xs: 3 }} spacing="lg">
      <StatRow label="Добавлено сериалов" value={stats.shows_tracked} />
      <StatRow label="Просмотрено сериалов" value={stats.shows_completed} />
      <StatRow label="Просмотрено сезонов" value={stats.seasons_watched} />
      <StatRow label="Просмотрено эпизодов" value={stats.episodes_watched} />
      <StatRow label="Потрачено времени" value={formatWatchTime(stats.minutes_watched)} />
    </SimpleGrid>
  )
}

/** StatsButton opens a modal with the user's aggregate viewing statistics.
 * The query is deferred until the modal is first opened. */
export function StatsButton() {
  const [opened, { open, close }] = useDisclosure(false)
  const query = useGetStats({ query: { enabled: opened } })

  return (
    <>
      <Button variant="default" onClick={open}>
        📊 Статистика
      </Button>
      <Modal opened={opened} onClose={close} title="Статистика просмотра" centered size="lg">
        {query.isLoading && <LoadingState />}
        {query.isError && <ErrorState message="Не удалось загрузить статистику" />}
        {query.isSuccess && (
          <Group>
            <StatsContent stats={query.data} />
          </Group>
        )}
      </Modal>
    </>
  )
}
