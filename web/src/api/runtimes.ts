import { apiClient } from './client'

export interface BackendRuntime {
  id: string; name: string; display_name: string
  backend_id: string; backend_version_id: string; source_template_name: string
  vendor: string; runtime_type: string; image_name: string; image_pull_policy: string
  entrypoint_override_json: any; args_override_json: any; default_env_json: any
  docker_json: any; model_mount_json: any; health_check_override_json: any
  parameter_schema_json?: any[]; parameter_values_json?: any[]
  source_backend_id?: string; source_backend_version_id?: string; source_version_revision?: string; version_snapshot_json?: any
  node_count?: number; ready_count?: number
  is_builtin: boolean; is_editable: boolean; tenant_id: string
  created_at: string; updated_at: string
}

export async function listRuntimes(): Promise<BackendRuntime[]> {
  return apiClient.get('/backend-runtimes')
}

export async function getRuntime(id: string): Promise<BackendRuntime> {
  return apiClient.get(`/backend-runtimes/${id}`)
}

export async function createRuntimeFromTemplate(data: Record<string, any>): Promise<BackendRuntime> {
  return apiClient.post('/backend-runtimes/from-template', data)
}

export async function patchRuntime(id: string, data: Record<string, any>): Promise<BackendRuntime> {
  return apiClient.patch(`/backend-runtimes/${id}`, data)
}

export async function deleteRuntime(id: string): Promise<any> {
  return apiClient.delete(`/backend-runtimes/${id}`)
}
