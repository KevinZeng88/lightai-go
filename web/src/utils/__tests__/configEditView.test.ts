import { describe, expect, it } from 'vitest'
import {
  buildConfigEditPatch,
  displayGroupForField,
  sortedSections,
  type ConfigEditField,
  type ConfigEditView,
} from '../configEditView'

function field(partial: Partial<ConfigEditField>): ConfigEditField {
  return {
    key: partial.key || 'field',
    internal_key: partial.internal_key || partial.key || 'field',
    semantic_key: partial.semantic_key,
    path: partial.path,
    label: partial.label || partial.key || 'field',
    section: partial.section || 'model_serving',
    order: partial.order ?? 10,
    type: partial.type || 'string',
    widget: partial.widget || 'string',
    value: partial.value ?? null,
    original_value: partial.original_value,
    enabled: partial.enabled ?? false,
    original_enabled: partial.original_enabled,
    has_enable: partial.has_enable ?? true,
    required: partial.required ?? false,
    readonly: partial.readonly ?? false,
    advanced: partial.advanced ?? false,
    view: partial.view,
    tier: partial.tier,
    visibility: partial.visibility,
    diagnostic: partial.diagnostic,
    risk: partial.risk,
  }
}

function view(fields: ConfigEditField[]): ConfigEditView {
  return {
    layer: 'backend_runtime',
    object_id: 'rt-test',
    object_kind: 'backend_runtime',
    sections: [{ key: 'model_serving', label: 'Model', order: 10, fields }],
  }
}

function patchView(fields: ConfigEditField[]): ConfigEditView {
  return {
    layer: 'node_backend_runtime',
    object_id: 'nbr-test',
    object_kind: 'node_backend_runtime',
    sections: [{ key: 'model_serving', label: 'Model', order: 10, fields }],
  }
}

