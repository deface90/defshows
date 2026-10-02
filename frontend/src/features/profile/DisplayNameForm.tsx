import { zodResolver } from '@hookform/resolvers/zod'
import { Button, Card, Group, Stack, TextInput, Title } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { getGetMeQueryKey } from '@/shared/api/auth/endpoints'
import {
  getGetSettingsQueryKey,
  updateSettings,
  useGetSettings,
} from '@/shared/api/tracking/endpoints'
import { ErrorState, LoadingState } from '@/shared/ui/states'

const schema = z.object({
  display_name: z
    .string()
    .trim()
    .max(50, 'Не более 50 символов')
    // blank is allowed and resets the name to the derived default
    .refine((v) => v.length === 0 || v.length >= 1, { message: 'Укажите имя' }),
})

type Input = z.infer<typeof schema>

/**
 * DisplayNameForm edits the user's display name via updateSettings. A blank value is
 * allowed and resets the name to the server-derived default. Saving invalidates the
 * settings and getMe queries so dependent views pick up the new name.
 */
export function DisplayNameForm() {
  const queryClient = useQueryClient()
  const settingsQuery = useGetSettings()

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isDirty },
  } = useForm<Input>({
    resolver: zodResolver(schema),
    defaultValues: { display_name: '' },
  })

  // Seed the field once settings load (and after a successful save re-sync).
  useEffect(() => {
    if (settingsQuery.data) reset({ display_name: settingsQuery.data.display_name })
  }, [settingsQuery.data, reset])

  const mutation = useMutation({
    mutationFn: (input: Input) =>
      updateSettings({
        timezone: settingsQuery.data?.timezone ?? 'UTC',
        is_public: settingsQuery.data?.is_public ?? false,
        display_name: input.display_name,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: getGetSettingsQueryKey() })
      queryClient.invalidateQueries({ queryKey: getGetMeQueryKey() })
      notifications.show({ message: 'Имя обновлено', color: 'green' })
    },
    onError: () => notifications.show({ message: 'Не удалось обновить имя', color: 'red' }),
  })

  if (settingsQuery.isLoading) return <LoadingState />
  if (settingsQuery.isError || !settingsQuery.data)
    return <ErrorState message="Не удалось загрузить настройки профиля" />

  return (
    <Card withBorder padding="lg">
      <form onSubmit={handleSubmit((v) => mutation.mutate(v))} noValidate>
        <Stack gap="sm">
          <Title order={4}>Отображаемое имя</Title>
          <TextInput
            label="Имя"
            placeholder="Как вас называть"
            description="Оставьте пустым, чтобы вернуть имя по умолчанию."
            error={errors.display_name?.message}
            {...register('display_name')}
          />
          <Group justify="flex-end">
            <Button type="submit" loading={mutation.isPending} disabled={!isDirty}>
              Сохранить
            </Button>
          </Group>
        </Stack>
      </form>
    </Card>
  )
}
