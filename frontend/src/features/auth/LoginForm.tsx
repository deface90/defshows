import { zodResolver } from '@hookform/resolvers/zod'
import { Alert, Button, PasswordInput, Stack, TextInput } from '@mantine/core'
import { useMutation } from '@tanstack/react-query'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useNavigate } from 'react-router-dom'
import { login } from '@/shared/api/auth/endpoints'
import { applySession } from '@/shared/auth/session'
import { loginSchema, type LoginInput } from './schemas'

export function LoginForm() {
  const navigate = useNavigate()
  const [apiError, setApiError] = useState('')
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<LoginInput>({ resolver: zodResolver(loginSchema) })

  const mutation = useMutation({
    mutationFn: (input: LoginInput) => login(input),
    onSuccess: (res) => {
      applySession(res)
      navigate('/', { replace: true })
    },
    onError: () => setApiError('Неверный email или пароль'),
  })

  return (
    <form onSubmit={handleSubmit((v) => mutation.mutate(v))} noValidate>
      <Stack>
        {apiError && <Alert color="red">{apiError}</Alert>}
        <TextInput
          label="Email"
          type="email"
          error={errors.email?.message}
          {...register('email')}
        />
        <PasswordInput label="Пароль" error={errors.password?.message} {...register('password')} />
        <Button type="submit" loading={mutation.isPending}>
          Войти
        </Button>
      </Stack>
    </form>
  )
}
