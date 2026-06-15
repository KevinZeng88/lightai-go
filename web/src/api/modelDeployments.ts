import { apiClient } from './client'

export interface ModelDeployment {
  id: string
  name: string
  display_name: string
  model_artifact_id: string
  runtime_environment_id: string
  run_template_id: string
  replicas: number
  desired_state: string
  status: string
  node_id: string
  gpu_ids: string[]
  host_port: number
  served_model_name: string
  max_model_len: number
  tensor_parallel_size: number
  gpu_memory_utilization: number
  dtype: string
  gpu_visible_env_key: string
  env_overrides?: any
  arg_overrides?: any
  extra_args?: any
  schedule_mode: string
  placement_strategy: string
  expose_mode: string
  service_path: string
  tenant_id: string
  owner_id: string
  created_by: string
  updated_by: string
  created_at: string
  updated_at: string
}

export interface DryRunResponse {
  valid: boolean
  errors: string[]
  warnings: string[]
  resolved_run_spec?: any
  equivalent_command_preview?: string
}

export interface StartResponse {
  instance_id?: string
  task_id?: string
  status: string
  equivalent_command_preview?: string
  warnings?: string[]
  existing_task_id?: string
  error?: string
  deployment_status?: string
}

export interface StopResponse {
  instance_id?: string
  task_id?: string
  status: string
  error?: string
  deployment_status?: string
  existing_task_id?: string
}

export async function fetchModelDeployments(): Promise<ModelDeployment[]> {
  const data = await apiClient.get('/model-deployments')
  return Array.isArray(data) ? data : []
}

export async function fetchModelDeployment(id: string): Promise<ModelDeployment> {
  return apiClient.get(`/model-deployments/${id}`)
}

export async function createModelDeployment(body: any): Promise<ModelDeployment> {
  return apiClient.post('/model-deployments', body)
}

export async function updateModelDeployment(id: string, body: any): Promise<ModelDeployment> {
  return apiClient.patch(`/model-deployments/${id}`, body)
}

export async function deleteModelDeployment(id: string): Promise<void> {
  await apiClient.delete(`/model-deployments/${id}`)
}

export async function dryRunDeployment(id: string, body?: any): Promise<DryRunResponse> {
  return apiClient.post(`/model-deployments/${id}/dry-run`, body || {})
}

export async function startDeployment(id: string): Promise<StartResponse> {
  return apiClient.post(`/model-deployments/${id}/start`)
}

export async function stopDeployment(id: string): Promise<StopResponse> {
  return apiClient.post(`/model-deployments/${id}/stop`)
}
