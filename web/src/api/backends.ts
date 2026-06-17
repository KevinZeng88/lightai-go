import { apiClient } from './client'

export interface InferenceBackend {
  id: string; name: string; display_name: string; description: string
  protocol_json: any; default_version: string; parameter_format: string
  common_parameters_json: any; default_env_json: any
  is_builtin: boolean; is_enabled: boolean; created_at: string; updated_at: string
}

export interface BackendVersion {
  id: string; backend_id: string; version: string; display_name: string
  is_default: boolean; default_entrypoint_json: any; default_args_json: any
  default_backend_params_json: any; parameter_defs_json: any
  health_check_json: any; default_container_port: number
  default_images_json: any; env_json: any
  is_deprecated: boolean; created_at: string; updated_at: string
}

export interface BackendRuntimeTemplate {
  name: string; source: string; content: string
}

export async function listBackends(): Promise<InferenceBackend[]> {
  return apiClient.get('/api/v1/inference-backends')
}

export async function getBackend(id: string): Promise<InferenceBackend> {
  return apiClient.get(`/api/v1/inference-backends/${id}`)
}

export async function listBackendVersions(backendId: string): Promise<BackendVersion[]> {
  return apiClient.get(`/api/v1/inference-backends/${backendId}/versions`)
}

export async function listRuntimeTemplates(): Promise<BackendRuntimeTemplate[]> {
  return apiClient.get('/api/v1/backend-runtime-templates')
}

export async function getRuntimeTemplate(name: string): Promise<BackendRuntimeTemplate> {
  return apiClient.get(`/api/v1/backend-runtime-templates/${name}`)
}
