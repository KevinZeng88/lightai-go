// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import ElementPlus from 'element-plus'
import BackendRuntimesPage from '../BackendRuntimesPage.vue'
import { toRuntimeTemplateDisplay } from '@/utils/runtimeDisplay'

// Mock API modules used by BackendRuntimesPage.vue.
vi.mock('@/api/client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
    defaults: { headers: { common: {} } },
  },
}))

vi.mock('@/api/runtimes', () => ({
  listRuntimes: vi.fn().mockResolvedValue([
    {
      id: 'runtime.vllm.nvidia-docker',
      name: 'runtime.vllm.nvidia-docker',
      display_name: 'vLLM NVIDIA Docker',
      backend_id: 'backend.vllm',
      backend_version_id: 'vllm-v0.23.0',
      vendor: 'nvidia',
      image_ref: 'vllm/vllm-openai:latest',
      is_builtin: true,
      is_editable: false,
      status: 'active',
      visibility: 'visible',
    },
  ]),
}))

vi.mock('@/api/configEdit', () => ({
  getConfigEditView: vi.fn().mockResolvedValue({
    layer: 'backend_runtime',
    object_id: 'runtime.vllm.nvidia-docker',
    object_kind: 'backend_runtime',
    sections: [{ key: 'basic', label: 'Basic', order: 10, fields: [] }],
  }),
  applyConfigEditPatch: vi.fn().mockResolvedValue({}),
}))

// Provide zh-CN i18n for tests.
const i18n = createI18n({ legacy: false, locale: 'zh-CN', fallbackLocale: 'zh-CN', messages: {} })

function mountPage() {
  return mount(BackendRuntimesPage, {
    global: {
      plugins: [createPinia(), i18n, ElementPlus],
      stubs: { RouterLink: true },
    },
  })
}

describe('BackendRuntimesPage integration', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('uses product display name for runtime list entry', () => {
    // Verify the display adapter always produces correct names.
    const row = {
      id: 'runtime.vllm.nvidia-docker',
      name: 'runtime.vllm.nvidia-docker',
      display_name: 'vLLM NVIDIA Docker',
      vendor: 'nvidia',
      backend_id: 'backend.vllm',
      is_editable: false,
      image_ref: 'vllm/vllm-openai:latest',
    }
    const display = toRuntimeTemplateDisplay(row)
    expect(display.displayName).toBe('vLLM NVIDIA Docker')
    expect(display.displayName).not.toBe('runtime.vllm.nvidia-docker')
  })

  it('clone default display name uses product name not raw id', () => {
    const row = {
      id: 'runtime.vllm.nvidia-docker',
      name: 'runtime.vllm.nvidia-docker',
      display_name: 'vLLM NVIDIA Docker',
      vendor: 'nvidia',
      backend_id: 'backend.vllm',
      is_editable: false,
      image_ref: 'vllm/vllm-openai:latest',
    }
    const display = toRuntimeTemplateDisplay(row)
    // Clone suffix logic: displayName + customSuffix
    const cloneDefault = `${display.displayName} - 用户配置`
    expect(cloneDefault).toBe('vLLM NVIDIA Docker - 用户配置')
    expect(cloneDefault).not.toContain('runtime.vllm.nvidia-docker')
  })

  it('version display is wildcard for builtin template', () => {
    const row = {
      id: 'runtime.vllm.nvidia-docker',
      name: 'runtime.vllm.nvidia-docker',
      is_editable: false,
      backend_version_id: 'vllm-v0.23.0',
      vendor: 'nvidia',
      backend_id: 'backend.vllm',
      image_ref: 'vllm/vllm-openai:latest',
    }
    const display = toRuntimeTemplateDisplay(row)
    expect(display.version).toBe('*')
    expect(display.versionDisplay).toBe('*')
  })

  it('version display is wildcard for user config too', () => {
    const row = {
      id: 'user-config-123',
      name: 'user-config-123',
      is_editable: true,
      backend_version_id: 'vllm-v0.23.0',
      vendor: 'nvidia',
      backend_id: 'backend.vllm',
      image_ref: 'vllm/vllm-openai:latest',
    }
    const display = toRuntimeTemplateDisplay(row)
    expect(display.version).toBe('*')
    expect(display.versionDisplay).toBe('*')
  })

  it('page mounts without errors', async () => {
    const wrapper = mountPage()
    await flushPromises()
    expect(wrapper.find('.page-container').exists()).toBe(true)
  })
})
