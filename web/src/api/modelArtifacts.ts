import { apiClient } from './client'

export interface ModelArtifact {
  id: string
  name: string
  display_name: string
  source_type: string
  path: string
  format: string
  task_type: string
  architecture: string
  size_label: string
  quantization: string
  default_context_length: number
  estimated_vram_bytes: number
  required_gpu_count: number
  tenant_id: string
  owner_id: string
  created_by: string
  updated_by: string
  created_at: string
  updated_at: string
}

export async function fetchModelArtifacts(): Promise<ModelArtifact[]> {
  const data = await apiClient.get('/model-artifacts')
  return Array.isArray(data) ? data : []
}

export async function fetchModelArtifact(id: string): Promise<ModelArtifact> {
  return apiClient.get(`/model-artifacts/${id}`)
}

export async function createModelArtifact(body: Partial<ModelArtifact>): Promise<ModelArtifact> {
  return apiClient.post('/model-artifacts', body)
}

export async function updateModelArtifact(id: string, body: Partial<ModelArtifact>): Promise<ModelArtifact> {
  return apiClient.patch(`/model-artifacts/${id}`, body)
}

export async function deleteModelArtifact(id: string): Promise<void> {
  await apiClient.delete(`/model-artifacts/${id}`)
}
