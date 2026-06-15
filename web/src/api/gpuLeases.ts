import { apiClient } from './client'

export interface GpuLease {
  id: string
  gpu_id: string
  node_id: string
  deployment_id: string
  instance_id: string
  tenant_id: string
  status: string
  expires_at: string
  reserved_at?: string
  activated_at?: string
  released_at?: string
  created_at: string
  updated_at: string
}

export async function fetchGpuLeases(): Promise<GpuLease[]> {
  const data = await apiClient.get('/gpu-leases')
  return Array.isArray(data) ? data : []
}

export async function fetchGpuLease(id: string): Promise<GpuLease> {
  return apiClient.get(`/gpu-leases/${id}`)
}
