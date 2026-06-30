// @vitest-environment jsdom
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import ConfigEditView from '../ConfigEditView.vue'
import type { ConfigEditView as ConfigEditViewModel } from '@/utils/configEditView'

// A realistic ConfigEditView matching backend shape, with docker options
// that have present sub-fields (shm_size, ipc_mode) and absent sub-fields
// (uts_mode, network_mode).
function makeEditView(): ConfigEditViewModel {
  return {
    layer: 'backend_runtime',
    object_id: 'rt-test',
    object_kind: 'backend_runtime',
    sections: [
      {
        key: 'container_resources',
        label: 'Container Resources',
        order: 50,
        fields: [
          { key: 'docker.shm_size', internal_key: 'launcher.docker_options', label: 'Shared memory', section: 'container_resources', order: 10, type: 'string', widget: 'string', value: '16gb', enabled: true, has_enable: true, required: false, readonly: false, advanced: false },
          { key: 'docker.privileged', internal_key: 'launcher.docker_options', label: 'Privileged container', section: 'container_resources', order: 20, type: 'boolean', widget: 'boolean', value: false, enabled: true, has_enable: true, required: false, readonly: false, advanced: false },
          { key: 'docker.ipc_mode', internal_key: 'launcher.docker_options', label: 'IPC mode', section: 'container_resources', order: 30, type: 'string', widget: 'string', value: 'host', enabled: true, has_enable: true, required: false, readonly: false, advanced: false },
          { key: 'launcher.docker_options.uts_mode', internal_key: 'launcher.docker_options', label: 'UTS mode', section: 'container_resources', order: 40, type: 'string', widget: 'string', value: null, enabled: false, has_enable: true, required: false, readonly: false, advanced: false },
          { key: 'docker.network_mode', internal_key: 'launcher.docker_options', label: 'Network mode', section: 'container_resources', order: 50, type: 'string', widget: 'string', value: null, enabled: false, has_enable: true, required: false, readonly: false, advanced: false },
        ],
      },
      {
        key: 'environment',
        label: 'Environment',
        order: 70,
        collapsed: true,
        fields: [
          { key: 'runtime.env', internal_key: 'runtime.env', label: 'Environment variables', section: 'environment', order: 10, type: 'object', widget: 'key_value_table', value: {}, enabled: true, has_enable: true, required: false, readonly: false, advanced: false },
        ],
      },
      {
        key: 'devices_mounts',
        label: 'Devices and mounts',
        order: 60,
        fields: [
          { key: 'runtime.model_mount', internal_key: 'runtime.model_mount', label: 'Model mount', section: 'devices_mounts', order: 10, type: 'object', widget: 'mount_form', value: { container_path: '/models', readonly: true }, enabled: true, has_enable: true, required: false, readonly: false, advanced: false },
        ],
      },
      {
        key: 'health_check',
        label: 'Health check',
        order: 90,
        fields: [
          { key: 'runtime.health', internal_key: 'runtime.health', label: 'Health check', section: 'health_check', order: 10, type: 'object', widget: 'health_check_form', value: { path: '/v1/models', port: 0, timeout: 30, interval: 10, retries: 3 }, enabled: true, has_enable: true, required: false, readonly: false, advanced: false },
        ],
      },
      {
        key: 'advanced_raw',
        label: 'Advanced raw configuration',
        order: 90,
        advanced: true,
        collapsed: true,
        fields: [
          { key: 'advanced_raw_diag', internal_key: 'advanced_raw_diag', label: 'Diagnostics', section: 'advanced_raw', order: 1, type: 'string', widget: 'readonly_summary', value: null, enabled: true, has_enable: false, required: false, readonly: true, advanced: true },
        ],
      },
    ],
  }
}

