import axios, { type AxiosError, type AxiosRequestConfig } from 'axios'
import { getApiBaseUrl } from '@/shared/lib/env'

// AuthHooks decouple the HTTP layer from the auth store: the store registers
// these once at app-init (see shared/auth/session), so http.ts never imports the
// store (avoids a circular dependency).
export interface AuthHooks {
  getAccessToken: () => string | null
  refresh: () => Promise<boolean>
  onLogout: () => void
}

let hooks: AuthHooks | null = null

/** setAuthHooks wires the auth store into the HTTP interceptors. */
export function setAuthHooks(h: AuthHooks | null) {
  hooks = h
}

export const axiosInstance = axios.create({ baseURL: getApiBaseUrl() })

axiosInstance.interceptors.request.use((config) => {
  const token = hooks?.getAccessToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Single-flight refresh: concurrent 401s await one refresh call, so the backend's
// single-use rotating refresh token isn't reused (which would revoke the family).
let refreshPromise: Promise<boolean> | null = null

function refreshOnce(): Promise<boolean> {
  if (!hooks) return Promise.resolve(false)
  if (!refreshPromise) {
    refreshPromise = hooks.refresh().finally(() => {
      refreshPromise = null
    })
  }
  return refreshPromise
}

axiosInstance.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const original = error.config as (AxiosRequestConfig & { _retry?: boolean }) | undefined
    if (error.response?.status === 401 && original && !original._retry && hooks) {
      original._retry = true
      const ok = await refreshOnce()
      if (ok) {
        return axiosInstance(original)
      }
      hooks.onLogout()
    }
    return Promise.reject(error)
  },
)

/** customInstance is the orval mutator: unwraps the axios response to T. */
export const customInstance = <T>(config: AxiosRequestConfig, options?: AxiosRequestConfig): Promise<T> => {
  return axiosInstance({ ...config, ...options }).then((r) => r.data as T)
}

export default customInstance
