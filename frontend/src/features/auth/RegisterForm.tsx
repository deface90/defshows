import { zodResolver } from '@hookform/resolvers/zod'
import { Alert, Button, PasswordInput, Stack, TextInput } from '@mantine/core'
import { useMutation } from '@tanstack/react-query'
import { isAxiosError } from 'axios'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useNavigate } from 'react-router-dom'
import { register as registerUser } from '@/shared/api/auth/endpoints'
import { applySession } from '@/shared/auth/session'
import { registerSchema, type RegisterInput } from './schemas'

/**
 * RegisterForm creates an account. By default it navigates home on success;
 * pass `onAuthed` (e.g. from the AuthModal) to stay in place and let the caller
 * finish a pending action instead.
 */
export function RegisterForm({ onAuthed }: { onAuthed?: () => void } = {}) {
  const navigate = useNavigate()
  const [apiError, setApiError] = useState('')
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<RegisterInput>({ resolver: zodResolver(registerSchema) })

  const mutation = useMutation({
    mutationFn: (input: RegisterInput) => registerUser({ email: input.email, password: input.password }),
    onSuccess: (res) => {
      applySession(res)
      if (onAuthed) onAuthed()
      else navigate('/', { replace: true })
    },
    onError: (err) => {
      if (isAxiosError(err) && err.response?.status === 409) {
        setApiError('Этот email уже зарегистрирован')
      } else {
        setApiError('Не удалось зарегистрироваться')
      }
    },
  })

  return (
    <form onSubmit={handleSubmit((v) => mutation.mutate(v))} noValidate>
      <Stack>
        {apiError && <Alert color="red">{apiError}</Alert>}
        <TextInput label="Email" type="email" error={errors.email?.message} {...register('email')} />
        <PasswordInput label="Пароль" error={errors.password?.message} {...register('password')} />
        <PasswordInput label="Повторите пароль" error={errors.confirm?.message} {...register('confirm')} />
        <Button type="submit" loading={mutation.isPending}>
          Зарегистрироваться
        </Button>
      </Stack>
    </form>
  )
}
