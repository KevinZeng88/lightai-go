import { useAuthStore } from '@/stores/auth'

/** Base path prefix for all LightAI management API calls. */
export const API_BASE = '/api/v1'

export class ApiError extends Error {
  status: number
  data: any

  constructor(status: number, data: any) {
    const msg = typeof data?.error === 'string' ? data.error : `HTTP ${status}`
    super(msg)
    this.name = 'ApiError'
    this.status = status
    this.data = data
  }
}

class ApiClient {
  // All API paths are relative to /api/v1. The client prepends the base.
  private apiBase = '/api/v1'

  private async request(method: string, url: string, body?: any, retryOnCsrf = true): Promise<any> {
    // Auto-prepend /api/v1 unless the URL already starts with it or is an external path.
    const fullUrl = url.startsWith('/api/v1') ? url : (url.startsWith('http') ? url : this.apiBase + url)
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    }

    // Add CSRF token for state-changing requests.
    const authStore = useAuthStore()
    if (method !== 'GET' && method !== 'HEAD' && authStore.csrfToken) {
      headers['X-CSRF-Token'] = authStore.csrfToken
    }

    const resp = await fetch(fullUrl, {
      method,
      headers,
      credentials: 'include',
      body: body ? JSON.stringify(body) : undefined,
    })

    // P0-007: Parse JSON, but handle non-JSON responses gracefully.
    let data: any
    try {
      data = await resp.json()
    } catch {
      data = {}
    }

    // P0-007: Throw structured error on non-2xx responses.
    if (!resp.ok) {
      // If CSRF token may have expired, try refreshing it and retry once.
      if (resp.status === 403 && retryOnCsrf && method !== 'GET' && method !== 'HEAD') {
        const refreshed = await this.refreshCsrfToken()
        if (refreshed) {
          return this.request(method, url, body, false) // Don't retry again
        }
      }

      // If unauthorized, clear auth state.
      if (resp.status === 401) {
        authStore.isLoggedIn = false
        authStore.user = null
        authStore.csrfToken = ''
      }

      throw new ApiError(resp.status, data)
    }

    // P0-007: Extract CSRF token from response if present.
    if (data?.csrf_token) {
      authStore.csrfToken = data.csrf_token
    }

    return data
  }

  // P0-007: Refresh CSRF token from server.
  private async refreshCsrfToken(): Promise<boolean> {
    try {
      const resp = await fetch(API_BASE + '/auth/me', {
        method: 'GET',
        credentials: 'include',
      })
      if (!resp.ok) return false
      const data = await resp.json()
      if (data?.csrf_token) {
        const authStore = useAuthStore()
        authStore.csrfToken = data.csrf_token
        return true
      }
      return false
    } catch {
      return false
    }
  }

  async get(url: string): Promise<any> {
    return this.request('GET', url)
  }

  async post(url: string, body?: any): Promise<any> {
    return this.request('POST', url, body)
  }

  async put(url: string, body?: any): Promise<any> {
    return this.request('PUT', url, body)
  }

  async delete(url: string): Promise<any> {
    return this.request('DELETE', url)
  }

  async patch(url: string, body?: any): Promise<any> {
    return this.request('PATCH', url, body)
  }
}

export const apiClient = new ApiClient()
