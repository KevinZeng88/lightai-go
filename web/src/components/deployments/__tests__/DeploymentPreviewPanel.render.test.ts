// @vitest-environment jsdom
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import ElementPlus from 'element-plus'
import DeploymentPreviewPanel from '../DeploymentPreviewPanel.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'zh-CN',
  fallbackLocale: 'zh-CN',
  messages: {
    'zh-CN': {
      common: { yes: '是', no: '否', error: '错误' },
      deployments: {
        previewRunPlan: '预览运行计划',
        canRun: '可运行',
        warnings: '警告',
        lintFindings: 'Lint 检查',
        dockerPreview: 'Docker 命令预览',
        gpuBindingGroup: 'GPU / Vendor 绑定注入',
        acceleratorIds: '加速卡',
        runPlanSourceNote: '来源图',
        finalRunPlan: '最终运行计划',
      },
      status: { warning: '警告', error: '错误', unknown: '未知' },
      runPlan: { lint: { unknown: '未知检查' } },
      preflight: { reason: { unknown: '未知错误' } },
    },
  },
})

function mountPanel(previewData: any, developerMode = false) {
  return mount(DeploymentPreviewPanel, {
    props: { previewData, loading: false, developerMode },
    global: {
      plugins: [ElementPlus, i18n],
      stubs: {
        JsonViewer: { template: '<pre data-testid="json-viewer">{{ JSON.stringify(value) }}</pre>', props: ['value'] },
      },
    },
  })
}

describe('DeploymentPreviewPanel', () => {
  it('renders runnable resolved plan, docker preview, GPU injection, and source map', () => {
    const wrapper = mountPanel({
      can_run: true,
      docker_preview: 'docker run -d --gpus "device=0" -e CUDA_VISIBLE_DEVICES=0 vllm/vllm-openai:latest --model /models/qwen',
      lint: { status: 'ok', findings: [] },
      preflight: { status: 'ok', errors: [], warnings: [] },
      run_plan: {
        device_binding: {
          enabled: true,
          selection_mode: 'auto',
          vendor: 'nvidia',
          gpu_device_ids: ['0'],
          source: 'node_inventory',
          patch_target: 'deployment.placement_json',
          injection_preview: [
            { target: 'docker_options', key: 'docker.gpus', value: 'device=0', source: 'derived', docker_effect: '--gpus', patch_target: 'deployment.placement_json' },
            { target: 'env', key: 'CUDA_VISIBLE_DEVICES', value: '0', source: 'derived', docker_effect: '-e', patch_target: 'deployment.placement_json' },
          ],
        },
        parameter_source_map: {
          image: [{ key: 'launcher.image', target: 'image', effective_source: 'node_backend_runtime', patch_target: 'node_backend_runtime.config_snapshot_json', docker_effect: 'docker image' }],
          system_generated: [{ key: 'docker.gpus', target: 'system_generated', effective_source: 'derived', patch_target: 'deployment.placement_json', docker_effect: '--gpus' }],
        },
      },
    }, true)

    const text = wrapper.text()
    expect(text).toContain('可运行')
    expect(text).toContain('是')
    expect((wrapper.find('textarea').element as HTMLTextAreaElement).value).toContain('docker run -d')
    expect(text).toContain('device=0')
    expect(text).toContain('CUDA_VISIBLE_DEVICES')
    expect(text).toContain('deployment.placement_json')
    expect(text).toContain('launcher.image')
  })

  it('hides raw source map and final run plan in normal mode', () => {
    const wrapper = mountPanel({
      can_run: true,
      docker_preview: 'docker run -d image',
      lint: { status: 'ok', findings: [] },
      preflight: { status: 'ok', errors: [], warnings: [] },
      run_plan: {
        parameter_source_map: {
          image: [{ key: 'launcher.image', target: 'image', effective_source: 'node_backend_runtime', patch_target: 'node_backend_runtime.config_snapshot_json', docker_effect: 'docker image' }],
        },
      },
    })

    const text = wrapper.text()
    expect(text).not.toContain('launcher.image')
    expect(wrapper.find('[data-testid="json-viewer"]').exists()).toBe(false)
  })

  it('dedupes resolve errors and shows message key path source without unknown fallback', () => {
    const issue = {
      code: 'resolve_error',
      message: 'template value was retained only in source map',
      key: 'launcher.command',
      path: ['runplan', 'args'],
      reason: 'final args resolved',
      source: 'runplan_resolver',
      severity: 'warning',
      blocking: false,
    }
    const wrapper = mountPanel({
      can_run: true,
      docker_preview: 'docker run -d image',
      lint: { status: 'ok', findings: [] },
      preflight: { status: 'ok', errors: [], warnings: [issue, { ...issue }] },
      run_plan: {},
    })

    const text = wrapper.text()
    expect((text.match(/\[resolve_error\]/g) || []).length).toBe(1)
    expect(text).toContain('template value was retained only in source map')
    expect(text).toContain('key: launcher.command')
    expect(text).toContain('path: runplan.args')
    expect(text).toContain('source: runplan_resolver')
    expect(text).toContain('blocking: 否')
    expect(text).not.toContain('[resolve_error] 未知错误')
  })
})
