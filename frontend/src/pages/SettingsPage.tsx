import { Divider, Stack, Title } from '@mantine/core'
import { PrefsForm } from '@/features/notif-prefs/PrefsForm'
import { BlockedUsersList } from '@/features/profile/BlockedUsersList'
import { DisplayNameForm } from '@/features/profile/DisplayNameForm'
import { VisibilityToggle } from '@/features/profile/VisibilityToggle'
import { ChangePasswordForm } from '@/features/profile/ChangePasswordForm'
import { DeleteAccountCard } from '@/features/profile/DeleteAccountCard'
import { LinkTelegramButton } from '@/features/telegram-link/LinkTelegramButton'
import { useDocumentTitle } from '@/shared/lib/useDocumentTitle'

/** SettingsPage bundles profile visibility, display name, notification prefs, timezone, Telegram linking, blocked users, and account deletion. */
export function SettingsPage() {
  useDocumentTitle('Настройки')
  return (
    <Stack>
      <Title order={3}>Настройки</Title>
      <VisibilityToggle />
      <DisplayNameForm />
      <ChangePasswordForm />
      <PrefsForm />
      <Divider my="sm" />
      <LinkTelegramButton />
      <Divider my="sm" />
      <BlockedUsersList />
      <Divider my="sm" />
      <DeleteAccountCard />
    </Stack>
  )
}
