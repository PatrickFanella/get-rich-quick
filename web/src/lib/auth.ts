const ACCESS_TOKEN_KEY = 'access_token'
const REFRESH_TOKEN_KEY = 'refresh_token'
const EXPIRES_AT_KEY = 'expires_at'

function getStorage() {
  if (typeof window === 'undefined') {
    return null
  }

  return window.localStorage
}

export function getAccessToken() {
  return getStorage()?.getItem(ACCESS_TOKEN_KEY) ?? null
}

export function setTokens(accessToken: string, refreshToken: string, expiresAt: string | number) {
  const storage = getStorage()

  if (!storage) {
    return
  }

  storage.setItem(ACCESS_TOKEN_KEY, accessToken)
  storage.setItem(REFRESH_TOKEN_KEY, refreshToken)
  storage.setItem(EXPIRES_AT_KEY, String(expiresAt))
}

export function clearTokens() {
  const storage = getStorage()

  if (!storage) {
    return
  }

  storage.removeItem(ACCESS_TOKEN_KEY)
  storage.removeItem(REFRESH_TOKEN_KEY)
  storage.removeItem(EXPIRES_AT_KEY)
}

export function isAuthenticated() {
  const accessToken = getAccessToken()

  if (!accessToken) {
    return false
  }

  const expiresAt = getStorage()?.getItem(EXPIRES_AT_KEY)

  if (!expiresAt) {
    return true
  }

  const parsedExpiresAt = Number(expiresAt)

  if (Number.isNaN(parsedExpiresAt)) {
    return false
  }

  return parsedExpiresAt > Date.now()
}
