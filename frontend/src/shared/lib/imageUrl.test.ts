import { afterEach, describe, expect, it } from 'vitest'
import { imageUrl } from './imageUrl'

const originalEnv = window.__ENV
afterEach(() => { window.__ENV = originalEnv })

describe('imageUrl', () => {
  it('uses the configured API origin and preserves the size and filename', () => {
    window.__ENV = { API_BASE_URL: 'https://api.example.com/api/' }
    expect(imageUrl('https://image.tmdb.org/t/p/w500/abc.jpg')).toBe('https://api.example.com/api/images/tmdb/w500/abc.jpg')
    expect(imageUrl('http://image.tmdb.org/t/p/original/abc.png')).toBe('https://api.example.com/api/images/tmdb/original/abc.png')
  })

  it('supports a relative API base', () => {
    window.__ENV = { API_BASE_URL: '/api' }
    expect(imageUrl('https://image.tmdb.org/t/p/w500/abc.jpg')).toBe('/api/images/tmdb/w500/abc.jpg')
  })

  it.each([
    'https://api.example.com/images/shows/42/poster.jpg',
    '/images/shows/42/poster.jpg',
    'https://image.tmdb.org.evil.example/t/p/w500/abc.jpg',
  ])('preserves non-TMDB URLs: %s', (src) => {
    expect(imageUrl(src)).toBe(src)
  })

  it('keeps missing posters absent', () => {
    expect(imageUrl(null)).toBeUndefined()
    expect(imageUrl('')).toBeUndefined()
    expect(imageUrl()).toBeUndefined()
  })
})
