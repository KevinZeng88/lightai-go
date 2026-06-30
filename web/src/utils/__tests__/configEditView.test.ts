import { describe, expect, it } from 'vitest'
import { displayGroupForField, sortedSections, type ConfigEditField, type ConfigEditView } from '../configEditView'

function field(partial: Partial<ConfigEditField>): ConfigEditField {
  return {
    key: partial.key || 'field',
    internal_key: partial.internal_key || partial.key || 'field',
    label: partial.label || partial.key || 'field',
    section: partial.section || 'model_serving',
    order: partial.order ?? 10,
    type: partial.type || 'string',
    widget: partial.widget || 'string',
    value: partial.value ?? null,
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

describe('configEditView grouping and ordering', () => {
  it('groups enabled, common, advanced, and expert parameters in product order', () => {
    const sections = sortedSections(view([
      field({ key: 'model_runtime.max_model_len', enabled: false, original_enabled: false, section: 'model_serving', order: 20 }),
      field({ key: 'model_runtime.tensor_parallel_size', enabled: false, original_enabled: false, advanced: true, view: 'advanced', section: 'backend_runtime', order: 10 }),
      field({ key: 'launcher.docker_options.privileged', enabled: false, original_enabled: false, view: 'security', section: 'security_high_risk', order: 1 }),
      field({ key: 'runtime.health', enabled: true, original_enabled: true, section: 'health_check', order: 1 }),
    ]))
    expect(sections.map(section => section.key)).toEqual([
      'enabled_parameters',
      'common_parameters',
      'advanced_parameters_group',
      'expert_parameters_group',
    ])
    expect(sections[0].fields[0].key).toBe('runtime.health')
    expect(sections[3].fields[0].key).toBe('launcher.docker_options.privileged')
  })

  it('keeps enabled high-risk and diagnostic fields in the expert group', () => {
    expect(displayGroupForField(field({
      key: 'docker.privileged',
      enabled: true,
      original_enabled: true,
      section: 'security_high_risk',
      view: 'security',
    }))).toBe('expert')
    expect(displayGroupForField(field({
      key: 'advanced_raw_diag',
      enabled: true,
      original_enabled: true,
      section: 'advanced_raw',
      diagnostic: true,
    }))).toBe('expert')
  })

  it('sorts fields inside a display group by section, display order, and key', () => {
    const sections = sortedSections(view([
      field({ key: 'runtime.health', section: 'health_check', order: 1 }),
      field({ key: 'model_runtime.dtype', section: 'model_serving', order: 2 }),
      field({ key: 'model_runtime.max_model_len', section: 'model_serving', order: 1 }),
    ]))
    expect(sections[0].key).toBe('common_parameters')
    expect(sections[0].fields.map(item => item.key)).toEqual([
      'model_runtime.max_model_len',
      'model_runtime.dtype',
      'runtime.health',
    ])
  })

  it('keeps group placement stable during editing until reload', () => {
    const editable = field({ key: 'model_runtime.max_model_len', enabled: false, original_enabled: false })
    expect(displayGroupForField(editable)).toBe('common')

    editable.enabled = true
    expect(displayGroupForField(editable)).toBe('common')

    const reloadedEnabled = field({ key: 'model_runtime.max_model_len', enabled: true, original_enabled: true })
    expect(displayGroupForField(reloadedEnabled)).toBe('enabled')

    const reloadedDisabled = field({ key: 'model_runtime.max_model_len', enabled: false, original_enabled: false })
    expect(displayGroupForField(reloadedDisabled)).toBe('common')
  })

  it('keeps raw and diagnostic fields in the expert group', () => {
    expect(displayGroupForField(field({ key: 'advanced_raw_diag', section: 'advanced_raw', diagnostic: true }))).toBe('expert')
    expect(displayGroupForField(field({ key: 'internal.debug', visibility: 'internal' }))).toBe('expert')
  })
})
