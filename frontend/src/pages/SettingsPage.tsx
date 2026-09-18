import { Divider, Stack, Title } from '@mantine/core'
import { PrefsForm } from '@/features/notif-prefs/PrefsForm'
import { VisibilityToggle } from '@/features/profile/VisibilityToggle'
import { LinkTelegramButton } from '@/features/telegram-link/LinkTelegramButton'

/** SettingsPage bundles profile visibility, notification prefs, timezone, and Telegram linking. */
export function SettingsPage() {
  return (
    <Stack>
      <Title order={3}>Настройки</Title>
      <VisibilityToggle />
      <PrefsForm />
      <Divider my="sm" />
      <LinkTelegramButton />
    </Stack>
  )
}
