import { apiClient } from './client'

export interface User {
  id: string; username: string; display_name: string; status: string
  is_platform_admin: boolean; must_change_password: boolean
  created_at: string; updated_at: string
}
export async function fetchUsers(): Promise<User[]> {
  const data = await apiClient.get('/users')
  return Array.isArray(data) ? data : []
}
export async function createUser(body: any): Promise<User> { return apiClient.post('/users', body) }
export async function updateUser(id: string, body: any): Promise<User> { return apiClient.put(`/users/${id}`, body) }
export async function disableUser(id: string): Promise<void> { await apiClient.post(`/users/${id}/disable`) }
export async function resetPassword(id: string, password: string): Promise<void> { await apiClient.post(`/users/${id}/reset-password`, { password }) }
