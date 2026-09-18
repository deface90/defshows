import { ActionIcon, Anchor, Button, Group, Stack, Text, TextInput } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import {
  addLink,
  deleteLink,
  getGetTrackedQueryKey,
  getListLinksQueryKey,
  updateLink,
  updateShow,
  useListLinks,
} from '@/shared/api/tracking/endpoints'
import type { AddLinkRequestKind, Link } from '@/shared/api/tracking/model'

const kinds: { value: AddLinkRequestKind; label: string }[] = [
  { value: 'download', label: 'Скачать' },
  { value: 'streaming', label: 'Смотреть' },
  { value: 'wiki', label: 'Wiki' },
  { value: 'imdb', label: 'IMDB' },
  { value: 'kinopoisk', label: 'Kinopoisk' },
]

function webURL(value: string): boolean {
  try {
    const url = new URL(value)
    return (url.protocol === 'https:' || url.protocol === 'http:') && !!url.hostname
  } catch {
    return false
  }
}

function EditableRow({ label, value, isLink = false, onSave }: {
  label: string
  value: string
  isLink?: boolean
  onSave: (value: string) => Promise<void>
}) {
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(value)
  const mutation = useMutation({
    mutationFn: onSave,
    onSuccess: () => setEditing(false),
    onError: () => notifications.show({ message: `Не удалось сохранить: ${label}`, color: 'red' }),
  })

  return (
    <Group gap="xs" align="center" wrap="wrap">
      <Text size="sm" fw={600} w={100}>{label}</Text>
      {editing ? (
        <Group
          component="form"
          gap="xs"
          style={{ flex: 1, minWidth: 200 }}
          onSubmit={(event) => {
            event.preventDefault()
            if (!mutation.isPending) mutation.mutate(draft.trim())
          }}
        >
          <TextInput
            aria-label={label}
            value={draft}
            autoFocus
            disabled={mutation.isPending}
            style={{ flex: 1, minWidth: 120 }}
            onChange={(event) => setDraft(event.currentTarget.value)}
            onKeyDown={(event) => {
              if (event.key === 'Escape' && !mutation.isPending) setEditing(false)
            }}
          />
          <ActionIcon type="submit" variant="light" loading={mutation.isPending} aria-label={`Сохранить: ${label}`} title="Сохранить">✓</ActionIcon>
          <ActionIcon variant="subtle" disabled={mutation.isPending} onClick={() => setEditing(false)} aria-label={`Отменить: ${label}`} title="Отменить">✕</ActionIcon>
        </Group>
      ) : (
        <>
          {isLink && webURL(value) ? (
            <Anchor href={value} target="_blank" rel="noopener noreferrer" size="sm" style={{ overflowWrap: 'anywhere', minWidth: 0 }}>
              {value}
            </Anchor>
          ) : (
            <Text size="sm" c={value ? undefined : 'dimmed'} style={{ overflowWrap: 'anywhere', minWidth: 0 }}>{value || 'нет'}</Text>
          )}
          <ActionIcon
            variant="subtle"
            aria-label={`Редактировать: ${label}`}
            title="Редактировать"
            onClick={() => { setDraft(value); setEditing(true) }}
          >✎</ActionIcon>
        </>
      )}
    </Group>
  )
}

export function LinksEditor({ showId, dubbing }: { showId: number; dubbing?: string }) {
  const queryClient = useQueryClient()
  const listQuery = useListLinks(showId)

  const saveLink = async (kind: AddLinkRequestKind | undefined, link: Link | undefined, value: string) => {
    if (link) {
      if (value) await updateLink(showId, link.id, { url: value })
      else await deleteLink(showId, link.id)
    } else if (value && kind) {
      await addLink(showId, { kind, url: value })
    }
    await queryClient.invalidateQueries({ queryKey: getListLinksQueryKey(showId) })
  }

  const saveDubbing = async (value: string) => {
    await updateShow(showId, { preferred_dubbing: value })
    await queryClient.invalidateQueries({ queryKey: getGetTrackedQueryKey(showId) })
  }

  return (
    <Stack gap="sm">
      {listQuery.isLoading ? <Text size="sm" c="dimmed">Загрузка ссылок…</Text> : listQuery.isError ? (
        <Group>
          <Text size="sm" c="red">Не удалось загрузить ссылки</Text>
          <Button size="xs" variant="subtle" onClick={() => listQuery.refetch()}>Повторить</Button>
        </Group>
      ) : kinds.flatMap((kind) => {
        const links = listQuery.data?.links.filter((link) => link.kind === kind.value) ?? []
        if (links.length === 0) return [
          <EditableRow key={kind.value} label={kind.label} value="" isLink onSave={(value) => saveLink(kind.value, undefined, value)} />,
        ]
        return links.map((link) => (
          <EditableRow
            key={link.id}
            label={link.label ? `${kind.label} · ${link.label}` : kind.label}
            value={link.url}
            isLink
            onSave={(value) => saveLink(kind.value, link, value)}
          />
        ))
      })}
      {!listQuery.isError && listQuery.data?.links
        .filter((link) => !kinds.some((kind) => kind.value === link.kind))
        .map((link) => (
          <EditableRow
            key={link.id}
            label={link.label || link.kind}
            value={link.url}
            isLink
            onSave={(value) => saveLink(undefined, link, value)}
          />
        ))}
      <EditableRow label="Озвучка" value={dubbing ?? ''} onSave={saveDubbing} />
    </Stack>
  )
}
