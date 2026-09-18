import { zodResolver } from '@hookform/resolvers/zod'
import { Button, Group, NumberInput, Stack, Switch, TextInput, Title } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Controller, useForm } from 'react-hook-form'
import { z } from 'zod'
import {
  getPrefs,
  getGetPrefsQueryKey,
  updatePrefs,
  useGetPrefs,
} from '@/shared/api/notifications/endpoints'
import {
  getGetSettingsQueryKey,
  updateSettings,
  useGetSettings,
} from '@/shared/api/tracking/endpoints'
import { ErrorState, LoadingState } from '@/shared/ui/states'

const prefsSchema = z.object({
  episode_release: z.boolean(),
  season_start: z.boolean(),
  weekly_digest: z.boolean(),
  lead_time_hours: z.number().int().min(0).max(168),
  timezone: z.string().trim().min(1, 'Укажите таймзону'),
})
type PrefsInput = z.infer<typeof prefsSchema>

/** PrefsForm edits notification toggles + lead time (prefs) and the user timezone (settings). */
export function PrefsForm() {
  const queryClient = useQueryClient()
  const prefsQuery = useGetPrefs()
  const settingsQuery = useGetSettings()

  const {
    control,
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<PrefsInput>({
    resolver: zodResolver(prefsSchema),
    values:
      prefsQuery.data && settingsQuery.data
        ? {
            episode_release: prefsQuery.data.episode_release,
            season_start: prefsQuery.data.season_start,
            weekly_digest: prefsQuery.data.weekly_digest,
            lead_time_hours: prefsQuery.data.lead_time_hours,
            timezone: settingsQuery.data.timezone,
          }
        : undefined,
  })

  const mutation = useMutation({
    mutationFn: async (input: PrefsInput) => {
      await Promise.all([
        updatePrefs({
          episode_release: input.episode_release,
          season_start: input.season_start,
          weekly_digest: input.weekly_digest,
          lead_time_hours: input.lead_time_hours,
        }),
        updateSettings({ timezone: input.timezone }),
      ])
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: getGetPrefsQueryKey() })
      queryClient.invalidateQueries({ queryKey: getGetSettingsQueryKey() })
      notifications.show({ message: 'Настройки сохранены', color: 'green' })
    },
    onError: () => notifications.show({ message: 'Не удалось сохранить настройки', color: 'red' }),
  })

  if (prefsQuery.isLoading || settingsQuery.isLoading) return <LoadingState />
  if (prefsQuery.isError || settingsQuery.isError) return <ErrorState message="Не удалось загрузить настройки" />

  return (
    <form onSubmit={handleSubmit((v) => mutation.mutate(v))} noValidate>
      <Stack>
        <Title order={4}>Уведомления</Title>
        <Controller
          control={control}
          name="episode_release"
          render={({ field }) => (
            <Switch
              label="Новый эпизод"
              checked={field.value}
              onChange={(e) => field.onChange(e.currentTarget.checked)}
            />
          )}
        />
        <Controller
          control={control}
          name="season_start"
          render={({ field }) => (
            <Switch
              label="Старт сезона"
              checked={field.value}
              onChange={(e) => field.onChange(e.currentTarget.checked)}
            />
          )}
        />
        <Controller
          control={control}
          name="weekly_digest"
          render={({ field }) => (
            <Switch
              label="Недельный дайджест"
              checked={field.value}
              onChange={(e) => field.onChange(e.currentTarget.checked)}
            />
          )}
        />
        <Controller
          control={control}
          name="lead_time_hours"
          render={({ field }) => (
            <NumberInput
              label="За сколько часов предупреждать"
              min={0}
              max={168}
              w={260}
              value={field.value}
              onChange={(v) => field.onChange(v === '' ? 0 : Number(v))}
              error={errors.lead_time_hours?.message}
            />
          )}
        />

        <Title order={4} mt="md">
          Таймзона
        </Title>
        <TextInput
          label="Таймзона (IANA)"
          placeholder="Europe/Moscow"
          w={260}
          error={errors.timezone?.message}
          {...register('timezone')}
        />

        <Group>
          <Button type="submit" loading={mutation.isPending}>
            Сохранить
          </Button>
        </Group>
      </Stack>
    </form>
  )
}
