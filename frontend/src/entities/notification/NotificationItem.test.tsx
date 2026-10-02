import { screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import type { NotificationItem as NotificationItemModel } from '@/shared/api/notifications/model'
import { renderWithProviders } from '@/test/render'
import { NotificationItem } from './NotificationItem'

const base: NotificationItemModel = {
  id: 1,
  type: 'episode_released',
  status: 'sent',
  read: false,
  payload: 'Вышел новый эпизод',
  created_at: '2026-05-01T10:00:00Z',
}

describe('NotificationItem', () => {
  it('labels a follow request and links to the actor profile', () => {
    renderWithProviders(
      <NotificationItem
        item={{ ...base, type: 'follow_request', payload: 'Аня хочет на вас подписаться', actor_id: 42 }}
      />,
    )
    expect(screen.getByText('Запрос на подписку')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Аня хочет на вас подписаться' })).toHaveAttribute(
      'href',
      '/users/42',
    )
  })

  it('labels a follow acceptance', () => {
    renderWithProviders(
      <NotificationItem item={{ ...base, type: 'follow_accepted', payload: 'Петя принял вашу заявку', actor_id: 7 }} />,
    )
    expect(screen.getByText('Новый подписчик')).toBeInTheDocument()
  })

  it('renders a non-follow notification as plain text', () => {
    renderWithProviders(<NotificationItem item={base} />)
    expect(screen.getByText('Новый эпизод')).toBeInTheDocument()
    expect(screen.getByText('Вышел новый эпизод')).toBeInTheDocument()
    expect(screen.queryByRole('link')).not.toBeInTheDocument()
  })
})
