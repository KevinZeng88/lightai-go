import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiClient, ApiError } from '@/api/client'

export interface UserInfo {
  id: string
  username: string
  display_name: string
  is_platform_admin: boolean
  must_change_password: boolean
}

export interface TenantInfo {
  id: string
  name: string
}

export interface RoleInfo {
  id: string
  name: string
  built_in: boolean
}

export const useAuthStore = defineStore('auth', () => {
  const user = ref<UserInfo | null>(null)
  const tenant = ref<TenantInfo | null>(null)
  const csrfToken = ref<string>('')
  const permissions = ref<string[]>([])
  const roles = ref<RoleInfo[]>([])
  const mustChangePassword = ref(false)
  const isLoggedIn = ref(false)

  async function login(username: string, password: string, tenantId?: string) {
    // P0-003/CODEX: Pass tenant_id for multi-tenant login.
    const body: Record<string, string> = { username, password }
    if (tenantId) body.tenant_id = tenantId
    // P0-007: apiClient.post now throws on non-2xx.
    const data = await apiClient.post('/api/v1/auth/login', body)

    // P0-007: Only set logged-in state on successful response.
    user.value = {
      id: data.user_id,
      username: data.username,
      display_name: data.display_name,
      is_platform_admin: data.is_platform_admin,
      must_change_password: data.must_change_password,
    }
    csrfToken.value = data.csrf_token || ''
    mustChangePassword.value = data.must_change_password || false
    isLoggedIn.value = true
    return data
  }

  async function fetchMe() {
    try {
      // P0-007: apiClient.get now throws on non-2xx.
      const data = await apiClient.get('/api/v1/auth/me')
      user.value = {
        id: data.user.id,
        username: data.user.username,
        display_name: data.user.display_name,
        is_platform_admin: data.user.is_platform_admin,
        must_change_password: data.user.must_change_password,
      }
      tenant.value = { id: data.tenant.id, name: data.tenant.name }
      roles.value = data.roles || []
      permissions.value = data.permissions || []
      mustChangePassword.value = data.user.must_change_password || false
      // P0-007: CSRF token may be refreshed via /me.
      if (data.csrf_token) {
        csrfToken.value = data.csrf_token
      }
      isLoggedIn.value = true
    } catch (e) {
      // P0-007: On any auth fetch failure, clear state.
      isLoggedIn.value = false
      user.value = null
      tenant.value = null
      csrfToken.value = ''
      permissions.value = []
      roles.value = []
    }
  }

  async function logout() {
    try {
      await apiClient.post('/api/v1/auth/logout', {})
    } catch {
      // Ignore errors during logout — we clear state anyway.
    }
    isLoggedIn.value = false
    user.value = null
    tenant.value = null
    csrfToken.value = ''
    permissions.value = []
    roles.value = []
  }

  async function changePassword(currentPassword: string, newPassword: string) {
    const data = await apiClient.post('/api/v1/auth/change-password', {
      current_password: currentPassword,
      new_password: newPassword,
    })
    // P0-007: After password change, clear must_change_password flag.
    if (data.status === 'ok') {
      mustChangePassword.value = false
      if (user.value) {
        user.value.must_change_password = false
      }
    }
    return data
  }

  return {
    user, tenant, csrfToken, permissions, roles,
    mustChangePassword, isLoggedIn,
    login, fetchMe, logout, changePassword,
  }
})
