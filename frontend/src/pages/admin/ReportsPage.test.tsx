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
    expect(screen.getByText('Оскорбления или травля')).toBeInTheDocument()
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

  it('paginates to older reports via the page control', async () => {
    const requestedPages: string[] = []
    server.use(
      http.get(`${base}/admin/reports`, ({ request }) => {
        const page = new URL(request.url).searchParams.get('page') ?? '1'
        requestedPages.push(page)
        // 60 total → 2 pages of 50.
        if (page === '2') {
          return HttpResponse.json({
            total: 60,
            reports: [{ ...openReport, id: 2, reporter: { id: 11, display_name: 'Старый репортёр' } }],
          })
        }
        return HttpResponse.json({ total: 60, reports: [openReport] })
      }),
    )
    renderWithProviders(<ReportsPage />)

    expect(await screen.findByText('Алиса')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: '2' }))
    expect(await screen.findByText('Старый репортёр')).toBeInTheDocument()
    await waitFor(() => expect(requestedPages).toContain('2'))
  })

  it('clamps to the last page when resolving the sole report on the final page', async () => {
    // Page 2 of 2 (total 60) holds one report; resolving it drops total to 50 (1 page).
    // The view must clamp page 2 → page 1 and show the remaining reports, not "Жалоб нет".
    let resolved = false
    const page1 = Array.from({ length: 50 }, (_, i) => ({
      ...openReport,
      id: 100 + i,
      reporter: { id: 100 + i, display_name: `Репортёр ${i}` },
    }))
    const requestedPages: string[] = []
    server.use(
      http.get(`${base}/admin/reports`, ({ request }) => {
        const page = new URL(request.url).searchParams.get('page') ?? '1'
        requestedPages.push(page)
        if (resolved) {
          // After resolve: only the 50 page-1 reports remain (1 page total).
          return page === '1'
            ? HttpResponse.json({ total: 50, reports: page1 })
            : HttpResponse.json({ total: 50, reports: [] })
        }
        return page === '2'
          ? HttpResponse.json({ total: 60, reports: [openReport] })
          : HttpResponse.json({ total: 60, reports: page1 })
      }),
      http.post(`${base}/admin/reports/1/resolve`, async () => {
        resolved = true
        return new HttpResponse(null, { status: 204 })
      }),
    )
    renderWithProviders(<ReportsPage />)

    // Go to page 2 where the lone extra report lives.
    expect(await screen.findByText('Репортёр 0')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: '2' }))
    const row = (await screen.findByText('Алиса')).closest('tr') as HTMLElement
    await userEvent.click(within(row).getByRole('button', { name: 'Решить' }))

    // Must land back on page 1 content, NOT the empty state.
    expect(await screen.findByText('Репортёр 0')).toBeInTheDocument()
    expect(screen.queryByText('Жалоб нет')).not.toBeInTheDocument()
    await waitFor(() => expect(requestedPages).toContain('1'))
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
