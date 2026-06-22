import { apiClient } from './client'

// ---- ModelArtifact ----
export interface ModelArtifact {
  id: string; name: string; display_name: string; path: string
  format: string; task_type: string; architecture: string
  size_label: string; quantization: string
  default_context_length: number; estimated_vram_bytes: number
  required_gpu_count: number; tenant_id: string
  capabilities?: string[]
  capability_sources?: Record<string, string>
  default_test_mode?: string
  created_at: string; updated_at: string
  locations?: ModelLocation[]
}

export interface ModelLocation {
  id: string; model_artifact_id: string; node_id: string
  path_type: string; model_root: string; relative_path: string; absolute_path: string
  size_bytes: number; checksum: string; manifest_digest: string
  discovered_metadata_json?: Record<string, any>
  match_status: string; verification_status: string
  manual_override: boolean; override_reason: string
  last_scanned_at: string; last_error: string
  tenant_id: string; created_at: string; updated_at: string
}

export interface DetectedMetadata {
  format?: string
  architecture?: string
  architectures?: string[]
  model_type?: string
  context_length?: number
  max_position_embeddings?: number
  quantization?: string
  quantization_config?: any
  file_size_bytes?: number
  parameter_count?: string
  embedding_length?: number
  block_count?: number
  vocab_size?: number
  head_count?: number
  head_count_kv?: number
  torch_dtype?: string
  rope_scaling?: any
  hidden_size?: number
  num_hidden_layers?: number
  num_attention_heads?: number
  num_key_value_heads?: number
  has_tokenizer?: boolean
  safetensors_count?: number
  warnings?: string[]
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
