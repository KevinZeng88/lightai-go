export interface RuntimeTemplateDisplay {
  id: string
  displayName: string
  rawName: string
  rawId: string
  sourceType: 'builtin' | 'user'
  sourceLabel: string
  vendor: string
  vendorDisplay: string
  backend: string
  backendDisplay: string
  version: string
  versionDisplay: string
  image: string
  formats: string[]
  readyCount: number
  managedBy: 'system' | 'user'
  raw: any
}

// Product-friendly display names for backends.
const BACKEND_DISPLAY: Record<string, string> = {
  vllm: 'vLLM',
  sglang: 'SGLang',
  llamacpp: 'llama.cpp',
  ollama: 'Ollama',
}

// Product-friendly display names for vendors.
const VENDOR_DISPLAY: Record<string, string> = {
  nvidia: 'NVIDIA',
  metax: 'MetaX',
  huawei: 'Huawei Ascend',
  ascend: 'Huawei Ascend',
  cpu: 'CPU',
}

export function toRuntimeTemplateDisplay(row: any): RuntimeTemplateDisplay {
  const vendor = row.vendor || 'unknown'
  const backendId = (row.backend_id || '').replace(/^backend\./, '')
  const version = extractVersion(row)
  const isEditable = !!row.is_editable

  const backendDisplay = BACKEND_DISPLAY[backendId] || backendId
  const vendorDisplay = VENDOR_DISPLAY[vendor] || vendor
  const versionDisplay = version === '*' ? '*' : (version || '')

  // displayName priority:
  // 1. row.display_name (user-specified), unless it looks like a tech slug
  // 2. Product-friendly: "vLLM / NVIDIA"
  // 3. row.name (normalized), unless it looks like a tech slug
  // 4. row.id
  let displayName = ''
  const rawDisplay = row.display_name || ''
  const rawName = row.name || ''

  // Normalize: strip "runtime." prefix from display_name and name.
  const normalizedDisplay = rawDisplay.replace(/^runtime\./, '')
  const normalizedName = rawName.replace(/^runtime\./, '')

  // Detect tech slugs like "vllm.nvidia-docker", "sglang.metax-docker", "llamacpp.cpu-docker".
  const techSlugPattern = /^[a-z]+\.[a-z]+[-.][a-z0-9]+/
  const displayIsTechSlug = techSlugPattern.test(normalizedDisplay)
  const nameIsTechSlug = techSlugPattern.test(normalizedName)

  // User-configs with a real display_name: use it (unless it's a bare tech slug).
  if (normalizedDisplay.trim() && !displayIsTechSlug) {
    displayName = normalizedDisplay.trim()
  } else if (backendDisplay && vendorDisplay) {
    displayName = `${backendDisplay} / ${vendorDisplay}`
  } else if (normalizedName.trim() && !nameIsTechSlug) {
    displayName = normalizedName.trim()
  } else {
    displayName = row.id || ''
  }

  // Source type and label.
  const sourceType: 'builtin' | 'user' = isEditable ? 'user' : 'builtin'
  const sourceLabel = isEditable ? 'userConfig' : 'builtinTemplate'

  return {
    id: row.id,
    displayName,
    rawName: row.name || '',
    rawId: row.id || '',
    sourceType,
    sourceLabel,
    vendor,
    vendorDisplay,
    backend: backendId,
    backendDisplay,
    version,
    versionDisplay: versionDisplay || version || '',
    image: row.image_ref || '',
    formats: extractSupportedFormats(row),
    readyCount: row.deployable_count ?? 0,
    managedBy: isEditable ? 'user' : 'system',
    raw: row,
  }
}

function extractVersion(row: any): string {
  // Builtin generic runtimes (no specific version) → show *.
  // API guarantees is_builtin and is_editable are present in list response.
  // is_editable === false means builtin/system template.
  const isBuiltin = row.is_builtin === true || row.is_editable === false
  if (isBuiltin) {
    const vid = row.backend_version_id || ''
    if (!vid || vid === 'latest') return '*'
  }
  const vid = row.backend_version_id || ''
  // If version_id looks like "version.vllm.v0.23.0", extract "v0.23.0"
  const match = vid.match(/v\d+\.\d+\.\d+/)
  if (match) return match[0]
  // Otherwise use shortened form
  if (vid.startsWith('version.')) return vid.replace(/^version\./, '').replace(/\..*\./, '.')
  return vid
}

function extractSupportedFormats(row: any): string[] {
  const cs = row.config_set
  if (!cs?.items) return []
  const formats: string[] = []
  for (const item of Object.values(cs.items) as any[]) {
    if (item?.render?.label && item.render.label.toLowerCase().includes('format')) {
      formats.push(item.render.label)
    }
  }
  return formats
}
