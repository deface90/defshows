import { ActionIcon, Badge, Button, Card, Group, Stack, Text, Textarea } from '@mantine/core'
import { notifications } from '@mantine/notifications'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import {
  deleteNote,
  getListNotesQueryKey,
  updateNote,
  useListNotes,
} from '@/shared/api/notes/endpoints'
import type { Note } from '@/shared/api/notes/model'
import { EmptyState, ErrorState, LoadingState } from '@/shared/ui/states'
import { NoteEditor } from './NoteEditor'

function scopeLabel(note: Note): string {
  if (note.scope === 'season') return `Сезон ${note.season_number}`
  if (note.scope === 'episode') return `S${note.season_number}E${note.episode_number}`
  return 'Сериал'
}

function NoteCard({ note, showId }: { note: Note; showId: number }) {
  const queryClient = useQueryClient()
  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: getListNotesQueryKey({ show_id: showId }) })
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(note.body)

  const save = useMutation({
    mutationFn: () => updateNote(note.id, { body: draft.trim() }),
    onSuccess: () => {
      setEditing(false)
      invalidate()
    },
    onError: () => notifications.show({ message: 'Не удалось сохранить', color: 'red' }),
  })

  const remove = useMutation({
    mutationFn: () => deleteNote(note.id),
    onSuccess: invalidate,
    onError: () => notifications.show({ message: 'Не удалось удалить', color: 'red' }),
  })

  return (
    <Card withBorder padding="sm">
      <Group justify="space-between" mb="xs">
        <Badge variant="light">{scopeLabel(note)}</Badge>
        <Group gap={4}>
          {!editing && (
            <ActionIcon variant="subtle" onClick={() => setEditing(true)} aria-label="Редактировать">
              ✎
            </ActionIcon>
          )}
          <ActionIcon
            variant="subtle"
            color="red"
            loading={remove.isPending}
            onClick={() => remove.mutate()}
            aria-label="Удалить заметку"
          >
            ✕
          </ActionIcon>
        </Group>
      </Group>
      {editing ? (
        <Stack gap="xs">
          <Textarea
            rows={3}
            aria-label="Текст заметки"
            value={draft}
            onChange={(e) => setDraft(e.currentTarget.value)}
          />
          <Group justify="flex-end" gap="xs">
            <Button
              variant="default"
              size="xs"
              onClick={() => {
                setDraft(note.body)
                setEditing(false)
              }}
            >
              Отмена
            </Button>
            <Button size="xs" loading={save.isPending} disabled={!draft.trim()} onClick={() => save.mutate()}>
              Сохранить
            </Button>
          </Group>
        </Stack>
      ) : (
        <Text size="sm" style={{ whiteSpace: 'pre-wrap' }}>
          {note.body}
        </Text>
      )}
    </Card>
  )
}

/** NotesList shows the user's private notes for a show plus a create form. */
export function NotesList({ showId }: { showId: number }) {
  const query = useListNotes({ show_id: showId }, { query: { enabled: Number.isFinite(showId) } })

  return (
    <Stack>
      <NoteEditor showId={showId} />
      {query.isLoading ? (
        <LoadingState />
      ) : query.isError ? (
        <ErrorState message="Не удалось загрузить заметки" />
      ) : !query.data?.notes.length ? (
        <EmptyState title="Заметок пока нет" description="Добавьте первую приватную заметку выше." />
      ) : (
        query.data.notes.map((note) => <NoteCard key={note.id} note={note} showId={showId} />)
      )}
    </Stack>
  )
}
