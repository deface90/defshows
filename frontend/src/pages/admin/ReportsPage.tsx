import {
  Anchor,
  Badge,
  Button,
  Group,
  SegmentedControl,
  Stack,
  Table,
  Text,
  Title,
} from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { Link } from 'react-router-dom'
import {
  getListReportsQueryKey,
  resolveReport,
  useListReports,
} from '@/shared/api/admin/endpoints'
import type { ListReportsStatus, Report, ReportReason, ReportStatus } from '@/shared/api/admin/model'
import { useDocumentTitle } from '@/shared/lib/useDocumentTitle'
import { EmptyState, ErrorState, LoadingState } from '@/shared/ui/states'
import { AdminNav } from './AdminNav'

const PAGE_SIZE = 50

// Status filter: "all" means no status param (backend returns every report).
type Filter = 'all' | ListReportsStatus
const FILTERS: { value: Filter; label: string }[] = [
  { value: 'open', label: 'Открытые' },
  { value: 'resolved', label: 'Решённые' },
  { value: 'dismissed', label: 'Отклонённые' },
  { value: 'all', label: 'Все' },
]

const reasonLabels: Record<ReportReason, string> = {
  spam: 'Спам',
  harassment: 'Оскорбления',
  inappropriate: 'Недопустимый контент',
  other: 'Другое',
}

const statusMeta: Record<ReportStatus, { label: string; color: string }> = {
  open: { label: 'Открыта', color: 'orange' },
  resolved: { label: 'Решена', color: 'green' },
  dismissed: { label: 'Отклонена', color: 'gray' },
}

/** ReportsPage lists moderation reports with status filter + resolve/dismiss (admin only). */
export function ReportsPage() {
  useDocumentTitle('Жалобы')
  const queryClient = useQueryClient()
  const [filter, setFilter] = useState<Filter>('open')

  const params = { status: filter === 'all' ? undefined : filter, page_size: PAGE_SIZE }
  const query = useListReports(params)
  const invalidate = () => queryClient.invalidateQueries({ queryKey: getListReportsQueryKey() })

  const resolve = useMutation({
    mutationFn: ({ id, status }: { id: number; status: 'resolved' | 'dismissed' }) =>
      resolveReport(id, { status }),
    onSuccess: () => {
      invalidate()
      notifications.show({ message: 'Жалоба обработана', color: 'green' })
    },
    onError: () => notifications.show({ message: 'Не удалось обработать жалобу', color: 'red' }),
  })

  return (
    <Stack>
      <AdminNav />
      <Title order={3}>Жалобы</Title>

      <SegmentedControl
        value={filter}
        onChange={(v) => setFilter(v as Filter)}
        data={FILTERS}
        aria-label="Фильтр по статусу"
      />

      {query.isLoading ? (
        <LoadingState />
      ) : query.isError ? (
        <ErrorState message="Не удалось загрузить жалобы" />
      ) : !query.data?.reports.length ? (
        <EmptyState title="Жалоб нет" />
      ) : (
        <Table.ScrollContainer minWidth={720}>
          <Table striped highlightOnHover verticalSpacing="sm">
            <Table.Thead>
              <Table.Tr>
                <Table.Th>Кто пожаловался</Table.Th>
                <Table.Th>На кого</Table.Th>
                <Table.Th>Причина</Table.Th>
                <Table.Th>Статус</Table.Th>
                <Table.Th />
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {query.data.reports.map((report) => (
                <ReportRow
                  key={report.id}
                  report={report}
                  onResolve={(status) => resolve.mutate({ id: report.id, status })}
                  pending={resolve.isPending}
                />
              ))}
            </Table.Tbody>
          </Table>
        </Table.ScrollContainer>
      )}
    </Stack>
  )
}

function ReportRow({
  report,
  onResolve,
  pending,
}: {
  report: Report
  onResolve: (status: 'resolved' | 'dismissed') => void
  pending: boolean
}) {
  const meta = statusMeta[report.status]
  const date = new Date(report.created_at).toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' })
  return (
    <Table.Tr>
      <Table.Td>
        <Anchor component={Link} to={`/users/${report.reporter.id}`} size="sm">
          {report.reporter.display_name}
        </Anchor>
        <Text c="dimmed" size="xs">
          {date}
        </Text>
      </Table.Td>
      <Table.Td>
        <Anchor component={Link} to={`/users/${report.target.id}`} size="sm">
          {report.target.display_name}
        </Anchor>
      </Table.Td>
      <Table.Td>
        <Text size="sm">{reasonLabels[report.reason]}</Text>
        {report.note && (
          <Text c="dimmed" size="xs" style={{ whiteSpace: 'pre-wrap' }}>
            {report.note}
          </Text>
        )}
      </Table.Td>
      <Table.Td>
        <Badge color={meta.color} variant="light">
          {meta.label}
        </Badge>
      </Table.Td>
      <Table.Td>
        {report.status === 'open' ? (
          <Group gap={4} justify="flex-end" wrap="nowrap">
            <Button
              size="xs"
              variant="light"
              color="green"
              loading={pending}
              onClick={() => onResolve('resolved')}
            >
              Решить
            </Button>
            <Button
              size="xs"
              variant="subtle"
              color="gray"
              loading={pending}
              onClick={() => onResolve('dismissed')}
            >
              Отклонить
            </Button>
          </Group>
        ) : null}
      </Table.Td>
    </Table.Tr>
  )
}
