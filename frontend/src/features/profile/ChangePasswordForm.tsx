import { zodResolver } from '@hookform/resolvers/zod'
import { Alert, Button, Card, PasswordInput, Stack, Title } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation } from '@tanstack/react-query'
import axios from 'axios'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { passwordChange } from '@/shared/api/auth/endpoints'
import { changePasswordSchema, type ChangePasswordInput } from '@/features/auth/schemas'
import { applySession } from '@/shared/auth/session'

/**
 * ChangePasswordForm lets a logged-in user change their password. The backend
 * revokes all other sessions and returns a fresh token pair, which we apply so
 * the current session stays valid.
 */
export function ChangePasswordForm() {
  const [apiError, setApiError] = useState('')
  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<ChangePasswordInput>({ resolver: zodResolver(changePasswordSchema) })

  const mutation = useMutation({
    mutationFn: (input: ChangePasswordInput) =>
      passwordChange({ current_password: input.current, new_password: input.password }),
    onSuccess: (res) => {
      applySession(res)
      reset()
      setApiError('')
      notifications.show({ message: 'Пароль изменён', color: 'green' })
    },
    onError: (err) => {
      const status = axios.isAxiosError(err) ? err.response?.status : undefined
      if (status === 401) setApiError('Неверный текущий пароль')
      else if (status === 400) setApiError('У этого аккаунта нет пароля (вход через соцсеть)')
      else setApiError('Не удалось изменить пароль')
    },
  })

  return (
    <Card withBorder padding="lg">
      <form onSubmit={handleSubmit((v) => mutation.mutate(v))} noValidate>
        <Stack>
          <Title order={4}>Смена пароля</Title>
          {apiError && <Alert color="red">{apiError}</Alert>}
          <PasswordInput
            label="Текущий пароль"
            error={errors.current?.message}
            {...register('current')}
          />
          <PasswordInput
            label="Новый пароль"
            error={errors.password?.message}
            {...register('password')}
          />
          <PasswordInput
            label="Повторите новый пароль"
            error={errors.confirm?.message}
            {...register('confirm')}
          />
          <Button type="submit" loading={mutation.isPending} style={{ alignSelf: 'flex-start' }}>
            Сменить пароль
          </Button>
        </Stack>
      </form>
    </Card>
  )
}
