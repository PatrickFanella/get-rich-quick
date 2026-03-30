import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { clearTokens, isAuthenticated, setTokens } from '@/lib/auth'

describe('auth', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('returns false when the token expiry is missing', () => {
    localStorage.setItem('access_token', 'access-token')

    expect(isAuthenticated()).toBe(false)
  })

  it('accepts millisecond expiry timestamps in the future', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-03-30T08:00:00Z'))

    setTokens('access-token', 'refresh-token', Date.now() + 60_000)

    expect(isAuthenticated()).toBe(true)
  })

  it('accepts second-based expiry timestamps in the future', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-03-30T08:00:00Z'))

    setTokens('access-token', 'refresh-token', Math.floor((Date.now() + 60_000) / 1000))

    expect(isAuthenticated()).toBe(true)
  })

  it('returns false after clearing tokens', () => {
    setTokens('access-token', 'refresh-token', Date.now() + 60_000)

    clearTokens()

    expect(isAuthenticated()).toBe(false)
  })
})
