import { apiClient } from './client'

export interface MetricsTarget {
  targets: string[]
  labels: Record<string, string>
}

export async function fetchMetricsTargets(): Promise<MetricsTarget[]> {
  const resp = await apiClient.get('/metrics/targets')
  return resp.data || []
}
