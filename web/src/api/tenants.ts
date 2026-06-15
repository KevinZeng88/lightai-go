import { apiClient } from './client'

export interface Tenant {
  id: string; slug: string; name: string; status: string; type: string
  created_at: string; updated_at: string
}
export async function fetchTenants(): Promise<Tenant[]> {
  const data = await apiClient.get('/tenants')
  return Array.isArray(data) ? data : []
}
export async function createTenant(body: any): Promise<Tenant> { return apiClient.post('/tenants', body) }
export async function updateTenant(id: string, body: any): Promise<Tenant> { return apiClient.put(`/tenants/${id}`, body) }
export async function disableTenant(id: string): Promise<void> { await apiClient.post(`/tenants/${id}/disable`) }
