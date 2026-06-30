import { describe, expect, it } from 'vitest'
import type { ConfigEditField } from '@/utils/configEditView'
import { resolveConfigFieldHelp, resolveConfigFieldLabel, resolveConfigFieldTooltip } from '@/utils/configEditFieldMeta'

const messages: Record<string, string> = {
  'runtimeFields.dtype.label': '数据类型',
  'runtimeFields.dtype.help': '模型权重和计算使用的数据类型。',
  'runtimeFields.dtype.tooltip': '选择后端 dtype。',
}

const t = (key: string) => messages[key] || key

function field(partial: Partial<ConfigEditField>): ConfigEditField {
  return {
    key: 'custom.self_describing_field',
    internal_key: 'custom.self_describing_field',
    label: '',
    section: 'model_serving',
    order: 1,
    type: 'string',
    widget: 'string',
    value: '',
    enabled: false,
    has_enable: true,
    required: false,
    readonly: false,
    advanced: false,
    ...partial,
  }
}

describe('configEditFieldMeta', () => {
  it('uses field-provided i18n metadata without backend knowledge', () => {
    const f = field({
      key: 'anything.backend_specific',
      label_i18n_key: 'runtimeFields.dtype.label',
      help_i18n_key: 'runtimeFields.dtype.help',
      tooltip_i18n_key: 'runtimeFields.dtype.tooltip',
      cli_flag: '--dtype',
    })
    expect(resolveConfigFieldLabel(f, t)).toBe('数据类型')
    expect(resolveConfigFieldHelp(f, t)).toBe('模型权重和计算使用的数据类型。')
    expect(resolveConfigFieldTooltip(f, t)).toContain('--dtype')
    expect(resolveConfigFieldTooltip(f, t)).toContain('模型权重和计算使用的数据类型。')
  })

  it('falls back to field label/title/help metadata before humanizing keys', () => {
    const f = field({ label: 'Runtime Batch Size', title: 'Ignored title', help: 'Self-contained help.' })
    expect(resolveConfigFieldLabel(f, t)).toBe('Runtime Batch Size')
    expect(resolveConfigFieldHelp(f, t)).toBe('Self-contained help.')
  })

  it('humanizes unknown keys only when metadata is missing', () => {
    const f = field({ key: 'vendor.future_new_param', internal_key: 'vendor.future_new_param', label: '配置项' })
    expect(resolveConfigFieldLabel(f, t)).toContain('Vendor future new param')
  })
})
