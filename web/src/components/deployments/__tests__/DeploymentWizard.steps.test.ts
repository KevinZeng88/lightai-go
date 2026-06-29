// @vitest-environment jsdom
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import ElementPlus from 'element-plus'
import DeploymentWizard from '../DeploymentWizard.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'zh-CN',
  fallbackLocale: 'zh-CN',
  messages: {
    'zh-CN': {
      common: { next: '下一步', prev: '上一步', refresh: '刷新', yes: '是', no: '否' },
      deployments: {
        wizardStepModel: '选择模型',
        wizardStepRuntime: '选择运行配置',
        wizardStepService: '服务配置',
        wizardStepOverrides: '参数覆盖',
        wizardStepPreview: '预览运行计划',
        noArtifacts: '暂无模型',
        nodeMismatch: '节点不匹配',
        invalidHostPort: '宿主机端口无效',
        invalidContainerPort: '容器端口无效',
        saveConfig: '保存',
      },
      runnerConfigs: { noConfigs: '暂无配置', showAll: '显示全部', showDeployableOnly: '仅可部署' },
    },
  },
})

function mountWizard() {
  return mount(DeploymentWizard, {
    props: {
      artifacts: [{
        id: 'model-1',
        name: 'model-1',
        locations: [{
          id: 'loc-1',
          node_id: 'node-1',
          verification_status: 'verified',
          match_status: 'exact_match',
        }],
      }],
      nodeRuntimes: [{
        id: 'nbr-1',
        node_id: 'node-1',
        deployable: true,
        status: 'ready',
        config_set: { items: { 'service.container_port': { value: { effective_value: 8000 } } } },
      }],
      modelLocations: [],
    },
    global: {
      plugins: [ElementPlus, i18n],
      stubs: {
        ModelSelector: {
          template: '<button data-testid="select-model" @click="$emit(\'update:model-value\', \'model-1\')">model</button>',
          emits: ['update:model-value'],
        },
        NodeRuntimeSelector: {
          template: '<button data-testid="select-nbr" @click="$emit(\'update:model-value\', \'nbr-1\')">runtime</button>',
          emits: ['update:model-value'],
        },
        DeploymentServiceEditor: {
          template: '<div data-testid="service-editor">service</div>',
          props: ['hostPort', 'containerPort', 'servedModelName'],
        },
        DeploymentOverrideEditor: {
          template: '<div data-testid="override-editor">overrides</div>',
          emits: ['update:overrides', 'update:patch'],
        },
        DeploymentPreviewPanel: {
          template: '<div data-testid="preview-panel">preview</div>',
        },
      },
    },
  })
}

async function clickNext(wrapper: any) {
  const next = wrapper.findAll('button').find((button: any) => button.text().includes('下一步'))
  expect(next).toBeTruthy()
  await next!.trigger('click')
}

describe('DeploymentWizard step transitions', () => {
  it('moves from service config to parameter overrides on Next', async () => {
    const wrapper = mountWizard()
    await wrapper.find('[data-testid="select-model"]').trigger('click')
    await clickNext(wrapper)
    await wrapper.find('[data-testid="select-nbr"]').trigger('click')
    await clickNext(wrapper)
    expect(wrapper.find('[data-testid="service-editor"]').exists()).toBe(true)

    await clickNext(wrapper)

    expect(wrapper.find('[data-testid="override-editor"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="service-editor"]').exists()).toBe(false)
  })
})
