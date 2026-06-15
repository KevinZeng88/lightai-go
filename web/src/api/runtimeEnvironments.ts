import { apiClient } from './client'

export interface DockerSpec {
  id?: string
  image?: string
  image_pull_policy?: string
  devices?: any
  privileged?: any
  ipc_mode?: any
  uts_mode?: any
  network_mode?: any
  shm_size?: any
  group_add?: any
  security_options?: any
  ulimits?: any
  restart_policy?: any
  gpu_visible_env_key?: string
}

export interface RuntimeEnvironment {
  id: string
  name: string
  display_name: string
  runtime_type: string
  backend_type: string
  vendor: string
  openai_compatible: boolean
  default_port: number
  health_check_path: string
  description: string
  tenant_id: string
  owner_id: string
  created_by: string
  updated_by: string
  created_at: string
  updated_at: string
  docker?: DockerSpec
}

export async function fetchRuntimeEnvironments(): Promise<RuntimeEnvironment[]> {
  const data = await apiClient.get('/runtime-environments')
  return Array.isArray(data) ? data : []
}

export async function fetchRuntimeEnvironment(id: string): Promise<RuntimeEnvironment> {
  return apiClient.get(`/runtime-environments/${id}`)
}

export async function createRuntimeEnvironment(body: any): Promise<RuntimeEnvironment> {
  return apiClient.post('/runtime-environments', body)
}

export async function updateRuntimeEnvironment(id: string, body: any): Promise<RuntimeEnvironment> {
  return apiClient.patch(`/runtime-environments/${id}`, body)
}

export async function deleteRuntimeEnvironment(id: string): Promise<void> {
  await apiClient.delete(`/runtime-environments/${id}`)
}
