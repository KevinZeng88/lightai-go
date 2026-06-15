import { apiClient } from './client'

export interface GPU {
  id: string
  node_id: string
  vendor: string
  index: number
  name: string
  uuid: string
  pci_bus_id: string
  driver_version: string
  memory_total_bytes: number
  memory_used_bytes: number
  memory_free_bytes: number
  memory_utilization_percent?: number
  gpu_utilization_percent?: number
  temperature_celsius?: number
  power_draw_watts?: number
  health: string
  status: string
  collected_at?: string
  created_at: string
  updated_at: string
}

export async function fetchGPUs(params?: { node_id?: string; vendor?: string }): Promise<GPU[]> {
  const query = new URLSearchParams()
  if (params?.node_id) query.set('node_id', params.node_id)
  if (params?.vendor) query.set('vendor', params.vendor)
  const qs = query.toString()
  const data = await apiClient.get('/gpus' + (qs ? '?' + qs : ''))
  return Array.isArray(data) ? data : []
}

export async function fetchGPU(id: string): Promise<GPU> {
  return await apiClient.get(`/api/v1/gpus/${id}`)
}
