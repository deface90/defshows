import { http, HttpResponse } from 'msw'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { server } from '@/shared/testing/mswServer'
import { customInstance, setAuthHooks } from './http'

const base = 'http://localhost:8080'

describe('http customInstance', () => {
  afterEach(() => setAuthHooks(null))

  it('attaches the bearer access token', async () => {
    setAuthHooks({ getAccessToken: () => 'tok', refresh: async () => false, onLogout: () => {} })
    let seenAuth = ''
    server.use(
      http.get(`${base}/ping`, ({ request }) => {
        seenAuth = request.headers.get('Authorization') ?? ''
        return HttpResponse.json({ ok: true })
      }),
    )

    const data = await customInstance<{ ok: boolean }>({ url: '/ping', method: 'GET' })
    expect(data.ok).toBe(true)
    expect(seenAuth).toBe('Bearer tok')
  })

  it('refreshes once on 401 then retries', async () => {
    let calls = 0
    const refresh = vi.fn(async () => true)
    setAuthHooks({ getAccessToken: () => 'tok', refresh, onLogout: () => {} })
    server.use(
      http.get(`${base}/secure`, () => {
        calls += 1
        if (calls === 1) return new HttpResponse(null, { status: 401 })
        return HttpResponse.json({ ok: true })
      }),
    )

    const data = await customInstance<{ ok: boolean }>({ url: '/secure', method: 'GET' })
    expect(data.ok).toBe(true)
    expect(refresh).toHaveBeenCalledTimes(1)
    expect(calls).toBe(2)
  })

  it('logs out when refresh fails', async () => {
    const onLogout = vi.fn()
    setAuthHooks({ getAccessToken: () => null, refresh: async () => false, onLogout })
    server.use(http.get(`${base}/secure`, () => new HttpResponse(null, { status: 401 })))

    await expect(customInstance({ url: '/secure', method: 'GET' })).rejects.toBeDefined()
    expect(onLogout).toHaveBeenCalledTimes(1)
  })
})
