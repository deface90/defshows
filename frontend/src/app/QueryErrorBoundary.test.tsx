import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { renderWithProviders } from '@/test/render'
import { QueryErrorBoundary } from './QueryErrorBoundary'

function Boom({ crash }: { crash: boolean }) {
  if (crash) throw new Error('kaboom')
  return <div>всё хорошо</div>
}

function Harness() {
  const [crash, setCrash] = useState(true)
  return (
    <div>
      <button onClick={() => setCrash(false)}>fix</button>
      <QueryErrorBoundary>
        <Boom crash={crash} />
      </QueryErrorBoundary>
    </div>
  )
}

describe('QueryErrorBoundary', () => {
  beforeEach(() => vi.spyOn(console, 'error').mockImplementation(() => {}))
  afterEach(() => vi.restoreAllMocks())

  it('catches render errors and recovers on retry', async () => {
    renderWithProviders(<Harness />)

    expect(screen.getByText('Что-то пошло не так')).toBeInTheDocument()

    // Fix the underlying condition, then retry to re-render children.
    await userEvent.click(screen.getByRole('button', { name: 'fix' }))
    await userEvent.click(screen.getByRole('button', { name: 'Повторить' }))

    expect(await screen.findByText('всё хорошо')).toBeInTheDocument()
  })
})
