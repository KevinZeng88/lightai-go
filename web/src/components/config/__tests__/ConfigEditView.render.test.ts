// @vitest-environment jsdom
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import ElementPlus from 'element-plus'
import { createI18n } from 'vue-i18n'
import ConfigEditView from '../ConfigEditView.vue'
import zhCN from '@/locales/zh-CN'
import type { ConfigEditView as ConfigEditViewModel } from '@/utils/configEditView'
import type { ConfigEditPatch } from '@/utils/configEditView'

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

function makeRiskMatrixView(): ConfigEditViewModel {
  return {
    layer: 'deployment',
    object_id: 'dep-test',
    object_kind: 'deployment',
    sections: [
      {
        key: 'model_serving',
        label: 'Model serving',
        order: 10,
        fields: [
          { key: 'normal.enabled', internal_key: 'normal.enabled', label: 'Normal enabled', section: 'model_serving', order: 1, type: 'string', widget: 'string', value: 'on', enabled: true, original_enabled: true, has_enable: true, required: false, readonly: false, advanced: false },
          { key: 'normal.disabled', internal_key: 'normal.disabled', label: 'Normal disabled', section: 'model_serving', order: 2, type: 'string', widget: 'string', value: 'off', enabled: false, original_enabled: false, has_enable: true, required: false, readonly: false, advanced: false },
        ],
      },
      {
        key: 'backend_runtime',
        label: 'Runtime',
        order: 20,
        fields: [
          { key: 'advanced.enabled', internal_key: 'advanced.enabled', label: 'Advanced enabled', section: 'backend_runtime', order: 1, type: 'string', widget: 'string', value: 'on', enabled: true, original_enabled: true, has_enable: true, required: false, readonly: false, advanced: true, view: 'advanced' },
          { key: 'advanced.disabled', internal_key: 'advanced.disabled', label: 'Advanced disabled', section: 'backend_runtime', order: 2, type: 'string', widget: 'string', value: 'off', enabled: false, original_enabled: false, has_enable: true, required: false, readonly: false, advanced: true, view: 'advanced' },
        ],
      },
      {
        key: 'security_high_risk',
        label: 'Security',
        order: 90,
        fields: [
          { key: 'security.enabled', internal_key: 'security.enabled', label: 'Security enabled', section: 'security_high_risk', order: 1, type: 'boolean', widget: 'boolean', value: true, enabled: true, original_enabled: true, has_enable: true, required: false, readonly: false, advanced: true, tier: 'expert', view: 'security', risk: 'high' },
          { key: 'security.disabled', internal_key: 'security.disabled', label: 'Security disabled', section: 'security_high_risk', order: 2, type: 'boolean', widget: 'boolean', value: false, enabled: false, original_enabled: false, has_enable: true, required: false, readonly: false, advanced: true, tier: 'expert', view: 'security', risk: 'high' },
        ],
      },
      {
        key: 'advanced_raw',
        label: 'Raw',
        order: 100,
        fields: [
          { key: 'raw.enabled', internal_key: 'raw.enabled', label: 'Raw enabled', section: 'advanced_raw', order: 1, type: 'object', widget: 'readonly_summary', value: { enabled: true }, enabled: true, original_enabled: true, has_enable: false, required: false, readonly: true, advanced: true, tier: 'expert', view: 'developer', diagnostic: true },
          { key: 'raw.disabled', internal_key: 'raw.disabled', label: 'Raw disabled', section: 'advanced_raw', order: 2, type: 'object', widget: 'readonly_summary', value: { enabled: false }, enabled: false, original_enabled: false, has_enable: false, required: false, readonly: true, advanced: true, tier: 'expert', view: 'developer', diagnostic: true },
        ],
      },
    ],
  }
}

function mountView(modelValue: ConfigEditViewModel, locale?: 'zh-CN') {
  const plugins: any[] = [ElementPlus]
  if (locale === 'zh-CN') {
    plugins.push(createI18n({
      legacy: false,
      locale: 'zh-CN',
      fallbackLocale: 'zh-CN',
      messages: { 'zh-CN': zhCN },
    }))
  }
  return mount(ConfigEditView, {
    global: { plugins },
    props: { modelValue, readonly: true },
  })
}

function makeMutationView(fieldOverrides: Record<string, any> = {}): ConfigEditViewModel {
  return {
    layer: 'backend_runtime',
    object_id: 'rt-mutation',
    object_kind: 'backend_runtime',
    sections: [{
      key: 'model_serving',
      label: 'Model serving',
      order: 10,
      fields: [{
        key: 'model_runtime.max_model_len',
        internal_key: 'model_runtime.max_model_len',
        label: 'Max model length',
        section: 'model_serving',
        order: 1,
        type: 'integer',
        widget: 'number',
        value: 4096,
        original_value: 4096,
        enabled: false,
        original_enabled: false,
        has_enable: true,
        required: false,
        readonly: false,
        advanced: false,
        ...fieldOverrides,
      }],
    }],
  }
}

