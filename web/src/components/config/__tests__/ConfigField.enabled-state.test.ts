// @vitest-environment jsdom
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import ConfigField from '../ConfigField.vue'
import type { ConfigEditField } from '@/utils/configEditView'

function field(partial: Partial<ConfigEditField> = {}): ConfigEditField {
  return {
    key: partial.key || 'model_runtime.max_model_len',
    internal_key: partial.internal_key || partial.key || 'model_runtime.max_model_len',
    label: partial.label || 'Max model length',
    section: partial.section || 'model_serving',
    order: partial.order ?? 1,
    type: partial.type || 'integer',
    widget: partial.widget || 'number',
    value: partial.value ?? 4096,
    enabled: partial.enabled ?? false,
    has_enable: partial.has_enable ?? true,
    required: partial.required ?? false,
    readonly: partial.readonly ?? false,
    advanced: partial.advanced ?? false,
    disabled: partial.disabled,
    tier: partial.tier,
    view: partial.view,
    risk: partial.risk,
    diagnostic: partial.diagnostic,
  }
}

function mountField(config: ConfigEditField, readonly = false) {
  return mount(ConfigField, {
    global: { plugins: [ElementPlus] },
    props: { field: config, readonly },
  })
}

describe('ConfigField enabled/value state', () => {
  it('keeps optional field value control editable when enabled=false', () => {
    const config = field({ enabled: false, value: 4096 })
    const wrapper = mountField(config)
    expect(wrapper.find('[data-testid="config-field-enabled"]').exists()).toBe(true)
    expect(wrapper.findComponent({ name: 'ElInputNumber' }).props('disabled')).not.toBe(true)
  })

  it('does not clear value when enabled is unchecked', async () => {
    const config = field({ enabled: true, value: 8192 })
    const wrapper = mountField(config)
    const checkbox = wrapper.findComponent({ name: 'ElCheckbox' })
    await checkbox.vm.$emit('update:modelValue', false)
    await checkbox.vm.$emit('change', false)
    expect(config.enabled).toBe(false)
    expect(config.value).toBe(8192)
  })

  it('disables checkbox and value control when readonly=true', () => {
    const config = field({ enabled: true, value: 4096 })
    const wrapper = mountField(config, true)
    expect(wrapper.findComponent({ name: 'ElCheckbox' }).props('disabled')).toBe(true)
    expect(wrapper.findComponent({ name: 'ElInputNumber' }).props('disabled')).toBe(true)
  })

  it('does not show enabled checkbox for required fields', () => {
    const config = field({ required: true, has_enable: true, enabled: true })
    const wrapper = mountField(config)
    expect(wrapper.find('[data-testid="config-field-enabled"]').exists()).toBe(false)
  })

  it('emits change when checkbox changes', async () => {
    const config = field({ enabled: false })
    const wrapper = mountField(config)
    const checkbox = wrapper.findComponent({ name: 'ElCheckbox' })
    await checkbox.vm.$emit('update:modelValue', true)
    await checkbox.vm.$emit('change', true)
    expect(config.enabled).toBe(true)
    expect(wrapper.emitted('change')?.length).toBe(1)
  })

  it('emits change when value changes', async () => {
    const config = field({ enabled: false, value: 4096 })
    const wrapper = mountField(config)
    const inputNumber = wrapper.findComponent({ name: 'ElInputNumber' })
    await inputNumber.vm.$emit('update:modelValue', 8192)
    await inputNumber.vm.$emit('change', 8192)
    expect(config.value).toBe(8192)
    expect(wrapper.emitted('change')?.length).toBe(1)
  })

  it('parses valid raw_json input into an object', async () => {
    const config = field({ key: 'raw.valid', widget: 'raw_json', type: 'object', value: { before: true } })
    const wrapper = mountField(config)
    await wrapper.find('textarea').setValue('{"after":true}')
    expect(config.value).toEqual({ after: true })
    expect(wrapper.emitted('change')?.length).toBe(1)
  })

  it('keeps invalid raw_json input as text without crashing', async () => {
    const config = field({ key: 'raw.invalid', widget: 'raw_json', type: 'object', value: { before: true } })
    const wrapper = mountField(config)
    await wrapper.find('textarea').setValue('{"after":')
    expect(config.value).toBe('{"after":')
    expect(wrapper.emitted('change')?.length).toBe(1)
  })

  it('does not render a public diagnostic badge for ordinary fields', () => {
    const config = field({ key: 'model_runtime.tensor_parallel_size', diagnostic: true })
    const wrapper = mountField(config)
    expect(wrapper.text()).not.toContain('configEdit.badges.diagnostic')
    expect(wrapper.text()).not.toContain('Diagnostic')
    expect(wrapper.text()).not.toContain('诊断')
  })

  it('edits generic array fallback as JSON instead of readonly [] text', async () => {
    const config = field({ key: 'custom.array', type: 'array', widget: 'unknown_widget', value: [] })
    const wrapper = mountField(config)
    const textarea = wrapper.find('textarea')
    expect(textarea.exists()).toBe(true)
    await textarea.setValue('["a","b"]')
    expect(config.value).toEqual(['a', 'b'])
    expect(wrapper.emitted('change')?.length).toBe(1)
  })

  it('edits generic object fallback as JSON', async () => {
    const config = field({ key: 'custom.object', type: 'object', widget: 'unknown_widget', value: {} })
    const wrapper = mountField(config)
    const textarea = wrapper.find('textarea')
    expect(textarea.exists()).toBe(true)
    await textarea.setValue('{"key":"value"}')
    expect(config.value).toEqual({ key: 'value' })
  })

  it('shows fallback help icon for visible fields without explicit help', () => {
    const config = field({ key: 'vendor.future_option', help: undefined })
    const wrapper = mountField(config)
    expect(wrapper.find('.field-help-icon').exists()).toBe(true)
  })
})
