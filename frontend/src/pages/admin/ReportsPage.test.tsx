import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import type { Report } from '@/shared/api/admin/model'
import { ReportsPage } from './ReportsPage'

const base = 'http://localhost:8080'

const openReport: Report = {
  id: 1,
  reporter: { id: 10, display_name: 'Алиса' },
  target: { id: 20, display_name: 'Боб' },
  reason: 'harassment',
  note: 'груб',
  status: 'open',
  created_at: '2026-10-01T10:00:00Z',
}

describe('ReportsPage', () => {
  it('lists open reports with reporter/target and reason', async () => {
    server.use(http.get(`${base}/admin/reports`, () => HttpResponse.json({ total: 1, reports: [openReport] })))
    renderWithProviders(<ReportsPage />)

    expect(await screen.findByText('Алиса')).toBeInTheDocument()
    expect(screen.getByText('Боб')).toBeInTheDocument()
    expect(screen.getByText('Оскорбления')).toBeInTheDocument()
    expect(screen.getByText('груб')).toBeInTheDocument()
  })

  it('filters by status via the segmented control', async () => {
    const statuses: (string | null)[] = []
    server.use(
      http.get(`${base}/admin/reports`, ({ request }) => {
        statuses.push(new URL(request.url).searchParams.get('status'))
        return HttpResponse.json({ total: 0, reports: [] })
      }),
    )
    renderWithProviders(<ReportsPage />)

    // Default filter is "open".
    await waitFor(() => expect(statuses).toContain('open'))

    await userEvent.click(screen.getByRole('radio', { name: 'Все' }))
    await waitFor(() => expect(statuses).toContain(null))
  })

  it('resolves an open report and refetches', async () => {
    let posted: unknown
    let listCalls = 0
    const reports = [openReport]
    server.use(
      http.get(`${base}/admin/reports`, () => {
        listCalls += 1
        return HttpResponse.json({ total: reports.length, reports })
      }),
      http.post(`${base}/admin/reports/1/resolve`, async ({ request }) => {
        posted = await request.json()
        reports.length = 0
        return new HttpResponse(null, { status: 204 })
      }),
    )
    renderWithProviders(<ReportsPage />)

    const row = (await screen.findByText('Алиса')).closest('tr') as HTMLElement
    await userEvent.click(within(row).getByRole('button', { name: 'Решить' }))

    await waitFor(() => expect(posted).toEqual({ status: 'resolved' }))
    expect(await screen.findByText('Жалоб нет')).toBeInTheDocument()
    expect(listCalls).toBeGreaterThan(1)
  })

  it('dismisses an open report with the dismissed status', async () => {
    let posted: unknown
    server.use(
      http.get(`${base}/admin/reports`, () => HttpResponse.json({ total: 1, reports: [openReport] })),
      http.post(`${base}/admin/reports/1/resolve`, async ({ request }) => {
        posted = await request.json()
        return new HttpResponse(null, { status: 204 })
      }),
    )
    renderWithProviders(<ReportsPage />)

    const row = (await screen.findByText('Алиса')).closest('tr') as HTMLElement
    await userEvent.click(within(row).getByRole('button', { name: 'Отклонить' }))

    await waitFor(() => expect(posted).toEqual({ status: 'dismissed' }))
  })
})
