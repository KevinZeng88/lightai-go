import { apiClient } from './client'

export interface PreflightResult {
  can_run: boolean
  candidate_nodes: { node_id: string; status: string; warnings?: string[] }[]
  errors: string[]
  warnings: string[]
}

export function preflightDeployment(data: {
  model_artifact_id: string; backend_runtime_id: string
  node_id?: string; gpu_ids?: string[]; host_port?: number
}): Promise<PreflightResult> {
  return apiClient.post('/deployments/preflight', data)
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
