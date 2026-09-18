import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { SearchPage } from './SearchPage'

const base = 'http://localhost:8080'

describe('SearchPage', () => {
  it('searches and renders results with an add button', async () => {
    server.use(
      http.get(`${base}/shows/search`, () =>
        HttpResponse.json({
          results: [{ tmdb_id: 1399, title: 'Game of Thrones', first_air_date: '2011-04-17' }],
        }),
      ),
    )
    renderWithProviders(<SearchPage />)

    await userEvent.type(screen.getByLabelText('Поиск сериалов'), 'thrones')

    expect(await screen.findByText('Game of Thrones', {}, { timeout: 3000 })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Добавить' })).toBeInTheDocument()
  })
})
