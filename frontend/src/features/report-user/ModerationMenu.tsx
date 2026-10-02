import { ActionIcon, Menu } from '@mantine/core'
import { useState } from 'react'
import { useBlockMutations } from '@/features/block-user/BlockButton'
import { ReportModal } from './ReportModal'

/**
 * ModerationMenu is the overflow menu shown next to another user (profile header,
 * feed-card actor): Block + Report. Blocking tears down follow edges server-side;
 * `onBlocked` lets the caller refresh any locally-held state (e.g. the activity feed).
 */
export function ModerationMenu({
  userId,
  onBlocked,
}: {
  userId: number
  onBlocked?: () => void
}) {
  const [reportOpen, setReportOpen] = useState(false)
  const { block, busy } = useBlockMutations(userId, onBlocked)

  return (
    <>
      <Menu position="bottom-end" withinPortal>
        <Menu.Target>
          <ActionIcon variant="subtle" color="gray" aria-label="Действия с пользователем">
            ⋯
          </ActionIcon>
        </Menu.Target>
        <Menu.Dropdown>
          <Menu.Item disabled={busy} onClick={() => block.mutate()}>
            Заблокировать
          </Menu.Item>
          <Menu.Item color="red" onClick={() => setReportOpen(true)}>
            Пожаловаться
          </Menu.Item>
        </Menu.Dropdown>
      </Menu>
      <ReportModal userId={userId} opened={reportOpen} onClose={() => setReportOpen(false)} />
    </>
  )
}
