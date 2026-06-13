import { useAuthStore } from '@/stores/auth'

const BASE = ''

interface ApiResponse {
  data: any
  status: number
}

class ApiClient {
  private async request(method: string, url: string, body?: any): Promise<ApiResponse> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    }

    // Add CSRF token for state-changing requests.
    const authStore = useAuthStore()
    if (method !== 'GET' && method !== 'HEAD' && authStore.csrfToken) {
      headers['X-CSRF-Token'] = authStore.csrfToken
    }

    const resp = await fetch(BASE + url, {
      method,
      headers,
      credentials: 'include',
      body: body ? JSON.stringify(body) : undefined,
    })

    const data = await resp.json().catch(() => ({}))
    return { data, status: resp.status }
  }

  async get(url: string): Promise<ApiResponse> {
    return this.request('GET', url)
  }

  async post(url: string, body?: any): Promise<ApiResponse> {
    return this.request('POST', url, body)
  }

  async put(url: string, body?: any): Promise<ApiResponse> {
    return this.request('PUT', url, body)
  }

  async delete(url: string): Promise<ApiResponse> {
    return this.request('DELETE', url)
  }
}

export const apiClient = new ApiClient()
