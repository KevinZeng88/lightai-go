export type ConfigEditView = {
  layer: string
  object_id: string
  object_kind: string
  readonly?: boolean
  sections: ConfigEditSection[]
  diagnostics?: Record<string, any>
  metadata?: Record<string, any>
}

export type ConfigEditSection = {
  key: string
  label: string
  description?: string
  order: number
  advanced?: boolean
  collapsed?: boolean
  fields: ConfigEditField[]
}

export type ConfigEditField = {
  key: string
  internal_key: string
  semantic_key?: string
  owner?: string
  tier?: string
  parent_key?: string
  path?: string[]
  label: string
  label_i18n_key?: string
  description_i18n_key?: string
  help?: string
  cli_flag?: string
  env_key?: string
  technical_key?: string
  section: string
  group?: string
  order: number
  type: string
  widget: string
  value: any
  default_value?: any
  enabled: boolean
  has_enable: boolean
  required: boolean
  readonly: boolean
  advanced: boolean
  visibility?: string
  options?: Array<{ label: string, value: any }>
  constraints?: Record<string, any>
  source?: Record<string, any>
  copied_from?: string
  dirty?: boolean
  warnings?: any[]
  diagnostic?: boolean
  original_value?: any
  original_enabled?: boolean
}

export type ConfigEditPatch = {
  layer: string
  object_id: string
  fields: ConfigEditFieldPatch[]
}

export type ConfigEditFieldPatch = {
  key: string
  internal_key: string
  path?: string[]
  value: any
  enabled?: boolean
}

export function cloneEditView(view: ConfigEditView | null): ConfigEditView | null {
  return view ? JSON.parse(JSON.stringify(view)) : null
}

export function buildConfigEditPatch(view: ConfigEditView): ConfigEditPatch {
  const fields: ConfigEditFieldPatch[] = []
  for (const section of sortedSections(view)) {
    for (const field of sortedFields(section)) {
      if (field.readonly) continue
      const nextEnabled = field.required ? true : field.enabled
      const originalEnabled = field.original_enabled ?? nextEnabled
      const hasOriginalValue = Object.prototype.hasOwnProperty.call(field, 'original_value')
      const originalValue = hasOriginalValue ? field.original_value : field.value
      if (stableJSON(field.value) === stableJSON(originalValue) && nextEnabled === originalEnabled) continue
      fields.push({
        key: field.semantic_key || field.key,
        internal_key: field.internal_key,
        path: field.path || [],
        value: field.value,
        enabled: nextEnabled,
      })
    }
  }
  return {
    layer: view.layer,
    object_id: view.object_id,
    fields,
  }
}

function stableJSON(value: any): string {
  return JSON.stringify(value ?? null)
}

export function sortedSections(view: ConfigEditView): ConfigEditSection[] {
  return [...(view.sections || [])].sort((a, b) => (a.order || 0) - (b.order || 0))
}

export function sortedFields(section: ConfigEditSection): ConfigEditField[] {
  return [...(section.fields || [])].sort((a, b) => {
    if ((a.order || 0) === (b.order || 0)) return a.label.localeCompare(b.label)
    return (a.order || 0) - (b.order || 0)
  })
}
