import { apiClient } from './client'

export interface Node {
  id: string
  agent_id: string
  hostname: string
  advertised_address: string
  metrics_enabled: boolean
  metrics_scheme: string
  metrics_port: number
  metrics_path: string
  status: string
  last_heartbeat_at?: string
  tenant_id: string
  created_at: string
  updated_at: string
}

export async function fetchNodes(): Promise<Node[]> {
  const resp = await apiClient.get('/api/nodes')
  return resp.data || []
}

export async function fetchNode(id: string): Promise<Node> {
  const resp = await apiClient.get(`/api/nodes/${id}`)
  return resp.data
}