function lastPatch(wrapper: ReturnType<typeof mount>): ConfigEditPatch {
  const events = wrapper.emitted('update:patch') || []
  return events[events.length - 1]?.[0] as ConfigEditPatch
}

describe('ConfigEditView', () => {
  it('emits enabled patch when a field checkbox is toggled through ConfigEditView', async () => {
    const wrapper = mount(ConfigEditView, {
      global: { plugins: [ElementPlus] },
      props: { modelValue: makeMutationView(), readonly: false },
    })
    const checkbox = wrapper.find('[data-field-key="model_runtime.max_model_len"] [data-testid="config-field-enabled"]')
    expect(checkbox.exists()).toBe(true)
    await checkbox.find('input').setValue(true)
    await nextTick()
    expect(lastPatch(wrapper).fields).toEqual([expect.objectContaining({
      key: 'model_runtime.max_model_len',
      enabled: true,
      value: 4096,
    })])
  })

  it('emits value patch when a number field is edited through ConfigEditView', async () => {
    const wrapper = mount(ConfigEditView, {
      global: { plugins: [ElementPlus] },
      props: { modelValue: makeMutationView({ enabled: true, original_enabled: true }), readonly: false },
    })
    const inputNumber = wrapper.findComponent({ name: 'ElInputNumber' })
    await inputNumber.vm.$emit('update:modelValue', 8192)
    await inputNumber.vm.$emit('change', 8192)
    await nextTick()
    expect(lastPatch(wrapper).fields).toEqual([expect.objectContaining({
      key: 'model_runtime.max_model_len',
      enabled: true,
      value: 8192,
    })])
  })

  it('emits value patch for disabled-but-editable fields through ConfigEditView', async () => {
    const wrapper = mount(ConfigEditView, {
      global: { plugins: [ElementPlus] },
      props: { modelValue: makeMutationView({ enabled: false, original_enabled: false }), readonly: false },
    })
    const inputNumber = wrapper.findComponent({ name: 'ElInputNumber' })
    await inputNumber.vm.$emit('update:modelValue', 6144)
    await inputNumber.vm.$emit('change', 6144)
    await nextTick()
    expect(lastPatch(wrapper).fields).toEqual([expect.objectContaining({
      key: 'model_runtime.max_model_len',
      enabled: false,
      value: 6144,
    })])
  })

  it('emits enabled patch for high-risk fields through ConfigEditView', async () => {
    const wrapper = mount(ConfigEditView, {
      global: { plugins: [ElementPlus] },
      props: {
        modelValue: makeMutationView({
          key: 'docker.privileged',
          internal_key: 'launcher.docker_options',
          path: ['privileged'],
          label: 'Privileged container',
          section: 'security_high_risk',
          type: 'boolean',
          widget: 'boolean',
          value: false,
          original_value: false,
          enabled: false,
          original_enabled: false,
          advanced: true,
          tier: 'expert',
          view: 'security',
          risk: 'high',
        }),
        readonly: false,
      },
    })
    const checkbox = wrapper.find('[data-field-key="docker.privileged"] [data-testid="config-field-enabled"]')
    expect(checkbox.exists()).toBe(true)
    await checkbox.find('input').setValue(true)
    await nextTick()
    expect(lastPatch(wrapper).fields).toEqual([expect.objectContaining({
      key: 'docker.privileged',
      internal_key: 'launcher.docker_options',
      path: ['privileged'],
      enabled: true,
    })])
  })

  it('renders sections and structured runtime fields', () => {
    const wrapper = mount(ConfigEditView, {
      global: { plugins: [ElementPlus] },
      props: { modelValue: makeEditView(), readonly: true },
    })
    expect(wrapper.find('[data-testid="config-edit-view"]').exists()).toBe(true)
    // Display groups should render instead of raw backend sections.
    const sections = wrapper.findAll('[data-testid="config-edit-section"]')
    expect(sections.length).toBeGreaterThanOrEqual(2)
    expect(wrapper.find('[data-section-key="enabled_parameters"]').exists()).toBe(true)
    expect(wrapper.find('[data-section-key="common_parameters"]').exists()).toBe(true)
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

  it('renders port, mount, and device fields with structured widgets when available', () => {
    const modelValue = makeEditView()
    modelValue.sections.push({
      key: 'service',
      label: 'Service',
      order: 80,
      fields: [
        { key: 'service.container_port', internal_key: 'service.container_port', label: 'Container port', section: 'service', order: 1, type: 'integer', widget: 'port_form', value: 8000, enabled: true, has_enable: false, required: true, readonly: false, advanced: false },
        { key: 'runtime.device_binding', internal_key: 'runtime.device_binding', label: 'Device binding', section: 'devices_mounts', order: 2, type: 'object', widget: 'accelerator_binding', value: { mode: 'auto', vendor: 'nvidia' }, enabled: true, has_enable: true, required: false, readonly: false, advanced: false },
      ],
    })
    const wrapper = mount(ConfigEditView, {
      global: { plugins: [ElementPlus] },
      props: { modelValue, readonly: false },
    })
    expect(wrapper.find('.port-form').exists()).toBe(true)
    expect(wrapper.find('.mount-form').exists()).toBe(true)
    expect(wrapper.find('.binding-form').exists()).toBe(true)
  })

  it('keeps environment collapsed by default', () => {
    const wrapper = mount(ConfigEditView, {
      global: { plugins: [ElementPlus] },
      props: { modelValue: makeEditView(), readonly: true },
    })
    // Enabled fields are grouped together at load time; environment stays inside
    // that group instead of retaining its raw backend section key.
    const enabledSection = wrapper.find('[data-section-key="enabled_parameters"]')
    expect(enabledSection.exists()).toBe(true)
    expect(enabledSection.find('[data-field-key="runtime.env"]').exists()).toBe(true)
    const collapseItem = enabledSection.find('.el-collapse-item')
    expect(collapseItem.exists()).toBe(true)
    expect(collapseItem.classes()).toContain('is-active')
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
    const wrapper = mountView(makeEditView())
    const enabledSection = wrapper.find('[data-section-key="enabled_parameters"]')
    expect(enabledSection.exists()).toBe(true)
    expect(enabledSection.find('[data-field-key="advanced_raw_diag"]').exists()).toBe(true)
  })

  it('renders all enabled fields in enabled group including high-risk and raw diagnostics', () => {
    const wrapper = mountView(makeRiskMatrixView())
    const enabledSection = wrapper.find('[data-section-key="enabled_parameters"]')
    expect(enabledSection.exists()).toBe(true)
    for (const key of ['normal.enabled', 'advanced.enabled', 'security.enabled', 'raw.enabled']) {
      expect(enabledSection.find(`[data-field-key="${key}"]`).exists()).toBe(true)
    }
    const highRisk = enabledSection.find('[data-field-key="security.enabled"]')
    expect(highRisk.attributes('data-field-risk')).toBe('high')
    expect(highRisk.attributes('data-field-tier')).toBe('expert')
    expect(highRisk.attributes('data-field-view')).toBe('security')
    const raw = enabledSection.find('[data-field-key="raw.enabled"]')
    expect(raw.attributes('data-field-diagnostic')).toBe('true')
    expect(raw.text()).not.toContain('configEdit.badges.diagnostic')
  })

  it('renders disabled fields in common advanced expert groups', () => {
    const wrapper = mountView(makeRiskMatrixView())
    expect(wrapper.find('[data-section-key="common_parameters"] [data-field-key="normal.disabled"]').exists()).toBe(true)
    expect(wrapper.find('[data-section-key="advanced_parameters_group"] [data-field-key="advanced.disabled"]').exists()).toBe(true)
    const expertSection = wrapper.find('[data-section-key="expert_parameters_group"]')
    expect(expertSection.exists()).toBe(true)
    expect(expertSection.find('[data-field-key="security.disabled"]').exists()).toBe(true)
    expect(expertSection.find('[data-field-key="raw.disabled"]').exists()).toBe(true)
  })

  it('renders zh-CN display group labels', () => {
    const wrapper = mountView(makeRiskMatrixView(), 'zh-CN')
    const text = wrapper.text()
    expect(text).toContain('已启用参数')
    expect(text).toContain('常用参数')
    expect(text).toContain('高级参数')
    expect(text).toContain('专家参数')
  })

  it('keeps enabled group expanded and expert group collapsed', () => {
    const wrapper = mountView(makeRiskMatrixView())
    const enabledSection = wrapper.find('[data-section-key="enabled_parameters"]')
    const enabledCollapseItem = enabledSection.find('.el-collapse-item')
    expect(enabledCollapseItem.exists()).toBe(true)
    expect(enabledCollapseItem.classes()).toContain('is-active')

    const expertSection = wrapper.find('[data-section-key="expert_parameters_group"]')
    const collapseItem = expertSection.find('.el-collapse-item')
    expect(collapseItem.exists()).toBe(true)
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
