import { zodResolver } from '@hookform/resolvers/zod'
import { Alert, Button, Stack, Text, TextInput } from '@mantine/core'
import { useMutation } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { passwordForgot } from '@/shared/api/auth/endpoints'
import { forgotPasswordSchema, type ForgotPasswordInput } from './schemas'

/**
 * ForgotPasswordForm requests a password-reset email. The response is
 * intentionally the same whether or not the email is registered, so the success
 * message is neutral (no account enumeration).
 */
export function ForgotPasswordForm() {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<ForgotPasswordInput>({ resolver: zodResolver(forgotPasswordSchema) })

  const mutation = useMutation({
    mutationFn: (input: ForgotPasswordInput) => passwordForgot(input),
    // Errors surface inline below; opt out of the global mutation toast.
    onError: () => {},
  })

  if (mutation.isSuccess) {
    return (
      <Alert color="green">
        Если такой аккаунт существует, мы отправили на его почту ссылку для сброса пароля.
        Проверьте входящие.
      </Alert>
    )
  }

  return (
    <form onSubmit={handleSubmit((v) => mutation.mutate(v))} noValidate>
      <Stack>
        <Text c="dimmed" size="sm">
          Введите email — мы пришлём ссылку для сброса пароля.
        </Text>
        {mutation.isError && <Alert color="red">Не удалось отправить письмо. Попробуйте позже.</Alert>}
        <TextInput
          label="Email"
          type="email"
          error={errors.email?.message}
          {...register('email')}
        />
        <Button type="submit" loading={mutation.isPending}>
          Отправить ссылку
        </Button>
      </Stack>
    </form>
  )
}
