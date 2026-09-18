import { ActionIcon, Anchor, Button, Group, Select, Stack, Text, TextInput } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import {
  addLink,
  deleteLink,
  getGetTrackedQueryKey,
  getListLinksQueryKey,
  updateShow,
  useListLinks,
} from '@/shared/api/tracking/endpoints'
import type { AddLinkRequestKind } from '@/shared/api/tracking/model'

const kindOptions = [
  { value: 'download', label: 'Скачать' },
  { value: 'streaming', label: 'Смотреть' },
  { value: 'wiki', label: 'Wiki' },
  { value: 'imdb', label: 'IMDB' },
  { value: 'kinopoisk', label: 'Kinopoisk' },
]

function kindLabel(kind: string): string {
  return kindOptions.find((o) => o.value === kind)?.label ?? kind
}

/**
 * LinksEditor shows the show's "where to watch / download / reference" links and
 * the preferred dubbing. It mirrors the prototype: a read-only view with an
 * "Edit" button that reveals the add/remove form and dubbing input.
 */
export function LinksEditor({ showId, dubbing }: { showId: number; dubbing?: string }) {
  const queryClient = useQueryClient()
  const listQuery = useListLinks(showId)
  const links = listQuery.data?.links ?? []
  const invalidate = () => queryClient.invalidateQueries({ queryKey: getListLinksQueryKey(showId) })

  const [editing, setEditing] = useState(false)
  const [kind, setKind] = useState<string>('download')
  const [label, setLabel] = useState('')
  const [url, setUrl] = useState('')
  const [dub, setDub] = useState(dubbing ?? '')

  const add = useMutation({
    mutationFn: () => addLink(showId, { kind: kind as AddLinkRequestKind, label, url }),
    onSuccess: () => {
      setLabel('')
      setUrl('')
      invalidate()
    },
    onError: () => notifications.show({ message: 'Не удалось добавить ссылку', color: 'red' }),
  })

  const remove = useMutation({
    mutationFn: (linkId: number) => deleteLink(showId, linkId),
    onSuccess: invalidate,
  })

  const saveDubbing = useMutation({
    mutationFn: () => updateShow(showId, { preferred_dubbing: dub }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: getGetTrackedQueryKey(showId) })
      notifications.show({ message: 'Озвучка сохранена', color: 'teal' })
    },
    onError: () => notifications.show({ message: 'Не удалось сохранить', color: 'red' }),
  })

  if (!editing) {
    return (
      <Stack gap="sm">
        <Group justify="space-between">
          <Text fw={600}>Ссылки</Text>
          <Button variant="default" size="xs" onClick={() => setEditing(true)}>
            Редактировать
          </Button>
        </Group>
        {links.length === 0 && !dubbing ? (
          <Text c="dimmed" size="sm">
            Ссылки не добавлены
          </Text>
        ) : (
          <>
            {links.length > 0 && (
              <Group gap="xs">
                {links.map((link) => (
                  <Anchor
                    key={link.id}
                    href={link.url}
                    target="_blank"
                    rel="noreferrer"
                    size="sm"
                    style={{
                      padding: '6px 11px',
                      borderRadius: 9,
                      background: 'var(--mantine-color-default-hover)',
                    }}
                  >
                    {link.label || kindLabel(link.kind)} ↗
                  </Anchor>
                ))}
              </Group>
            )}
            {dubbing && (
              <Text size="sm" c="dimmed">
                Озвучка: {dubbing}
              </Text>
            )}
          </>
        )}
      </Stack>
    )
  }

  return (
    <Stack gap="sm">
      <Group justify="space-between">
        <Text fw={600}>Ссылки</Text>
        <Button size="xs" onClick={() => setEditing(false)}>
          Готово
        </Button>
      </Group>

      {links.map((link) => (
        <Group key={link.id} justify="space-between">
          <Anchor href={link.url} target="_blank" rel="noreferrer" size="sm">
            [{link.kind}] {link.label || link.url}
          </Anchor>
          <ActionIcon
            variant="subtle"
            color="red"
            onClick={() => remove.mutate(link.id)}
            aria-label="Удалить"
          >
            ✕
          </ActionIcon>
        </Group>
      ))}

      <Group align="flex-end" gap="xs">
        <Select
          w={130}
          data={kindOptions}
          value={kind}
          onChange={(v) => v && setKind(v)}
          aria-label="Тип ссылки"
        />
        <TextInput placeholder="Название" value={label} onChange={(e) => setLabel(e.currentTarget.value)} />
        <TextInput
          placeholder="URL"
          style={{ flex: 1 }}
          value={url}
          onChange={(e) => setUrl(e.currentTarget.value)}
        />
        <Button onClick={() => add.mutate()} loading={add.isPending} disabled={!url}>
          Добавить
        </Button>
      </Group>

      <Group align="flex-end" gap="xs">
        <TextInput
          label="Озвучка"
          placeholder="напр. LostFilm"
          style={{ flex: 1 }}
          value={dub}
          onChange={(e) => setDub(e.currentTarget.value)}
        />
        <Button variant="default" onClick={() => saveDubbing.mutate()} loading={saveDubbing.isPending}>
          Сохранить озвучку
        </Button>
      </Group>
    </Stack>
  )
}
