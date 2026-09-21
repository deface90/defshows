import { Badge, Box, Button, Card, Group, Pagination, Stack, Text, TextInput, Title } from '@mantine/core'
import { Link, useLocation, useSearchParams } from 'react-router-dom'
import { useEffect, useState } from 'react'
import { useListUsers } from '@/shared/api/users/endpoints'
import { useAuthStore } from '@/shared/auth/authStore'
import { useDocumentTitle } from '@/shared/lib/useDocumentTitle'
import { EmptyState, ErrorState, LoadingState } from '@/shared/ui/states'

/** UsersPage lists all users with a link into each public/private profile. */
export function UsersPage() {
  useDocumentTitle('Пользователи')
  const [params, setParams] = useSearchParams()
  const location = useLocation()
  const search = params.get('q')?.trim() ?? ''
  const rawPage = Number(params.get('page') ?? 1)
  const page = Number.isSafeInteger(rawPage) && rawPage >= 1 && rawPage <= 1_000_000 ? rawPage : 1
  const [draft, setDraft] = useState(search)
  useEffect(() => setDraft(search), [search])
  const pageSize = 20
  const query = useListUsers({ q: search || undefined, page, page_size: pageSize })
  const total = query.data?.total ?? 0
  const totalPages = Math.ceil(total / pageSize)
  const changePage = (next: number) => {
    const updated = new URLSearchParams(params)
    if (next === 1) updated.delete('page')
    else updated.set('page', String(next))
    setParams(updated)
  }
  const me = useAuthStore((s) => s.user)
  const users = query.data?.users ?? []

  return (
    <Stack gap="lg">
      <div>
        <Title order={2}>Пользователи</Title>
        <Text c="dimmed" mt={4}>
          Загляните в чужие коллекции сериалов
        </Text>
      </div>

      <Group component="form" align="flex-end" onSubmit={(event) => {
        event.preventDefault()
        const updated = new URLSearchParams(params)
        const value = draft.trim()
        if (value) updated.set('q', value)
        else updated.delete('q')
        updated.delete('page')
        setParams(updated)
      }}>
        <TextInput
          label="Поиск пользователей"
          placeholder="Имя пользователя…"
          value={draft}
          onChange={(event) => setDraft(event.currentTarget.value)}
          maxLength={100}
          style={{ flex: 1, minWidth: 180 }}
        />
        <Button type="submit">Найти</Button>
        {search && <Button variant="subtle" onClick={() => {
          setDraft('')
          const updated = new URLSearchParams(params)
          updated.delete('q')
          updated.delete('page')
          setParams(updated)
        }}>Сбросить</Button>}
      </Group>

      {query.isSuccess && <Text size="sm" c="dimmed">Найдено пользователей: {total}</Text>}
      <Card withBorder padding={0}>
        {query.isLoading && <LoadingState />}
        {query.isError && <Stack p="md">
          <ErrorState message="Не удалось загрузить список пользователей" />
          <Button variant="light" onClick={() => query.refetch()}>Повторить</Button>
        </Stack>}
        {query.isSuccess && users.length === 0 && (
          <Stack p="md">
            <EmptyState title={page > 1 ? 'На этой странице никого нет' : search ? 'Пользователи не найдены' : 'Пока никого нет'} />
            {page > 1 && <Button variant="light" onClick={() => changePage(1)}>На первую страницу</Button>}
          </Stack>
        )}
        {users.map((u, i) => (
          <Box
            key={u.id}
            component={Link}
            to={`/users/${u.id}`}
            state={{ directoryFrom: `${location.pathname}${location.search}` }}
            style={{
              display: 'block',
              color: 'inherit',
              textDecoration: 'none',
              borderTop: i > 0 ? '1px solid var(--mantine-color-default-border)' : undefined,
            }}
          >
            <Group wrap="nowrap" justify="space-between" p="md">
              <Stack gap={2} style={{ minWidth: 0 }}>
                <Group gap={8} wrap="nowrap">
                  <Text fw={600} truncate>
                    {u.display_name}
                  </Text>
                  {me?.id === u.id && (
                    <Text span c="dimmed" size="sm">
                      (вы)
                    </Text>
                  )}
                </Group>
                <Text c="dimmed" size="sm">
                  {u.shows_count} сериалов
                </Text>
              </Stack>
              <Badge variant="light" color={u.is_public ? 'brand' : 'gray'} style={{ flexShrink: 0 }}>
                {u.is_public ? 'Публичный' : 'Приватный'}
              </Badge>
            </Group>
          </Box>
        ))}
      </Card>
      {query.isSuccess && totalPages > 1 && page <= totalPages && (
        <Pagination
          total={totalPages}
          value={page}
          onChange={changePage}
          getItemProps={(number) => ({ 'aria-label': `Страница ${number}` })}
        />
      )}
    </Stack>
  )
}
