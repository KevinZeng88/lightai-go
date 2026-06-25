import { apiClient } from './client'

export interface BackendRuntime {
  id: string
  name: string
  display_name: string
  backend_id: string
  backend_version_id: string
  source_template_name: string
  vendor: string
  runtime_type: string
  image_ref: string
  config_set: Record<string, any>
  source_metadata?: Record<string, any>
  node_count?: number
  ready_count?: number
  deployable_count?: number
  is_builtin: boolean
  is_editable: boolean
  tenant_id: string
  created_at: string
  updated_at: string
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