describe('configEditView grouping and ordering', () => {
  it.each([
    ['enabled=true normal -> enabled', field({ key: 'normal.enabled', enabled: true, original_enabled: true }), 'enabled'],
    ['enabled=true advanced -> enabled', field({ key: 'advanced.enabled', enabled: true, original_enabled: true, advanced: true, view: 'advanced' }), 'enabled'],
    ['enabled=true expert -> enabled', field({ key: 'expert.enabled', enabled: true, original_enabled: true, tier: 'expert', view: 'developer' }), 'enabled'],
    ['enabled=true high-risk/security -> enabled', field({ key: 'security.enabled', enabled: true, original_enabled: true, section: 'security_high_risk', view: 'security', risk: 'high' }), 'enabled'],
    ['enabled=true raw/diagnostic -> enabled', field({ key: 'raw.enabled', enabled: true, original_enabled: true, section: 'advanced_raw', diagnostic: true }), 'enabled'],
    ['enabled=false high-risk/security -> expert', field({ key: 'security.disabled', enabled: false, original_enabled: false, section: 'security_high_risk', view: 'security', risk: 'high' }), 'expert'],
    ['enabled=false raw/diagnostic -> expert', field({ key: 'raw.disabled', enabled: false, original_enabled: false, section: 'advanced_raw', diagnostic: true }), 'expert'],
    ['enabled=false diagnostic only -> common', field({ key: 'normal.diagnostic_only', enabled: false, original_enabled: false, diagnostic: true }), 'common'],
    ['enabled=false advanced -> advanced', field({ key: 'advanced.disabled', enabled: false, original_enabled: false, advanced: true, view: 'advanced' }), 'advanced'],
    ['enabled=false normal -> common', field({ key: 'normal.disabled', enabled: false, original_enabled: false }), 'common'],
  ])('%s', (_name, input, expected) => {
    expect(displayGroupForField(input)).toBe(expected)
  })

  it('groups mixed enabled and disabled fields in product order', () => {
    const sections = sortedSections(view([
      field({ key: 'normal.enabled', enabled: true, original_enabled: true, section: 'model_serving', order: 1 }),
      field({ key: 'advanced.enabled', enabled: true, original_enabled: true, advanced: true, view: 'advanced', section: 'backend_runtime', order: 1 }),
      field({ key: 'security.enabled', enabled: true, original_enabled: true, view: 'security', section: 'security_high_risk', risk: 'high', order: 1 }),
      field({ key: 'raw.enabled', enabled: true, original_enabled: true, section: 'advanced_raw', diagnostic: true, order: 1 }),
      field({ key: 'normal.disabled', enabled: false, original_enabled: false, section: 'model_serving', order: 2 }),
      field({ key: 'advanced.disabled', enabled: false, original_enabled: false, advanced: true, view: 'advanced', section: 'backend_runtime', order: 2 }),
      field({ key: 'security.disabled', enabled: false, original_enabled: false, view: 'security', section: 'security_high_risk', risk: 'high', order: 2 }),
      field({ key: 'raw.disabled', enabled: false, original_enabled: false, section: 'advanced_raw', diagnostic: true, order: 2 }),
    ]))
    expect(sections.map(section => section.key)).toEqual([
      'enabled_parameters',
      'common_parameters',
      'advanced_parameters_group',
      'expert_parameters_group',
    ])
    expect(sections[0].fields.map(item => item.key)).toEqual([
      'normal.enabled',
      'advanced.enabled',
      'security.enabled',
      'raw.enabled',
    ])
    expect(sections[1].fields.map(item => item.key)).toEqual(['normal.disabled'])
    expect(sections[2].fields.map(item => item.key)).toEqual(['advanced.disabled'])
    expect(sections[3].fields.map(item => item.key)).toEqual(['security.disabled', 'raw.disabled'])
  })

  it('sorts fields inside a display group by section rank, display order, and key', () => {
    const sections = sortedSections(view([
      field({ key: 'raw.z', section: 'advanced_raw', order: 1, enabled: true, original_enabled: true }),
      field({ key: 'security.a', section: 'security_high_risk', order: 2, enabled: true, original_enabled: true }),
      field({ key: 'docker.a', section: 'docker', order: 1, enabled: true, original_enabled: true }),
      field({ key: 'env.a', section: 'environment', order: 1, enabled: true, original_enabled: true }),
      field({ key: 'health.a', section: 'health_check', order: 1, enabled: true, original_enabled: true }),
      field({ key: 'model.b', section: 'model_serving', order: 2, enabled: true, original_enabled: true }),
      field({ key: 'model.a', section: 'model_serving', order: 1, enabled: true, original_enabled: true }),
    ]))
    expect(sections[0].key).toBe('enabled_parameters')
    expect(sections[0].fields.map(item => item.key)).toEqual([
      'model.a',
      'model.b',
      'health.a',
      'env.a',
      'docker.a',
      'security.a',
      'raw.z',
    ])
  })

  it('keeps group placement stable during editing until reload', () => {
    const disabledAtLoad = field({ key: 'normal.toggle', enabled: false, original_enabled: false })
    expect(displayGroupForField(disabledAtLoad)).toBe('common')
    disabledAtLoad.enabled = true
    expect(displayGroupForField(disabledAtLoad)).toBe('common')

    const enabledAtLoad = field({ key: 'normal.untoggle', enabled: true, original_enabled: true })
    expect(displayGroupForField(enabledAtLoad)).toBe('enabled')
    enabledAtLoad.enabled = false
    expect(displayGroupForField(enabledAtLoad)).toBe('enabled')

    expect(displayGroupForField(field({ key: 'normal.reload.enabled', enabled: true, original_enabled: true }))).toBe('enabled')
    expect(displayGroupForField(field({ key: 'normal.reload.disabled', enabled: false, original_enabled: false }))).toBe('common')
  })

  it('does not place a field in expert group only because diagnostic is true', () => {
    expect(displayGroupForField(field({
      key: 'model_runtime.tensor_parallel_size',
      enabled: false,
      original_enabled: false,
      diagnostic: true,
      advanced: false,
      section: 'model_serving',
    }))).toBe('common')
  })

  it('builds patches for changed value and enabled state while preserving semantic key and path', () => {
    const patch = buildConfigEditPatch(patchView([
      field({
        key: 'model_runtime.max_model_len.display',
        semantic_key: 'model_runtime.max_model_len',
        internal_key: 'model_runtime.max_model_len',
        path: ['value'],
        value: 8192,
        original_value: 4096,
        enabled: true,
        original_enabled: false,
      }),
    ]))
    expect(patch.fields).toEqual([{
      key: 'model_runtime.max_model_len',
      internal_key: 'model_runtime.max_model_len',
      path: ['value'],
      value: 8192,
      enabled: true,
    }])
  })

  it('keeps value changes when enabled remains false', () => {
    const patch = buildConfigEditPatch(patchView([
      field({ key: 'model_runtime.dtype', value: 'float16', original_value: 'auto', enabled: false, original_enabled: false }),
    ]))
    expect(patch.fields).toHaveLength(1)
    expect(patch.fields[0].value).toBe('float16')
    expect(patch.fields[0].enabled).toBe(false)
  })

  it('forces required fields enabled and skips readonly local changes', () => {
    const patch = buildConfigEditPatch(patchView([
      field({
        key: 'service.container_port',
        value: 8000,
        original_value: 8000,
        required: true,
        has_enable: false,
        enabled: false,
        original_enabled: false,
      }),
      field({
        key: 'readonly.changed',
        value: 'next',
        original_value: 'prev',
        readonly: true,
        enabled: true,
        original_enabled: true,
      }),
    ]))
    expect(patch.fields).toHaveLength(1)
    expect(patch.fields[0].key).toBe('service.container_port')
    expect(patch.fields[0].enabled).toBe(true)
  })

  it('returns no patch when value and enabled are unchanged', () => {
    const patch = buildConfigEditPatch(patchView([
      field({ key: 'unchanged', value: 1, original_value: 1, enabled: false, original_enabled: false }),
    ]))
    expect(patch.fields).toHaveLength(0)
  })
})
