import { Divider, Stack, Title } from '@mantine/core'
import { PrefsForm } from '@/features/notif-prefs/PrefsForm'
import { VisibilityToggle } from '@/features/profile/VisibilityToggle'
import { ChangePasswordForm } from '@/features/profile/ChangePasswordForm'
import { DeleteAccountCard } from '@/features/profile/DeleteAccountCard'
import { LinkTelegramButton } from '@/features/telegram-link/LinkTelegramButton'
import { useDocumentTitle } from '@/shared/lib/useDocumentTitle'

/** SettingsPage bundles profile visibility, notification prefs, timezone, Telegram linking, and account deletion. */
export function SettingsPage() {
  useDocumentTitle('Настройки')
  return (
    <Stack>
      <Title order={3}>Настройки</Title>
      <VisibilityToggle />
      <ChangePasswordForm />
      <PrefsForm />
      <Divider my="sm" />
      <LinkTelegramButton />
      <Divider my="sm" />
      <DeleteAccountCard />
    </Stack>
  )
}
