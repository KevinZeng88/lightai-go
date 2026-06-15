import { apiClient } from './client'

export interface ModelInstance {
  id: string
  deployment_id: string
  replica_index: number
  node_id: string
  agent_id: string
  runtime_type: string
  gpu_ids: string[]
  gpu_lease_ids: string[]
  desired_state: string
  actual_state: string
  container_id: string
  process_id: number
  remote_url: string
  endpoint_url: string
  host_port: number
  container_port: number
  restart_count: number
  last_error: string
  last_exit_code: number
  resolved_run_spec?: any
  started_at: string
  stopped_at: string
  last_heartbeat_at: string
  created_at: string
  updated_at: string
}

export interface LogsResponse {
  instance_id?: string
  logs?: string
  task_id?: string
  status?: string
  message?: string
}

export async function fetchModelInstances(deploymentId?: string): Promise<ModelInstance[]> {
  const qs = deploymentId ? `?deployment_id=${encodeURIComponent(deploymentId)}` : ''
  const data = await apiClient.get(`/model-instances${qs}`)
  return Array.isArray(data) ? data : []
}

export async function fetchModelInstance(id: string): Promise<ModelInstance> {
  return apiClient.get(`/model-instances/${id}`)
}

export async function fetchInstanceLogs(id: string): Promise<LogsResponse> {
  return apiClient.get(`/model-instances/${id}/logs`)
}