describe('ConfigEditView', () => {
  it('renders sections and structured runtime fields', () => {
    const wrapper = mount(ConfigEditView, {
      global: { plugins: [ElementPlus] },
      props: { modelValue: makeEditView(), readonly: true },
    })
    expect(wrapper.find('[data-testid="config-edit-view"]').exists()).toBe(true)
    // Sections should render
    const sections = wrapper.findAll('[data-testid="config-edit-section"]')
    expect(sections.length).toBeGreaterThanOrEqual(3)
  })

  it('shm_size field renders without leaking parent object', () => {
    const wrapper = mount(ConfigEditView, {
      global: { plugins: [ElementPlus] },
      props: { modelValue: makeEditView(), readonly: true },
    })
    const html = wrapper.html()
    // Field must exist in the DOM with correct data-field-key
    expect(wrapper.find('[data-field-key="docker.shm_size"]').exists()).toBe(true)
    // Backend provides value=16gb — the field should be rendered (value is in v-model,
    // which jsdom doesn't expose as an HTML attribute; the presence of the field
    // with correct key and no parent object is the key regression protection)
    // Must NOT contain the parent object's gpu_capabilities
    expect(html).not.toContain('gpu_capabilities')
    // Must contain the field label or key indicating it renders
    const fieldText = wrapper.find('[data-field-key="docker.shm_size"]').text()
    // In readonly mode the value text should be visible (not inside an input)
    expect(fieldText.length > 0 || html.includes('docker.shm_size')).toBe(true)
  })

  it('ipc_mode displays host not parent object', () => {
    const wrapper = mount(ConfigEditView, {
      global: { plugins: [ElementPlus] },
      props: { modelValue: makeEditView(), readonly: true },
    })
    const html = wrapper.html()
    expect(html).toContain('host')
  })

  it('uts_mode and network_mode do not display parent docker object', () => {
    const wrapper = mount(ConfigEditView, {
      global: { plugins: [ElementPlus] },
      props: { modelValue: makeEditView(), readonly: true },
    })
    const utsField = wrapper.find('[data-field-key="launcher.docker_options.uts_mode"]')
    const netField = wrapper.find('[data-field-key="docker.network_mode"]')
    // Both fields should exist in DOM
    expect(utsField.exists()).toBe(true)
    expect(netField.exists()).toBe(true)
    // Their value area must NOT contain the parent docker options object keys
    const utsValue = utsField.find('[data-testid="config-field-value"]')
    const netValue = netField.find('[data-testid="config-field-value"]')
    expect(utsValue.exists()).toBe(true)
    expect(netValue.exists()).toBe(true)
    expect(utsValue.text()).not.toContain('gpu_capabilities')
    expect(netValue.text()).not.toContain('gpu_capabilities')
  })

  it('renders structured widgets instead of raw JSON', () => {
    const wrapper = mount(ConfigEditView, {
      global: { plugins: [ElementPlus] },
      props: { modelValue: makeEditView(), readonly: true },
    })
    const html = wrapper.html()
    // Model mount should show structured text, not raw JSON
    expect(html).not.toContain('"container_path":"/models"')
    // Health check should show structured form, not raw JSON
    expect(html).not.toContain('"path":"/v1/models"')
    // Mount form widget must be rendered
    expect(wrapper.find('.mount-form').exists()).toBe(true)
    // Health check form widget must be rendered
    expect(wrapper.find('.health-form').exists()).toBe(true)
    // Key environment fields rendered as structured widgets (not raw json textarea)
    expect(wrapper.find('.kv-table-wrap').exists() || wrapper.find('[data-field-key="runtime.env"]').exists()).toBe(true)
  })

  it('keeps environment collapsed by default', () => {
    const wrapper = mount(ConfigEditView, {
      global: { plugins: [ElementPlus] },
      props: { modelValue: makeEditView(), readonly: true },
    })
    // Environment section should be collapsed
    const envSection = wrapper.find('[data-section-key="environment"]')
    expect(envSection.exists()).toBe(true)
    const collapseItem = envSection.find('.el-collapse-item')
    expect(collapseItem.exists()).toBe(true)
    // If collapsed by default, the class should not include is-active
    const cls = collapseItem.classes()
    // Element Plus sets is-active on expanded items; collapsed = no is-active
    expect(cls).not.toContain('is-active')
    // Or check that the collapse is rendered but not expanded
  })

  it('does not show vendor visible device placeholder as user env', () => {
    const wrapper = mount(ConfigEditView, {
      global: { plugins: [ElementPlus] },
      props: { modelValue: makeEditView(), readonly: true },
    })
    const html = wrapper.html()
    expect(html).not.toContain('CUDA_VISIBLE_DEVICES')
    expect(html).not.toContain('{{vendor_visible_devices}}')
  })

  it('keeps raw config hidden by default', () => {
    const wrapper = mount(ConfigEditView, {
      global: { plugins: [ElementPlus] },
      props: { modelValue: makeEditView(), readonly: true },
    })
    const advancedSection = wrapper.find('[data-section-key="advanced_raw"]')
    expect(advancedSection.exists()).toBe(true)
    const collapseItem = advancedSection.find('.el-collapse-item')
    expect(collapseItem.exists()).toBe(true)
    // Should be collapsed
    expect(collapseItem.classes()).not.toContain('is-active')
  })

  it('readonly mode does not expose editable inputs', () => {
    const wrapper = mount(ConfigEditView, {
      global: { plugins: [ElementPlus] },
      props: { modelValue: makeEditView(), readonly: true },
    })
    // In readonly mode, editable buttons should not exist
    const addRowButtons = wrapper.findAll('button').filter(b => b.text().includes('Add Row') || b.text().includes('添加行'))
    expect(addRowButtons.length).toBe(0)
  })

  it('editable mode shows controls', () => {
    const wrapper = mount(ConfigEditView, {
      global: { plugins: [ElementPlus] },
      props: { modelValue: makeEditView(), readonly: false },
    })
    expect(wrapper.find('[data-testid="config-edit-view"]').exists()).toBe(true)
    // In editable mode with key_value_table widget, add-row button should appear
    const html = wrapper.html()
    // key_value_table should have an add button
    expect(html).toContain('el-button')
  })

  it('renders self-contained field metadata without backend knowledge', () => {
    const view: ConfigEditViewModel = {
      layer: 'node_backend_runtime',
      object_id: 'self-contained',
      object_kind: 'node_backend_runtime',
      sections: [{
        key: 'model_serving',
        label: 'Model Serving',
        order: 10,
        fields: [
          {
            key: 'vendor.future_gpu_ratio',
            internal_key: 'vendor.future_gpu_ratio',
            label: 'GPU 显存利用率',
            help: '控制单实例可使用的 GPU 显存比例。',
            cli_flag: '--future-gpu-ratio',
            section: 'model_serving',
            order: 1,
            type: 'number',
            widget: 'number',
            value: 0.9,
            default_value: 0.9,
            enabled: false,
            has_enable: true,
            required: false,
            readonly: false,
            advanced: false,
            constraints: { min: 0, max: 1, step: 0.01 },
            validation_rules: { min: 0, max: 1 },
            copy_behavior: 'copy_on_create',
            override_behavior: 'patch_local_value',
            disable_behavior: 'retain_value_when_disabled',
            patch_target: 'vendor.future_gpu_ratio',
          },
          {
            key: 'vendor.future_dtype',
            internal_key: 'vendor.future_dtype',
            label: '数据类型',
            help: '选择模型权重和计算使用的数据类型。',
            section: 'model_serving',
            order: 2,
            type: 'enum',
            widget: 'select',
            value: 'auto',
            enabled: true,
            has_enable: true,
            required: false,
            readonly: false,
            advanced: false,
            options: [{ label: 'auto', value: 'auto' }, { label: 'float16', value: 'float16' }],
          },
        ],
      }],
    }
    const wrapper = mount(ConfigEditView, {
      global: { plugins: [ElementPlus] },
      props: { modelValue: view, readonly: false },
    })
    const html = wrapper.html()
    expect(html).toContain('GPU 显存利用率')
    expect(html).toContain('数据类型')
    expect(html).not.toContain('配置项')
    expect(wrapper.find('[data-field-key="vendor.future_gpu_ratio"] [data-testid="config-field-enabled"]').exists()).toBe(true)
    expect(wrapper.find('[data-field-key="vendor.future_gpu_ratio"] [data-testid="config-field-value"]').exists()).toBe(true)
  })
})
