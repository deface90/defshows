import { ActionIcon, Anchor, Badge, Button, Group, Modal, Stack, Table, Title } from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import {
  createDubbingStudio,
  deleteDubbingStudio,
  getListDubbingStudiosQueryKey,
  updateDubbingStudio,
  useListDubbingStudios,
} from '@/shared/api/admin/endpoints'
import type { DubbingStudio } from '@/shared/api/admin/model'
import { DubbingForm } from '@/features/admin/DubbingForm'
import type { DubbingStudioInput } from '@/features/admin/schemas'
import { EmptyState, ErrorState, LoadingState } from '@/shared/ui/states'

/** AdminDubbingPage manages the DubbingStudio reference table (admin only). */
export function AdminDubbingPage() {
  const queryClient = useQueryClient()
  const query = useListDubbingStudios()
  const invalidate = () => queryClient.invalidateQueries({ queryKey: getListDubbingStudiosQueryKey() })

  const [opened, { open, close }] = useDisclosure(false)
  const [editing, setEditing] = useState<DubbingStudio | null>(null)

  const openCreate = () => {
    setEditing(null)
    open()
  }
  const openEdit = (studio: DubbingStudio) => {
    setEditing(studio)
    open()
  }

  const save = useMutation({
    mutationFn: (values: DubbingStudioInput) => {
      const body = { name: values.name, site_url: values.site_url || undefined, active: values.active }
      return editing ? updateDubbingStudio(editing.id, body) : createDubbingStudio(body)
    },
    onSuccess: () => {
      close()
      invalidate()
      notifications.show({ message: 'Сохранено', color: 'green' })
    },
    onError: () => notifications.show({ message: 'Не удалось сохранить', color: 'red' }),
  })

  const remove = useMutation({
    mutationFn: (id: number) => deleteDubbingStudio(id),
    onSuccess: invalidate,
    onError: () => notifications.show({ message: 'Не удалось удалить', color: 'red' }),
  })

  return (
    <Stack>
      <Group justify="space-between">
        <Title order={3}>Студии озвучки</Title>
        <Button onClick={openCreate}>Добавить</Button>
      </Group>

      {query.isLoading ? (
        <LoadingState />
      ) : query.isError ? (
        <ErrorState message="Не удалось загрузить список" />
      ) : !query.data?.studios.length ? (
        <EmptyState title="Студий пока нет" />
      ) : (
        <Table striped highlightOnHover>
          <Table.Thead>
            <Table.Tr>
              <Table.Th>Название</Table.Th>
              <Table.Th>Сайт</Table.Th>
              <Table.Th>Статус</Table.Th>
              <Table.Th />
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {query.data.studios.map((studio) => (
              <Table.Tr key={studio.id}>
                <Table.Td>{studio.name}</Table.Td>
                <Table.Td>
                  {studio.site_url ? (
                    <Anchor href={studio.site_url} target="_blank" rel="noreferrer" size="sm">
                      {studio.site_url}
                    </Anchor>
                  ) : (
                    '—'
                  )}
                </Table.Td>
                <Table.Td>
                  <Badge color={studio.active ? 'green' : 'gray'} variant="light">
                    {studio.active ? 'Активна' : 'Отключена'}
                  </Badge>
                </Table.Td>
                <Table.Td>
                  <Group gap={4} justify="flex-end">
                    <ActionIcon variant="subtle" onClick={() => openEdit(studio)} aria-label="Редактировать">
                      ✎
                    </ActionIcon>
                    <ActionIcon
                      variant="subtle"
                      color="red"
                      onClick={() => remove.mutate(studio.id)}
                      aria-label="Удалить"
                    >
                      ✕
                    </ActionIcon>
                  </Group>
                </Table.Td>
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>
      )}

      <Modal opened={opened} onClose={close} title={editing ? 'Редактировать студию' : 'Новая студия'}>
        <DubbingForm
          key={editing?.id ?? 'new'}
          defaultValues={
            editing
              ? { name: editing.name, site_url: editing.site_url ?? '', active: editing.active }
              : undefined
          }
          submitLabel={editing ? 'Сохранить' : 'Создать'}
          isPending={save.isPending}
          onSubmit={(v) => save.mutate(v)}
          onCancel={close}
        />
      </Modal>
    </Stack>
  )
}
