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
  capabilities_json?: any; docker_options_json?: any; model_mount_json?: any; vendor_options_json?: any
  image_candidates_json?: any; default_endpoints_json?: any; default_args_schema_json?: any
  default_host?: string; protocol?: string; readonly?: boolean; description?: string; managed_by?: string; source?: string
  is_deprecated: boolean; created_at: string; updated_at: string
}

export interface BackendRuntimeTemplate {
  name: string; source: string; content: string
}

export async function listBackends(): Promise<InferenceBackend[]> {
  return apiClient.get('/backends')
}

export async function getBackend(id: string): Promise<InferenceBackend> {
  return apiClient.get(`/backends/${id}`)
}

export async function listBackendVersions(backendId: string): Promise<BackendVersion[]> {
  return apiClient.get(`/backends/${backendId}/versions`)
}

export async function createBackendVersion(backendId: string, data: any): Promise<BackendVersion> {
  return apiClient.post(`/backends/${backendId}/versions`, data)
}

export async function patchBackendVersion(versionId: string, data: any): Promise<BackendVersion> {
  return apiClient.patch(`/backend-versions/${versionId}`, data)
}

export async function cloneBackendVersion(versionId: string): Promise<BackendVersion> {
  return apiClient.post(`/backend-versions/${versionId}/clone`)
}

export async function deleteBackendVersion(versionId: string): Promise<any> {
  return apiClient.delete(`/backend-versions/${versionId}`)
}

export async function reloadBackendCatalog(): Promise<any> {
  return apiClient.post('/backend-catalog/reload')
}

export async function listRuntimeTemplates(): Promise<BackendRuntimeTemplate[]> {
  return apiClient.get('/backend-runtime-templates')
}

export async function getRuntimeTemplate(name: string): Promise<BackendRuntimeTemplate> {
  return apiClient.get(`/backend-runtime-templates/${name}`)
}
