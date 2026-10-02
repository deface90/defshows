import { zodResolver } from '@hookform/resolvers/zod'
import { Button, Group, Modal, Select, Stack, Textarea } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation } from '@tanstack/react-query'
import { Controller, useForm } from 'react-hook-form'
import { z } from 'zod'
import { createReport } from '@/shared/api/social/endpoints'
import { ReportRequestReason } from '@/shared/api/social/model'

const reasonOptions = [
  { value: ReportRequestReason.spam, label: 'Спам' },
  { value: ReportRequestReason.harassment, label: 'Оскорбления' },
  { value: ReportRequestReason.inappropriate, label: 'Неприемлемый контент' },
  { value: ReportRequestReason.other, label: 'Другое' },
]

const reportSchema = z.object({
  reason: z.enum(['spam', 'harassment', 'inappropriate', 'other'], {
    message: 'Выберите причину',
  }),
  note: z.string().max(1000, 'Не более 1000 символов').optional(),
})

type ReportInput = z.infer<typeof reportSchema>

/**
 * ReportModal collects a reason (required) and an optional note, then files a report
 * via POST /me/reports. It is controlled by the caller via `opened`/`onClose`.
 */
export function ReportModal({
  userId,
  opened,
  onClose,
}: {
  userId: number
  opened: boolean
  onClose: () => void
}) {
  const {
    control,
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<ReportInput>({
    resolver: zodResolver(reportSchema),
    defaultValues: { note: '' },
  })

  const mutation = useMutation({
    mutationFn: (input: ReportInput) =>
      createReport({
        target_user_id: userId,
        reason: input.reason,
        note: input.note || undefined,
      }),
    onSuccess: () => {
      notifications.show({ message: 'Жалоба отправлена' })
      reset({ note: '' })
      onClose()
    },
    onError: () => notifications.show({ message: 'Не удалось отправить жалобу', color: 'red' }),
  })

  return (
    <Modal opened={opened} onClose={onClose} title="Пожаловаться на пользователя" centered>
      <form onSubmit={handleSubmit((v) => mutation.mutate(v))} noValidate>
        <Stack gap="md">
          <Controller
            control={control}
            name="reason"
            render={({ field }) => (
              <Select
                label="Причина"
                placeholder="Выберите причину"
                data={reasonOptions}
                value={field.value ?? null}
                onChange={(v) => field.onChange(v ?? undefined)}
                error={errors.reason?.message}
              />
            )}
          />
          <Textarea
            label="Комментарий (необязательно)"
            placeholder="Что случилось?"
            rows={3}
            error={errors.note?.message}
            {...register('note')}
          />
          <Group justify="flex-end">
            <Button variant="default" onClick={onClose}>
              Отмена
            </Button>
            <Button type="submit" loading={mutation.isPending}>
              Отправить
            </Button>
          </Group>
        </Stack>
      </form>
    </Modal>
  )
}
