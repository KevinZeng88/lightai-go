import { apiClient } from './client'

export interface Role { id: string; name: string; display_name: string; description: string; built_in: boolean; status: string }
export interface Permission { id: string; code: string; scope: string; description: string }
export async function fetchRoles(): Promise<Role[]> {
  const data = await apiClient.get('/roles')
  return Array.isArray(data) ? data : []
}
export async function fetchPermissions(): Promise<Permission[]> {
  const data = await apiClient.get('/permissions')
  return Array.isArray(data) ? data : []
}
export async function createRole(body: any): Promise<Role> { return apiClient.post('/roles', body) }
export async function deleteRole(id: string): Promise<void> { await apiClient.delete(`/roles/${id}`) }
export async function updateRolePermissions(id: string, permissionIds: string[]): Promise<void> { await apiClient.put(`/roles/${id}/permissions`, { permission_ids: permissionIds }) }
