import { apiClient } from './client'

export interface MetricsTarget {
  targets: string[]
  labels: Record<string, string>
}

export async function fetchMetricsTargets(): Promise<MetricsTarget[]> {
  const data = await apiClient.get('/metrics/targets')
  return Array.isArray(data) ? data : []
}
