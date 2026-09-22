import { zodResolver } from '@hookform/resolvers/zod'
import { Alert, Button, PasswordInput, Stack } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { passwordReset } from '@/shared/api/auth/endpoints'
import { resetPasswordSchema, type ResetPasswordInput } from './schemas'

/**
 * ResetPasswordForm sets a new password using the token from the emailed link
 * (`/reset-password?token=...`). On success it redirects to the login page.
 */
export function ResetPasswordForm() {
  const [params] = useSearchParams()
  const token = params.get('token') ?? ''
  const navigate = useNavigate()
  const [apiError, setApiError] = useState('')
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<ResetPasswordInput>({ resolver: zodResolver(resetPasswordSchema) })

  const mutation = useMutation({
    mutationFn: (input: ResetPasswordInput) =>
      passwordReset({ token, new_password: input.password }),
    onSuccess: () => {
      notifications.show({ message: 'Пароль обновлён. Войдите с новым паролем.', color: 'green' })
      navigate('/login', { replace: true })
    },
    onError: () => setApiError('Ссылка недействительна или устарела. Запросите сброс заново.'),
  })

  if (!token) {
    return <Alert color="red">Ссылка недействительна: отсутствует токен сброса.</Alert>
  }

  return (
    <form onSubmit={handleSubmit((v) => mutation.mutate(v))} noValidate>
      <Stack>
        {apiError && <Alert color="red">{apiError}</Alert>}
        <PasswordInput
          label="Новый пароль"
          error={errors.password?.message}
          {...register('password')}
        />
        <PasswordInput
          label="Повторите пароль"
          error={errors.confirm?.message}
          {...register('confirm')}
        />
        <Button type="submit" loading={mutation.isPending}>
          Сохранить пароль
        </Button>
      </Stack>
    </form>
  )
}
