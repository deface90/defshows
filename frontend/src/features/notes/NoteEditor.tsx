import { zodResolver } from '@hookform/resolvers/zod'
import { Button, Group, NumberInput, Select, Stack, Textarea } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Controller, useForm } from 'react-hook-form'
import { createNote, getListNotesQueryKey } from '@/shared/api/notes/endpoints'
import type { CreateNoteRequestScope } from '@/shared/api/notes/model'
import { noteSchema, type NoteInput } from './schemas'

const scopeOptions = [
  { value: 'show', label: 'Весь сериал' },
  { value: 'season', label: 'Сезон' },
  { value: 'episode', label: 'Эпизод' },
]

/** NoteEditor is the create form for a private note scoped to a show/season/episode. */
export function NoteEditor({ showId }: { showId: number }) {
  const queryClient = useQueryClient()
  const {
    control,
    register,
    handleSubmit,
    watch,
    reset,
    formState: { errors },
  } = useForm<NoteInput>({
    resolver: zodResolver(noteSchema),
    defaultValues: { scope: 'show', season_number: null, episode_number: null, body: '' },
  })
  const scope = watch('scope')

  const mutation = useMutation({
    mutationFn: (input: NoteInput) =>
      createNote({
        show_id: showId,
        scope: input.scope as CreateNoteRequestScope,
        season_number: input.scope === 'show' ? null : input.season_number,
        episode_number: input.scope === 'episode' ? input.episode_number : null,
        body: input.body,
      }),
    onSuccess: () => {
      reset({ scope, season_number: null, episode_number: null, body: '' })
      queryClient.invalidateQueries({ queryKey: getListNotesQueryKey({ show_id: showId }) })
    },
    onError: () => notifications.show({ message: 'Не удалось сохранить заметку', color: 'red' }),
  })

  return (
    <form onSubmit={handleSubmit((v) => mutation.mutate(v))} noValidate>
      <Stack gap="xs">
        <Group align="flex-start" gap="xs">
          <Controller
            control={control}
            name="scope"
            render={({ field }) => (
              <Select
                w={140}
                data={scopeOptions}
                value={field.value}
                onChange={(v) => v && field.onChange(v)}
                aria-label="Область заметки"
              />
            )}
          />
          {scope !== 'show' && (
            <Controller
              control={control}
              name="season_number"
              render={({ field }) => (
                <NumberInput
                  w={110}
                  min={1}
                  placeholder="Сезон"
                  aria-label="Номер сезона"
                  value={field.value ?? ''}
                  onChange={(v) => field.onChange(v === '' ? null : Number(v))}
                  error={errors.season_number?.message}
                />
              )}
            />
          )}
          {scope === 'episode' && (
            <Controller
              control={control}
              name="episode_number"
              render={({ field }) => (
                <NumberInput
                  w={110}
                  min={1}
                  placeholder="Эпизод"
                  aria-label="Номер эпизода"
                  value={field.value ?? ''}
                  onChange={(v) => field.onChange(v === '' ? null : Number(v))}
                  error={errors.episode_number?.message}
                />
              )}
            />
          )}
        </Group>
        <Textarea
          placeholder="Ваша приватная заметка…"
          aria-label="Текст заметки"
          rows={3}
          error={errors.body?.message}
          {...register('body')}
        />
        <Group justify="flex-end">
          <Button type="submit" loading={mutation.isPending}>
            Добавить заметку
          </Button>
        </Group>
      </Stack>
    </form>
  )
}
