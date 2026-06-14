/**
 * P0-003/CODEX: Multi-tenant login UI verification.
 * Verifies that selectedTenantId flows through auth.login() into the API request body.
 *
 * Run: npx vitest run src/stores/__tests__/auth.test.ts
 * Or: npm test (if configured)
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

// Mock the apiClient
vi.mock('@/api/client', () => ({
  apiClient: {
    post: vi.fn(),
    get: vi.fn(),
  },
  ApiError: class extends Error {
    status: number
    data: any
    constructor(status: number, data: any) {
      super(data?.error || `HTTP ${status}`)
      this.status = status
      this.data = data
    }
  },
}))

import { useAuthStore } from '@/stores/auth'
import { apiClient } from '@/api/client'

describe('P0-003: Multi-tenant login', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('sends tenant_id when tenantId is provided', async () => {
    const mockPost = vi.mocked(apiClient.post)
    mockPost.mockResolvedValueOnce({
      user_id: 'u1',
      username: 'admin',
      display_name: 'Admin',
      is_platform_admin: true,
      must_change_password: false,
      tenant_id: 'tenant-a',
      tenant_name: 'Tenant A',
      csrf_token: 'csrf-abc',
    })

    const auth = useAuthStore()
    await auth.login('admin', 'password', 'tenant-a')

    // Verify tenant_id was included in the request body.
    expect(mockPost).toHaveBeenCalledWith('/api/auth/login', {
      username: 'admin',
      password: 'password',
      tenant_id: 'tenant-a',
    })
  })

  it('does not send tenant_id when not provided (single-tenant)', async () => {
    const mockPost = vi.mocked(apiClient.post)
    mockPost.mockResolvedValueOnce({
      user_id: 'u1',
      username: 'admin',
      display_name: 'Admin',
      is_platform_admin: true,
      must_change_password: false,
      tenant_id: 'default',
      tenant_name: 'Default',
      csrf_token: 'csrf-abc',
    })

    const auth = useAuthStore()
    await auth.login('admin', 'password') // no tenantId

    expect(mockPost).toHaveBeenCalledWith('/api/auth/login', {
      username: 'admin',
      password: 'password',
    })
  })

  it('sets isLoggedIn after successful login (tenant set via fetchMe)', async () => {
    const mockPost = vi.mocked(apiClient.post)
    mockPost.mockResolvedValueOnce({
      user_id: 'u1',
      username: 'admin',
      display_name: 'Admin',
      is_platform_admin: false,
      must_change_password: false,
      csrf_token: 'csrf-xyz',
    })

    const auth = useAuthStore()
    await auth.login('admin', 'password', 'tenant-b')

    expect(auth.isLoggedIn).toBe(true)
    expect(auth.user?.username).toBe('admin')
    expect(auth.csrfToken).toBe('csrf-xyz')
  })

  it('does not set logged-in state on login failure', async () => {
    const mockPost = vi.mocked(apiClient.post)
    mockPost.mockRejectedValueOnce(new (await import('@/api/client')).ApiError(401, { error: 'invalid credentials' }))

    const auth = useAuthStore()
    try {
      await auth.login('admin', 'wrong')
    } catch {
      // expected
    }

    expect(auth.isLoggedIn).toBe(false)
  })
})
