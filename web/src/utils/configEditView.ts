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

export type ConfigEditDisplayGroup = 'enabled' | 'common' | 'advanced' | 'expert'

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
  display_group?: ConfigEditDisplayGroup
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
  const fields = (view.sections || []).flatMap(section =>
    (section.fields || []).map(field => ({ ...field, section: field.section || section.key })),
  )
  if (!fields.length) {
    return [...(view.sections || [])].sort((a, b) => (a.order || 0) - (b.order || 0))
  }
  const groups: Record<ConfigEditDisplayGroup, ConfigEditField[]> = {
    enabled: [],
    common: [],
    advanced: [],
    expert: [],
  }
  for (const field of fields) {
    groups[displayGroupForField(field)].push(field)
  }
  return DISPLAY_GROUPS
    .filter(group => groups[group.key].length > 0)
    .map(group => ({
      key: group.sectionKey,
      label: group.label,
      order: group.order,
      advanced: group.key === 'advanced' || group.key === 'expert',
      collapsed: group.key === 'expert',
      fields: sortedFields({ key: group.sectionKey, label: group.label, order: group.order, fields: groups[group.key] }),
    }))
}

export function sortedFields(section: ConfigEditSection): ConfigEditField[] {
  return [...(section.fields || [])].sort((a, b) => {
    const sectionDelta = sectionRank(a.section) - sectionRank(b.section)
    if (sectionDelta !== 0) return sectionDelta
    const orderDelta = (a.order || 0) - (b.order || 0)
    if (orderDelta !== 0) return orderDelta
    return fieldSortKey(a).localeCompare(fieldSortKey(b))
  })
}

const DISPLAY_GROUPS: Array<{ key: ConfigEditDisplayGroup, sectionKey: string, label: string, order: number }> = [
  { key: 'enabled', sectionKey: 'enabled_parameters', label: 'Enabled parameters', order: 10 },
  { key: 'common', sectionKey: 'common_parameters', label: 'Common parameters', order: 20 },
  { key: 'advanced', sectionKey: 'advanced_parameters_group', label: 'Advanced parameters', order: 30 },
  { key: 'expert', sectionKey: 'expert_parameters_group', label: 'Expert parameters', order: 40 },
]

const SECTION_RANKS: Record<string, number> = {
  model: 10,
  model_serving: 10,
  runtime: 20,
  backend_runtime: 20,
  resource: 30,
  container_resources: 30,
  service: 40,
  health: 50,
  health_check: 50,
  mount: 60,
  devices_mounts: 60,
  env: 70,
  environment: 70,
  docker: 80,
  security: 90,
  security_high_risk: 90,
  raw: 100,
  advanced_raw: 100,
}

export function displayGroupForField(field: ConfigEditField): ConfigEditDisplayGroup {
  if (isExpertField(field)) return 'expert'
  const enabledAtLoad = field.original_enabled ?? field.enabled
  if (enabledAtLoad) return 'enabled'
  if (field.advanced || field.view === 'advanced' || field.tier === 'advanced') return 'advanced'
  return 'common'
}

function isExpertField(field: ConfigEditField): boolean {
  return field.view === 'developer' ||
    field.view === 'security' ||
    field.tier === 'expert' ||
    field.section === 'security_high_risk' ||
    field.section === 'advanced_raw' ||
    field.visibility === 'internal' ||
    field.visibility === 'hidden' ||
    !!field.diagnostic
}

function sectionRank(section?: string): number {
  if (!section) return 999
  return SECTION_RANKS[section] ?? 999
}

function fieldSortKey(field: ConfigEditField): string {
  return field.path?.join('.') || field.semantic_key || field.key || field.internal_key || field.label || ''
}
