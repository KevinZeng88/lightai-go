import { apiClient } from './client'

export interface RunTemplate {
  id: string
  name: string
  display_name: string
  runtime_type: string
  vendor: string
  backend_type: string
  required_variables?: string[]
  optional_variables?: string[]
  env_mappings?: any
  args_template?: string[]
  volume_mappings?: any
  port_mappings?: any
  backend_flags?: any
  description: string
  tenant_id: string
  owner_id: string
  created_by: string
  updated_by: string
  created_at: string
  updated_at: string
}

export interface RenderPreviewResponse {
  valid: boolean
  errors: string[]
  warnings: string[]
  resolved_run_spec?: any
  equivalent_command_preview?: string
}

export async function fetchRunTemplates(): Promise<RunTemplate[]> {
  const data = await apiClient.get('/run-templates')
  return Array.isArray(data) ? data : []
}

export async function fetchRunTemplate(id: string): Promise<RunTemplate> {
  return apiClient.get(`/run-templates/${id}`)
}

export async function createRunTemplate(body: any): Promise<RunTemplate> {
  return apiClient.post('/run-templates', body)
}

export async function updateRunTemplate(id: string, body: any): Promise<RunTemplate> {
  return apiClient.patch(`/run-templates/${id}`, body)
}

export async function deleteRunTemplate(id: string): Promise<void> {
  await apiClient.delete(`/run-templates/${id}`)
}

export async function renderPreview(id: string, body: any): Promise<RenderPreviewResponse> {
  return apiClient.post(`/run-templates/${id}/render-preview`, body)
}
