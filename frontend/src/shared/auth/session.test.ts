import { http, HttpResponse } from 'msw'
import { afterEach, describe, expect, it } from 'vitest'
import { customInstance, setAuthHooks } from '@/shared/api/http'
import { server } from '@/shared/testing/mswServer'
import { useAuthStore } from './authStore'
import { bootstrapSession, initAuth } from './session'

const base = 'http://localhost:8080'
const tokens = { access_token: 'new-access', refresh_token: 'new-refresh' }
const user = { id: 1, display_name: 'A', role: 'user', timezone: 'UTC' }

describe('session', () => {
  afterEach(() => {
    useAuthStore.getState().clear()
    localStorage.clear()
    setAuthHooks(null)
  })

  it('bootstrapSession restores a session from a stored refresh token', async () => {
    localStorage.setItem('defshows_refresh', 'stored')
    server.use(
      http.post(`${base}/auth/refresh`, () => HttpResponse.json(tokens)),
      http.get(`${base}/auth/me`, () => HttpResponse.json(user)),
    )
    initAuth()
    await bootstrapSession()
    expect(useAuthStore.getState().isAuthenticated).toBe(true)
    expect(useAuthStore.getState().accessToken).toBe('new-access')
  })

  it('bootstrapSession clears the session when refresh fails', async () => {
    localStorage.setItem('defshows_refresh', 'bad')
    server.use(http.post(`${base}/auth/refresh`, () => new HttpResponse(null, { status: 401 })))
    initAuth()
    await bootstrapSession()
    expect(useAuthStore.getState().isAuthenticated).toBe(false)
    expect(localStorage.getItem('defshows_refresh')).toBeNull()
  })

  it('two concurrent 401s trigger exactly one refresh', async () => {
    localStorage.setItem('defshows_refresh', 'stored')
    let refreshCount = 0
    let refreshed = false
    server.use(
      http.post(`${base}/auth/refresh`, () => {
        refreshCount += 1
        refreshed = true
        return HttpResponse.json(tokens)
      }),
      http.get(`${base}/auth/me`, () => HttpResponse.json(user)),
      http.get(`${base}/me/shows`, () =>
        refreshed ? HttpResponse.json({ ok: true }) : new HttpResponse(null, { status: 401 }),
      ),
    )
    initAuth()

    const [a, b] = await Promise.all([
      customInstance<{ ok: boolean }>({ url: '/me/shows', method: 'GET' }),
      customInstance<{ ok: boolean }>({ url: '/me/shows', method: 'GET' }),
    ])
    expect(a.ok).toBe(true)
    expect(b.ok).toBe(true)
    expect(refreshCount).toBe(1)
  })
})
