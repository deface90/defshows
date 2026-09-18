import { describe, expect, it } from 'vitest'
import { noteSchema } from './schemas'

const base = { scope: 'show' as const, season_number: null, episode_number: null, body: 'hi' }

describe('noteSchema', () => {
  it('accepts a show-scoped note', () => {
    expect(noteSchema.safeParse(base).success).toBe(true)
  })

  it('rejects an empty body', () => {
    const res = noteSchema.safeParse({ ...base, body: '   ' })
    expect(res.success).toBe(false)
  })

  it('requires a season number for season scope', () => {
    const res = noteSchema.safeParse({ ...base, scope: 'season' })
    expect(res.success).toBe(false)
    if (!res.success) {
      expect(res.error.issues.some((i) => i.path[0] === 'season_number')).toBe(true)
    }
  })

  it('requires an episode number for episode scope', () => {
    const res = noteSchema.safeParse({ ...base, scope: 'episode', season_number: 1 })
    expect(res.success).toBe(false)
    if (!res.success) {
      expect(res.error.issues.some((i) => i.path[0] === 'episode_number')).toBe(true)
    }
  })

  it('accepts a fully specified episode note', () => {
    const res = noteSchema.safeParse({ scope: 'episode', season_number: 1, episode_number: 3, body: 'x' })
    expect(res.success).toBe(true)
  })
})
