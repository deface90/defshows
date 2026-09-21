import { Anchor, Paper, Stack, Text, Title } from '@mantine/core'
import { Link } from 'react-router-dom'
import { legalOperator } from '@/shared/legal/operator'
import { useDocumentTitle } from '@/shared/lib/useDocumentTitle'
import { NotFoundPage } from './NotFoundPage'

export function LegalPage() {
  useDocumentTitle('Реквизиты')
  if (!legalOperator) return <NotFoundPage />

  return (
    <Stack maw={800} mx="auto" gap="lg">
      <Title order={1}>Юридическая информация</Title>
      <Text>defShows — сервис для учёта просмотра сериалов на shows.deface.dev.</Text>
      <Paper withBorder p="lg">
        <Stack gap="sm">
          <Title order={2} size="h3">Владелец сервиса</Title>
          <Text>{legalOperator.kind}: {legalOperator.name}</Text>
          <Text>ИНН: {legalOperator.inn}</Text>
          <Text>Обращения по работе сервиса и персональным данным:{' '}
            <Anchor href={`mailto:${legalOperator.email}`}>{legalOperator.email}</Anchor>
          </Text>
        </Stack>
      </Paper>
      <Anchor component={Link} to="/privacy">Политика обработки персональных данных</Anchor>
    </Stack>
  )
}
