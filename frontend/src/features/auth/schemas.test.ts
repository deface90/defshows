import { describe, expect, it } from 'vitest'
import {
  changePasswordSchema,
  forgotPasswordSchema,
  loginSchema,
  registerSchema,
  resetPasswordSchema,
} from './schemas'

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

  it('validates forgot-password email', () => {
    expect(forgotPasswordSchema.safeParse({ email: 'user@example.com' }).success).toBe(true)
    expect(forgotPasswordSchema.safeParse({ email: 'nope' }).success).toBe(false)
  })

  it('validates reset-password matching and length', () => {
    expect(resetPasswordSchema.safeParse({ password: 'pw12345', confirm: 'pw12345' }).success).toBe(true)
    expect(resetPasswordSchema.safeParse({ password: '123', confirm: '123' }).success).toBe(false)
    expect(resetPasswordSchema.safeParse({ password: 'pw12345', confirm: 'other12' }).success).toBe(false)
  })

  it('validates change-password fields', () => {
    expect(
      changePasswordSchema.safeParse({ current: 'old', password: 'pw12345', confirm: 'pw12345' }).success,
    ).toBe(true)
    // Missing current password.
    expect(
      changePasswordSchema.safeParse({ current: '', password: 'pw12345', confirm: 'pw12345' }).success,
    ).toBe(false)
    // Mismatched confirm.
    const r = changePasswordSchema.safeParse({ current: 'old', password: 'pw12345', confirm: 'nope123' })
    expect(r.success).toBe(false)
    if (!r.success) {
      expect(r.error.issues.some((i) => i.path.includes('confirm'))).toBe(true)
    }
  })
})
