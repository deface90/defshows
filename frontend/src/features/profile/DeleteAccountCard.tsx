import { Alert, Button, Card, Group, Modal, Stack, Text, Title } from '@mantine/core'
import { useDisclosure } from '@mantine/hooks'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { deleteMe } from '@/shared/api/auth/endpoints'
import { useAuthStore } from '@/shared/auth/authStore'

/**
 * DeleteAccountCard permanently deletes the current account (and, via FK
 * cascade on the backend, all of its data) after an explicit confirmation.
 */
export function DeleteAccountCard() {
  const [opened, modal] = useDisclosure(false)
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const mutation = useMutation({
    mutationFn: () => deleteMe(),
    onSuccess: () => {
      modal.close()
      useAuthStore.getState().clear()
      queryClient.clear()
      notifications.show({ message: 'Аккаунт удалён' })
      navigate('/login', { replace: true })
    },
  })

  return (
    <Card withBorder padding="lg">
      <Stack>
        <Title order={4}>Удаление аккаунта</Title>
        <Text size="sm" c="dimmed">
          Аккаунт и все данные — коллекция, отметки просмотра, заметки, уведомления — будут удалены
          без возможности восстановления.
        </Text>
        <Button color="red" variant="light" onClick={modal.open} style={{ alignSelf: 'flex-start' }}>
          Удалить аккаунт
        </Button>
      </Stack>
      <Modal opened={opened} onClose={modal.close} centered title="Удалить аккаунт?">
        <Stack>
          <Text size="sm">Это действие нельзя отменить.</Text>
          {mutation.isError && <Alert color="red">Не удалось удалить аккаунт</Alert>}
          <Group justify="flex-end">
            <Button variant="default" onClick={modal.close}>
              Отмена
            </Button>
            <Button color="red" loading={mutation.isPending} onClick={() => mutation.mutate()}>
              Удалить навсегда
            </Button>
          </Group>
        </Stack>
      </Modal>
    </Card>
  )
}
