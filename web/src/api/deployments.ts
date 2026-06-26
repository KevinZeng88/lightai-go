import { apiClient } from './client'

export interface PreflightResult {
  can_run: boolean
  candidate_nodes: { node_id: string; status: string; warnings?: string[] }[]
  errors: string[]
  warnings: string[]
}

export interface PreviewResult {
  can_run: boolean
  run_plan: any
  docker_preview: string
  lint: { status: string; findings: any[] }
  resource_admission: { status: string; findings: any[] }
  preflight: { status: string; errors: any[]; warnings: any[] }
}

export function preflightDeployment(data: {
  model_artifact_id: string; node_backend_runtime_id: string
  node_id?: string; accelerator_ids?: string[]; host_port?: number
}): Promise<PreflightResult> {
  return apiClient.post('/deployments/preflight', data)
}

export function previewDeployment(data: {
  name?: string; display_name?: string
  model_artifact_id: string; node_backend_runtime_id: string
  placement_json?: Record<string, any>; service_json?: Record<string, any>
  config_overrides?: Record<string, any>
}): Promise<PreviewResult> {
  return apiClient.post('/deployments/preview', data)
}

export function dryRunDeployment(id: string): Promise<any> {
  return apiClient.post(`/deployments/${id}/dry-run`)
}

export function startDeployment(id: string): Promise<any> {
  return apiClient.post(`/deployments/${id}/start`)
}

export function stopDeployment(id: string): Promise<any> {
  return apiClient.post(`/deployments/${id}/stop`)
}

export function createDeployment(data: any): Promise<any> {
  return apiClient.post('/deployments', data)
}
