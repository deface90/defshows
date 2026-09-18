import { afterEach, describe, expect, it } from 'vitest'
import { getApiBaseUrl } from './env'

describe('getApiBaseUrl', () => {
  afterEach(() => {
    delete window.__ENV
  })

  it('uses window.__ENV when set to a real value', () => {
    window.__ENV = { API_BASE_URL: 'https://api.example.com' }
    expect(getApiBaseUrl()).toBe('https://api.example.com')
  })

  it('ignores an unsubstituted template placeholder', () => {
    window.__ENV = { API_BASE_URL: '${API_BASE_URL}' }
    expect(getApiBaseUrl()).toBe('http://localhost:8080')
  })

  it('falls back to the default when unset', () => {
    expect(getApiBaseUrl()).toBe('http://localhost:8080')
  })
})
