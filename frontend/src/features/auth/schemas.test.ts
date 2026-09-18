import { describe, expect, it } from 'vitest'
import { loginSchema, registerSchema } from './schemas'

describe('auth schemas', () => {
  it('accepts a valid login', () => {
    expect(loginSchema.safeParse({ email: 'user@example.com', password: 'x' }).success).toBe(true)
  })

  it('rejects a bad email', () => {
    expect(loginSchema.safeParse({ email: 'nope', password: 'x' }).success).toBe(false)
  })

  it('accepts matching register passwords', () => {
    const r = registerSchema.safeParse({ email: 'user@example.com', password: 'pw12345', confirm: 'pw12345' })
    expect(r.success).toBe(true)
  })

  it('rejects short password', () => {
    const r = registerSchema.safeParse({ email: 'user@example.com', password: '123', confirm: '123' })
    expect(r.success).toBe(false)
  })

  it('rejects mismatched confirm', () => {
    const r = registerSchema.safeParse({ email: 'user@example.com', password: 'pw12345', confirm: 'nope123' })
    expect(r.success).toBe(false)
    if (!r.success) {
      expect(r.error.issues.some((i) => i.path.includes('confirm'))).toBe(true)
    }
  })
})
