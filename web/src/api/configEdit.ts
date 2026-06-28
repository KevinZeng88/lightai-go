import { apiClient } from './client'
import type { ConfigEditPatch, ConfigEditView } from '@/utils/configEditView'

export async function getConfigEditView(payload: {
  object_kind: string
  object_id: string
  layer: string
  mode?: string
}): Promise<ConfigEditView> {
  const resp = await apiClient.post('/config-edit/view', payload)
  // Backend returns envelope { config_edit_view, config_view }
  return resp?.config_edit_view ?? resp
}

export async function applyConfigEditPatch(payload: {
  object_kind: string
  object_id: string
  layer: string
  patch: ConfigEditPatch
}): Promise<{ config_set: Record<string, any> }> {
  return apiClient.post('/config-edit/apply', payload)
}
