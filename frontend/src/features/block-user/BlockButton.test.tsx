import { Button } from '@mantine/core'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { renderWithProviders } from '@/test/render'
import { BlockButton, useBlockMutations } from './BlockButton'

const base = 'http://localhost:8080'

/** Test harness that exposes the block mutation from the hook. */
function BlockHarness({ userId }: { userId: number }) {
  const { block, busy } = useBlockMutations(userId)
  return (
    <Button loading={busy} onClick={() => block.mutate()}>
      Заблокировать
    </Button>
  )
}

describe('useBlockMutations', () => {
  it('blocks via POST /me/blocks/{id}', async () => {
    let method: string | undefined
    server.use(
      http.post(`${base}/me/blocks/7`, ({ request }) => {
        method = request.method
        return new HttpResponse(null, { status: 204 })
      }),
    )
    renderWithProviders(<BlockHarness userId={7} />)
    await userEvent.click(screen.getByRole('button', { name: 'Заблокировать' }))
    await waitFor(() => expect(method).toBe('POST'))
  })
})

describe('BlockButton', () => {
  it('unblocks via DELETE /me/blocks/{id}', async () => {
    let method: string | undefined
    server.use(
      http.delete(`${base}/me/blocks/9`, ({ request }) => {
        method = request.method
        return new HttpResponse(null, { status: 204 })
      }),
    )
    renderWithProviders(<BlockButton userId={9} />)
    await userEvent.click(screen.getByRole('button', { name: 'Разблокировать' }))
    await waitFor(() => expect(method).toBe('DELETE'))
  })
})
