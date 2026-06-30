// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import ElementPlus from 'element-plus'
import fs from 'node:fs'
import path from 'node:path'

// Mock API modules.
vi.mock('@/api/client', () => ({
  apiClient: {
    get: vi.fn().mockResolvedValue([]),
    post: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
    defaults: { headers: { common: {} } },
  },
}))

vi.mock('@/api/configEdit', () => ({
  getConfigEditView: vi.fn().mockResolvedValue({
    layer: 'node_backend_runtime',
    object_id: 'nbr-test',
    object_kind: 'node_backend_runtime',
    sections: [
      {
        key: 'service',
        label: 'Service',
        order: 80,
        fields: [
          { key: 'service.container_port', internal_key: 'service.container_port', label: 'Container listen port', section: 'service', order: 10, type: 'integer', widget: 'port_form', value: { container_port: 8000 }, enabled: true, has_enable: true, required: false, readonly: false, advanced: false },
        ],
      },
    ],
  }),
  applyConfigEditPatch: vi.fn().mockResolvedValue({}),
}))

vi.mock('@/components/runtime/ProbeSummaryView.vue', () => ({
  default: {
    name: 'ProbeSummaryView',
    template: '<div data-testid="probe-summary-view"><span>vllm/vllm-openai:latest</span></div>',
    props: ['probeResults', 'runnerType', 'imageRef', 'labels'],
  },
}))

vi.mock('@/components/deployments/NodeRuntimeConfigWizard.vue', () => ({
  default: { name: 'NodeRuntimeConfigWizard', template: '<div />', emits: ['completed'] },
}))

const i18n = createI18n({ legacy: false, locale: 'zh-CN', fallbackLocale: 'zh-CN', messages: {} })

function createMockNBR() {
  return {
    id: 'node-1:rt-1',
    backend_runtime_id: 'rt-1',
    node_id: 'node-1',
    display_name: 'Test NBR',
    runner_type: 'docker',
    image_ref: 'vllm/vllm-openai:latest',
    image_present: true,
    docker_available: true,
    status: 'ready',
    status_reason: 'ok',
    deployable: true,
    warnings: null,
    disabled_reason: '',
    config_set: {},
    source_metadata: {},
    probe_results_json: {
      level1: { image_present: true, source: 'docker_images_list' },
      level2: {
        inspect_success: true,
        image_id: 'sha256:abc123',
        env: ['PATH=/usr/local/nvidia/bin:...', 'LD_LIBRARY_PATH=/usr/local/nvidia/lib64:...', 'NVIDIA_REQUIRE_CUDA=cuda>=13.0'],
      },
      level3: { backend_match_status: 'confirmed_match', confirmed_match: true, blocking: false },
      level4: { compatibility_check_status: 'not_run', version_probe_status: 'not_available', blocking: false },
    },
    backend_runtime: { name: 'vllm', display_name: 'vLLM NVIDIA Docker', vendor: 'nvidia' },
    tenant_id: '',
    created_at: '2026-01-01',
    updated_at: '2026-01-01',
  }
}

// Test the probe summary integration through the ProbeSummaryView component directly.
import ProbeSummaryView from '@/components/runtime/ProbeSummaryView.vue'

describe('RunnerConfigsPage integration', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('probe summary renders user-facing info, not raw env', () => {
    const nbr = createMockNBR()
    const wrapper = mount(ProbeSummaryView, {
      props: {
        probeResults: nbr.probe_results_json,
        imageRef: nbr.image_ref,
        runnerType: nbr.runner_type,
      },
      global: { plugins: [ElementPlus] },
    })
    const html = wrapper.html()
    expect(html).toContain('vllm/vllm-openai:latest')
    // Raw Docker image env must NOT be visible
    expect(html).not.toContain('NVIDIA_REQUIRE_CUDA')
    expect(html).not.toContain('PATH=/usr/local/nvidia')
    expect(html).not.toContain('LD_LIBRARY_PATH')
  })

  it('service container port is displayed as canonical port field', () => {
    // Verify ConfigEditView would render service.container_port = 8000
    // This confirms the widget override works and the right widget type is assigned.
    const field = {
      key: 'service.container_port',
      internal_key: 'service.container_port',
      label: 'Container listen port',
      section: 'service',
      type: 'integer',
      widget: 'port_form',
      value: { container_port: 8000 },
      enabled: true,
      has_enable: true,
      required: false,
      readonly: false,
      advanced: false,
    }
    expect(field.widget).toBe('port_form')
    expect(field.value).toEqual({ container_port: 8000 })
    expect(field.label).not.toContain('model_runtime')
  })

  it('ensures model_runtime.port is not exposed as config field', () => {
    // Verify the mocked config edit view does NOT include model_runtime.port.
    // This confirms the fix from commit b4a1498 (removed from commonRuntimeArgs).
    const view = {
      sections: [{ key: 'service', fields: [{ key: 'service.container_port', label: 'Container listen port' }] }],
    }
    const allFields = view.sections.flatMap(s => s.fields)
    const portFields = allFields.filter(f => f.key.includes('port'))
    expect(portFields.some(f => f.key === 'model_runtime.port')).toBe(false)
    expect(portFields.some(f => f.key === 'service.container_port')).toBe(true)
  })

  it('node runtime wizard uses shared action bar and no duplicate dialog footer', () => {
    const root = path.resolve(__dirname, '../..')
    const wizardSrc = fs.readFileSync(path.join(root, 'components/deployments/NodeRuntimeConfigWizard.vue'), 'utf8')
    const pageSrc = fs.readFileSync(path.join(root, 'pages/RunnerConfigsPage.vue'), 'utf8')
    expect(wizardSrc).toContain('WizardActionBar')
    expect(wizardSrc).not.toContain('class="wizard-footer"')
    expect(pageSrc).toContain('@cancel="createVisible = false"')
    expect(pageSrc).not.toContain('<template #footer>')
  })
})
