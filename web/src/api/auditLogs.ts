import { apiClient } from './client'

export interface AuditLogEntry {
  id: string
  action: string
  entity_type: string
  entity_id: string
  detail: string
  operator_user_id: string
  created_at: string
}

export interface AuditLogResponse {
  entries: AuditLogEntry[]
  total: number
}

export async function fetchAuditLogs(params?: {
  action?: string
  entity_type?: string
  entity_id?: string
  limit?: number
  offset?: number
}): Promise<AuditLogResponse> {
  const qs = new URLSearchParams()
  if (params?.action) qs.set('action', params.action)
  if (params?.entity_type) qs.set('entity_type', params.entity_type)
  if (params?.entity_id) qs.set('entity_id', params.entity_id)
  if (params?.limit) qs.set('limit', String(params.limit))
  if (params?.offset) qs.set('offset', String(params.offset))
  const url = '/audit-logs' + (qs.toString() ? '?' + qs.toString() : '')
  return apiClient.get(url) as Promise<AuditLogResponse>
}
