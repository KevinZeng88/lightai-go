import { apiClient } from './client'

// ---- ModelArtifact ----
export interface ModelArtifact {
  id: string; name: string; display_name: string; path: string
  format: string; task_type: string; architecture: string
  size_label: string; quantization: string
  default_context_length: number; estimated_vram_bytes: number
  required_gpu_count: number; tenant_id: string
  created_at: string; updated_at: string
  locations?: ModelLocation[]
}

export interface ModelLocation {
  id: string; model_artifact_id: string; node_id: string
  path_type: string; model_root: string; relative_path: string; absolute_path: string
  size_bytes: number; checksum: string; manifest_digest: string
  match_status: string; verification_status: string
  manual_override: boolean; override_reason: string
  last_scanned_at: string; last_error: string
  tenant_id: string; created_at: string; updated_at: string
}

export function listModelArtifacts(): Promise<ModelArtifact[]> {
  return apiClient.get('/model-artifacts')
}
export function getModelArtifact(id: string): Promise<ModelArtifact> {
  return apiClient.get(`/model-artifacts/${id}`)
}
export function createModelArtifact(data: any): Promise<ModelArtifact> {
  return apiClient.post('/model-artifacts', data)
}
export function updateModelArtifact(id: string, data: any): Promise<ModelArtifact> {
  return apiClient.patch(`/model-artifacts/${id}`, data)
}
export function deleteModelArtifact(id: string): Promise<any> {
  return apiClient.delete(`/model-artifacts/${id}`)
}
export function discoverModelArtifact(data: any): Promise<ModelArtifact> {
  return apiClient.post('/model-artifacts/discover', data)
}

// ---- ModelLocation ----
export function createModelLocation(artifactID: string, data: any): Promise<ModelLocation> {
  return apiClient.post(`/model-artifacts/${artifactID}/locations`, data)
}
export function updateModelLocation(artifactID: string, locationID: string, data: any): Promise<ModelLocation> {
  return apiClient.patch(`/model-artifacts/${artifactID}/locations/${locationID}`, data)
}
export function deleteModelLocation(artifactID: string, locationID: string): Promise<any> {
  return apiClient.delete(`/model-artifacts/${artifactID}/locations/${locationID}`)
}
export function rescanModelLocation(artifactID: string, locationID: string): Promise<any> {
  return apiClient.post(`/model-artifacts/${artifactID}/locations/${locationID}/rescan`)
}
export function attestModelLocation(artifactID: string, locationID: string, data: any): Promise<any> {
  return apiClient.post(`/model-artifacts/${artifactID}/locations/${locationID}/attest`, data)
}
