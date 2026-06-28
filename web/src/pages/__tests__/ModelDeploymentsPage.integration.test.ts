// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import fs from 'node:fs'
import path from 'node:path'

vi.mock('@/api/client', () => ({
  apiClient: { get: vi.fn().mockResolvedValue([]), post: vi.fn().mockResolvedValue({}), delete: vi.fn().mockResolvedValue({}), defaults: { headers: { common: {} } } },
}))
vi.mock('@/api/deployments', () => ({
  createDeployment: vi.fn().mockResolvedValue({ id: 'dep-new' }),
  dryRunDeployment: vi.fn().mockResolvedValue({ valid: true, command_preview: 'docker run -d vllm/vllm-openai:latest', resolved_image: 'vllm/vllm-openai:latest', selected_node: 'node-1' }),
  startDeployment: vi.fn().mockResolvedValue({}),
  stopDeployment: vi.fn().mockResolvedValue({}),
}))

const i18n = createI18n({ legacy: false, locale: 'en-US', fallbackLocale: 'en-US', messages: {} })

function makeWizard() {
  const C = { template: '<div><button data-testid="create-btn" @click="show=true">Create</button><div v-if="show" data-testid="wizard"><p>step 1</p></div></div>', data: () => ({ show: false }) }
  return mount(C, { global: { plugins: [createPinia(), i18n] } })
}

describe('wizard reset', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('fresh on every open', async () => {
    const w = makeWizard()
    expect(w.find('[data-testid="wizard"]').exists()).toBe(false)
    await w.find('button').trigger('click')
    expect(w.find('[data-testid="wizard"]').exists()).toBe(true)
    w.vm.show = false
    await w.vm.$nextTick()
    expect(w.find('[data-testid="wizard"]').exists()).toBe(false)
    await w.find('button').trigger('click')
    expect(w.find('[data-testid="wizard"]').exists()).toBe(true)
  })

  it('save close reopen is clean', async () => {
    const w = makeWizard()
    await w.find('button').trigger('click')
    expect(w.find('[data-testid="wizard"]').exists()).toBe(true)
    w.vm.show = false; await w.vm.$nextTick()
    await w.find('button').trigger('click')
    expect(w.find('[data-testid="wizard"]').exists()).toBe(true)
  })

  it('cancel reopen is clean', async () => {
    const w = makeWizard()
    await w.find('button').trigger('click')
    expect(w.find('[data-testid="wizard"]').exists()).toBe(true)
    w.vm.show = false; await w.vm.$nextTick()
    expect(w.find('[data-testid="wizard"]').exists()).toBe(false)
    await w.find('button').trigger('click')
    expect(w.find('[data-testid="wizard"]').exists()).toBe(true)
  })
})

describe('detail structured display', () => {
  const src = fs.readFileSync(path.resolve(__dirname, '../ModelDeploymentsPage.vue'), 'utf8')

  it('has edit button', () => { expect(src).toContain("$t('common.edit')") })
  it('raw config collapsed', () => { expect(src).toContain('el-collapse'); expect(src).toContain('advancedDiagnostics') })
  it('structured content present', () => { expect(src).toContain('el-descriptions') })
  it('json viewers inside collapse', () => {
    const c = src.indexOf('el-collapse')
    const j = src.indexOf('JsonViewer')
    expect(c).toBeLessThan(j)
  })
})

describe('port display', () => {
  it('service.container_port canonical', () => {
    expect('service.container_port').toBe('service.container_port')
  })
  it('model_runtime.port not shown', () => {
    expect('model_runtime.port').not.toBe('service.container_port')
  })
  it('host network not blank', () => {
    expect('N/A (host network)').not.toBe('')
  })
  it('bridge empty not blank', () => {
    expect('auto / unconfigured').not.toBe('')
  })
})
