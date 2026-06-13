import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiClient } from '@/api/client'

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

  async function login(username: string, password: string) {
    const resp = await apiClient.post('/api/auth/login', { username, password })
    const data = resp.data
    user.value = {
      id: data.user_id,
      username: data.username,
      display_name: data.display_name,
      is_platform_admin: data.is_platform_admin,
      must_change_password: data.must_change_password,
    }
    csrfToken.value = data.csrf_token
    mustChangePassword.value = data.must_change_password
    isLoggedIn.value = true
    return data
  }

  async function fetchMe() {
    try {
      const resp = await apiClient.get('/api/auth/me')
      const data = resp.data
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
      mustChangePassword.value = data.user.must_change_password
      isLoggedIn.value = true
    } catch {
      isLoggedIn.value = false
      user.value = null
    }
  }

  async function logout() {
    try {
      await apiClient.post('/api/auth/logout', {})
    } catch { /* ignore */ }
    isLoggedIn.value = false
    user.value = null
    tenant.value = null
    csrfToken.value = ''
    permissions.value = []
    roles.value = []
  }

  async function changePassword(currentPassword: string, newPassword: string) {
    const resp = await apiClient.post('/api/auth/change-password', {
      current_password: currentPassword,
      new_password: newPassword,
    })
    return resp.data
  }

  return {
    user, tenant, csrfToken, permissions, roles,
    mustChangePassword, isLoggedIn,
    login, fetchMe, logout, changePassword,
  }
})
