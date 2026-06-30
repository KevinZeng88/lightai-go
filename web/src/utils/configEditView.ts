export type ConfigEditView = {
  layer: string
  object_id: string
  object_kind: string
  template_id?: string
  snapshot_id?: string
  parent?: {
    object_kind: string
    object_id: string
    snapshot_id?: string
  }
  child_init?: {
    strategy: string
    allowed_children?: string[]
    copy_scope?: string
  }
  view_level?: 'normal' | 'advanced' | 'security' | 'developer'
  readonly?: boolean
  sections: ConfigEditSection[]
  components?: ConfigEditComponent[]
  fields?: ConfigEditField[]
  effects_preview?: ConfigEditEffectPreview[]
  diagnostics?: Record<string, any>
  metadata?: Record<string, any>
}

export type ConfigEditComponent = {
  key: string
  type: string
  renderer: string
  label: string
  section: string
  view: 'normal' | 'advanced' | 'security' | 'developer'
  order: number
  fields: string[]
  enabled: boolean
  readonly: boolean
  source?: Record<string, any>
  reset?: ConfigEditReset
  effects?: ConfigEditEffectPreview[]
}

export type ConfigEditReset = {
  allow_reset_to_parent?: boolean
  allow_reset_to_default?: boolean
}

export type ConfigEditEffectPreview = {
  component_key: string
  field_key?: string
  type: string
  target: string
  key?: string
  value?: any
  source?: string
  patch_target?: string
  docker_effect?: string
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
  title_i18n_key?: string
  description_i18n_key?: string
  help_i18n_key?: string
  tooltip_i18n_key?: string
  title?: string
  description?: string
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
  validation_rules?: Record<string, any>
  placeholder?: string
  sensitive?: boolean
  disabled?: boolean
  source?: Record<string, any>
  value_source?: string
  last_value_layer?: string
  inherited_value?: any
  copy_behavior?: string
  override_behavior?: string
  disable_behavior?: string
  patch_target?: string
  copied_from?: string
  dirty?: boolean
  warnings?: any[]
  diagnostic?: boolean
  original_value?: any
  original_enabled?: boolean
  component_key?: string
  view?: 'normal' | 'advanced' | 'security' | 'developer'
  reset?: ConfigEditReset
  effects?: ConfigEditEffectPreview[]
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
