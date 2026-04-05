const DEFAULT_API_BASE_URL = 'http://localhost:8081'

export function getApiBaseUrl() {
  const configuredBaseUrl = (import.meta.env.VITE_API_BASE_URL || '').trim()
  if (configuredBaseUrl) {
    return configuredBaseUrl.replace(/\/$/, '')
  }

  // When accessed remotely, use the same hostname as the browser with the API port.
  if (typeof window !== 'undefined' && window.location.hostname !== 'localhost') {
    return `http://${window.location.hostname}:8081`
  }

  return DEFAULT_API_BASE_URL
}

export function getWebSocketUrl(path = '/ws') {
  const url = new URL(getApiBaseUrl())
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
  url.pathname = path.startsWith('/') ? path : `/${path}`
  url.search = ''
  url.hash = ''
  return url.toString()
}
