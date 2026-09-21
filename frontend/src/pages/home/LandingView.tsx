import { Button, Card, Container, Group, SimpleGrid, Stack, Text, Title } from '@mantine/core'
import { Link } from 'react-router-dom'

const features = [
  {
    title: 'Каталог без границ',
    body: 'Листайте популярное, ищите сериалы и открывайте страницы с сезонами, эпизодами и рейтингами — без регистрации.',
  },
  {
    title: 'Ваш прогресс',
    body: 'Отмечайте просмотренные серии, ведите список «на просмотре» и не теряйте, на чём остановились.',
  },
  {
    title: 'Умные подборки',
    body: 'Рекомендации по вашим жанрам и языкам — на основе того, что вы уже смотрите.',
  },
]

/** LandingView is the marketing home shown to guests at `/`. */
export function LandingView() {
  return (
    <Container size="md" py="xl">
      <Stack align="center" gap="lg" ta="center">
        <Title order={1} fz={{ base: 40, sm: 56 }} lh={1.05}>
          Смотрите. Отмечайте.{' '}
          <Text span inherit c="orange.5">
            Не теряйтесь.
          </Text>
        </Title>
        <Text c="dimmed" fz="lg" maw={560}>
          defShows — трекер сериалов, где можно свободно листать каталог, а аккаунт заводить
          только когда захотите сохранить прогресс.
        </Text>
        <Group>
          <Button component={Link} to="/register" size="md" color="orange">
            Начать
          </Button>
          <Button component={Link} to="/discover" size="md" variant="default">
            Смотреть каталог
          </Button>
        </Group>
      </Stack>

      <SimpleGrid cols={{ base: 1, sm: 3 }} spacing="lg" mt={64}>
        {features.map((f) => (
          <Card key={f.title} withBorder padding="lg" radius="md">
            <Text fw={600} mb={6}>
              {f.title}
            </Text>
            <Text c="dimmed" fz="sm">
              {f.body}
            </Text>
          </Card>
        ))}
      </SimpleGrid>
    </Container>
  )
}
