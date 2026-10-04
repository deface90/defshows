import { MantineProvider } from '@mantine/core'
import { Notifications } from '@mantine/notifications'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Button } from '@mantine/core'
import { useState } from 'react'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'
import { theme } from '@/app/theme'
import { getGetUserProfileQueryKey, useGetUserProfile } from '@/shared/api/users/endpoints'
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

  it('removes the now-inaccessible profile from the cache on block', async () => {
    server.use(
      http.post(`${base}/me/blocks/11`, () => new HttpResponse(null, { status: 204 })),
    )
    // A stale profile sits in the cache before the block.
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    queryClient.setQueryData(getGetUserProfileQueryKey(11), {
      id: 11,
      display_name: 'Stale User',
      is_public: true,
    })
    render(
      <MantineProvider env="test" theme={theme} defaultColorScheme="dark">
        <Notifications />
        <QueryClientProvider client={queryClient}>
          <BlockHarness userId={11} />
        </QueryClientProvider>
      </MantineProvider>,
    )
    await userEvent.click(screen.getByRole('button', { name: 'Заблокировать' }))
    await waitFor(() =>
      expect(queryClient.getQueryData(getGetUserProfileQueryKey(11))).toBeUndefined(),
    )
  })
})

/**
 * Harness mimicking UserProfilePage: a mounted profile observer renders the profile name,
 * and a block button whose onBlocked callback hides the profile (navigate-away equivalent).
 * Proves that after blocking the stale profile is no longer shown to an already-mounted
 * observer — cache removal alone does not unmount observers.
 */
function ProfileHarness({ userId }: { userId: number }) {
  const [blocked, setBlocked] = useState(false)
  const { block, busy } = useBlockMutations(userId, () => setBlocked(true))
  const profileQuery = useGetUserProfile(userId, { query: { retry: false } })
  if (blocked) return <div>Профиль недоступен</div>
  return (
    <div>
      {profileQuery.data && <div>{profileQuery.data.display_name}</div>}
      <Button loading={busy} onClick={() => block.mutate()}>
        Заблокировать
      </Button>
    </div>
  )
}

describe('block from an open profile', () => {
  it('stops rendering the stale profile after block', async () => {
    server.use(
      http.get(`${base}/users/13`, () =>
        HttpResponse.json({
          id: 13,
          display_name: 'Victim User',
          is_public: true,
          shows_count: 0,
          followers_count: 0,
          following_count: 0,
          is_following: 'none',
        }),
      ),
      http.post(`${base}/me/blocks/13`, () => new HttpResponse(null, { status: 204 })),
    )
    renderWithProviders(<ProfileHarness userId={13} />)
    // Profile is visible first.
    expect(await screen.findByText('Victim User')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Заблокировать' }))
    // After blocking, the stale profile is gone and the unavailable state shows.
    await waitFor(() => expect(screen.getByText('Профиль недоступен')).toBeInTheDocument())
    expect(screen.queryByText('Victim User')).not.toBeInTheDocument()
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
