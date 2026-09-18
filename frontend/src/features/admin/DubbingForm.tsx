import { zodResolver } from '@hookform/resolvers/zod'
import { Button, Group, Stack, Switch, TextInput } from '@mantine/core'
import { Controller, useForm } from 'react-hook-form'
import { dubbingStudioSchema, type DubbingStudioInput } from './schemas'

interface DubbingFormProps {
  defaultValues?: Partial<DubbingStudioInput>
  submitLabel: string
  isPending?: boolean
  onSubmit: (values: DubbingStudioInput) => void
  onCancel: () => void
}

/** DubbingForm is the shared create/edit form for a DubbingStudio. */
export function DubbingForm({ defaultValues, submitLabel, isPending, onSubmit, onCancel }: DubbingFormProps) {
  const {
    control,
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<DubbingStudioInput>({
    resolver: zodResolver(dubbingStudioSchema),
    defaultValues: { name: '', site_url: '', active: true, ...defaultValues },
  })

  return (
    <form onSubmit={handleSubmit(onSubmit)} noValidate>
      <Stack>
        <TextInput label="Название" error={errors.name?.message} {...register('name')} />
        <TextInput label="Сайт" placeholder="https://…" error={errors.site_url?.message} {...register('site_url')} />
        <Controller
          control={control}
          name="active"
          render={({ field }) => (
            <Switch
              label="Активна"
              checked={field.value}
              onChange={(e) => field.onChange(e.currentTarget.checked)}
            />
          )}
        />
        <Group justify="flex-end">
          <Button variant="default" onClick={onCancel}>
            Отмена
          </Button>
          <Button type="submit" loading={isPending}>
            {submitLabel}
          </Button>
        </Group>
      </Stack>
    </form>
  )
}
