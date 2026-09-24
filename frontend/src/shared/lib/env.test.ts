import { afterEach, describe, expect, it } from 'vitest'
import { getApiBaseUrl, getAppStoreUrl, getMetrikaId } from './env'

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

describe('getAppStoreUrl', () => {
  afterEach(() => {
    delete window.__ENV
  })

  it('uses window.__ENV when set to a real value', () => {
    window.__ENV = { APP_STORE_URL: 'https://apps.apple.com/app/id6814790859' }
    expect(getAppStoreUrl()).toBe('https://apps.apple.com/app/id6814790859')
  })

  it('ignores an unsubstituted template placeholder', () => {
    window.__ENV = { APP_STORE_URL: '${APP_STORE_URL}' }
    expect(getAppStoreUrl()).toBeUndefined()
  })

  it('is undefined when unset, so callers can hide the link', () => {
    expect(getAppStoreUrl()).toBeUndefined()
  })
})

describe('getMetrikaId', () => {
  afterEach(() => {
    delete window.__ENV
  })

  it('parses a configured counter id', () => {
    window.__ENV = { METRIKA_ID: '112993969' }
    expect(getMetrikaId()).toBe(112993969)
  })

  it('is undefined for a placeholder, empty or malformed value', () => {
    for (const METRIKA_ID of ['${METRIKA_ID}', '', 'abc', '-1']) {
      window.__ENV = { METRIKA_ID }
      expect(getMetrikaId()).toBeUndefined()
    }
  })
})
