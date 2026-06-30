import { apiClient } from './client'
import type { ConfigEditPatch, ConfigEditView } from '@/utils/configEditView'

export async function getConfigEditView(payload: {
  object_kind: string
  object_id: string
  layer: string
  mode?: string
  view_level?: 'normal' | 'advanced' | 'security' | 'developer'
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

export async function listConfigEditTemplates(): Promise<any> {
  return apiClient.get('/config-edit/templates')
}

export async function getConfigEditTemplate(id: string): Promise<any> {
  return apiClient.get(`/config-edit/templates/${encodeURIComponent(id)}`)
}

export async function validateConfigEditTemplate(template: Record<string, any>): Promise<any> {
  return apiClient.post('/config-edit/templates/validate', template)
}

export async function cloneConfigEditTemplate(id: string): Promise<any> {
  return apiClient.post(`/config-edit/templates/${encodeURIComponent(id)}/clone`, {})
}
