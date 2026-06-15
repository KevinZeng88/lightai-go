import { apiClient } from './client'

export interface Node {
  id: string
  agent_id: string
  hostname: string
  primary_ip: string
  advertised_address: string
  os: string
  arch: string
  kernel: string
  agent_version: string
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

export interface NodeSystemInfo {
  cpu_utilization_percent: string
  memory_total_bytes: number
  memory_used_bytes: number
  swap_total_bytes: number
  swap_used_bytes: number
  uptime_seconds: string
  cpu_cores: number
  load1: string
  load5: string
  load15: string
  collected_at: string
  filesystems: FilesystemInfo[]
  networks: NetworkInfo[]
}

export interface FilesystemInfo {
  mount_point: string
  device: string
  fs_type: string
  total_bytes: number
  used_bytes: number
  free_bytes: number
  used_percent: string
}

export interface NetworkInfo {
  name: string
  up: boolean
  bytes_recv: number
  bytes_sent: number
}

export async function fetchNodes(): Promise<Node[]> {
  const data = await apiClient.get('/api/nodes')
  return Array.isArray(data) ? data : []
}

export async function fetchNode(id: string): Promise<Node> {
  const data = await apiClient.get(`/api/nodes/${id}`)
  return data
}

// P1-004: Fetch host system metrics for a node.
export async function fetchNodeSystem(id: string): Promise<NodeSystemInfo> {
  const data = await apiClient.get(`/api/nodes/${id}/system`)
  return data
}
