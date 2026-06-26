export interface RuntimeTemplateDisplay {
  id: string
  displayName: string
  vendor: string
  backend: string
  version: string
  image: string
  formats: string[]
  readyCount: number
  managedBy: 'system' | 'user'
  raw: any
}

export function toRuntimeTemplateDisplay(row: any): RuntimeTemplateDisplay {
  const vendor = row.vendor || 'unknown'
  const backendId = (row.backend_id || '').replace(/^backend\./, '')
  const version = extractVersion(row)

  return {
    id: row.id,
    displayName: `${vendor}.${backendId}`,
    vendor,
    backend: backendId,
    version,
    image: row.image_ref || '',
    formats: extractSupportedFormats(row),
    readyCount: row.deployable_count ?? 0,
    managedBy: row.is_editable ? 'user' : 'system',
    raw: row,
  }
}

function extractVersion(row: any): string {
  // Try to get version from source_template_name (e.g., "vllm-nvidia-docker" → we want the version from backend_version_id)
  // Fall back to backend_version_id or source
  const vid = row.backend_version_id || ''
  // If version_id looks like "version.vllm.v0.23.0", extract "v0.23.0"
  const match = vid.match(/v\d+\.\d+\.\d+/)
  if (match) return match[0]
  // Otherwise just use a shortened form
  if (vid.startsWith('version.')) return vid.replace(/^version\./, '').replace(/\..*\./, '.')
  return vid
}

function extractSupportedFormats(row: any): string[] {
  const cs = row.config_set
  if (!cs?.items) return []
  const formats: string[] = []
  // Check for format-related config items
  for (const item of Object.values(cs.items) as any[]) {
    if (item?.render?.label && item.render.label.toLowerCase().includes('format')) {
      formats.push(item.render.label)
    }
  }
  return formats
}
