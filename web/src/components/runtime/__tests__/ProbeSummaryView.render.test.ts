// @vitest-environment jsdom
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ProbeSummaryView from '../ProbeSummaryView.vue'

function makeProbeResults(overrides: Record<string, any> = {}) {
  return {
    level1: { image_present: true, source: 'docker_images_list', image_ref: 'vllm/vllm-openai:latest' },
    level2: {
      inspect_success: true,
      image_id: 'sha256:abc123def4567890abcdef1234567890',
      repotags: ['vllm/vllm-openai:latest'],
      env: [
        'PATH=/usr/local/nvidia/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin',
        'LD_LIBRARY_PATH=/usr/local/nvidia/lib:/usr/local/nvidia/lib64',
        'CUDA_VERSION=13.0.2',
        'NVIDIA_REQUIRE_CUDA=cuda>=13.0 brand=unknown,driver>=535,driver<536',
        'NV_CUDA_CUDART_VERSION=13.0.2',
      ],
    },
    level3: { backend_match_status: 'confirmed_match', confirmed_match: true, blocking: false, match_method: 'repo_pattern', match_detail: 'repo tags match pattern vllm for backend vllm' },
    level4: { compatibility_check_status: 'not_run', version_probe_status: 'not_available', blocking: false },
    process_start_detection: { status: 'image_default_selected', confidence: 'high', source: 'backend_profile+image_inspect' },
    ...overrides,
  }
}

describe('ProbeSummaryView', () => {
  it('renders user-facing probe summary by default', () => {
    const wrapper = mount(ProbeSummaryView, {
      props: { probeResults: makeProbeResults(), imageRef: 'vllm/vllm-openai:latest' },
    })
    expect(wrapper.find('[data-testid="probe-summary-view"]').exists()).toBe(true)
    expect(wrapper.html()).toContain('vllm/vllm-openai:latest')
    expect(wrapper.html()).toContain('13.0.2')
    expect(wrapper.find('.el-descriptions').exists()).toBe(true)
  })

  it('keeps raw image env hidden until diagnostics are expanded', () => {
    const wrapper = mount(ProbeSummaryView, {
      props: { probeResults: makeProbeResults(), imageRef: 'vllm/vllm-openai:latest' },
    })
    // Raw probe collapse must exist but collapsed by default.
    const collapse = wrapper.find('[data-testid="raw-probe-collapse"]')
    expect(collapse.exists()).toBe(true)
    // The collapse content should NOT be in the visible DOM (aria-hidden or display:none).
    // Verify via the collapse-item not being active.
    const collapseItem = wrapper.find('.el-collapse-item')
    expect(collapseItem.exists()).toBe(true)
    // In Element Plus, collapsed items have is-active control. Check the collapse isn't showing expanded.
    const rawHtml = collapseItem.attributes('class') || ''
    expect(rawHtml).not.toContain('is-active')
  })

  it('does not show development wording in default summary', () => {
    const wrapper = mount(ProbeSummaryView, {
      props: { probeResults: makeProbeResults() },
    })
    const html = wrapper.html()
    expect(html).not.toContain('deferred to future design')
    expect(html).not.toContain('not yet implemented')
  })

  it('shows empty state when probe data is empty', () => {
    const wrapper = mount(ProbeSummaryView, {
      props: { probeResults: null },
    })
    expect(wrapper.find('[data-testid="probe-summary-empty"]').exists()).toBe(true)
  })

  it('shows empty state when probe data is empty object', () => {
    const wrapper = mount(ProbeSummaryView, {
      props: { probeResults: {} },
    })
    expect(wrapper.find('[data-testid="probe-summary-empty"]').exists()).toBe(true)
  })

  it('shows blocking when level4 blocks', () => {
    const wrapper = mount(ProbeSummaryView, {
      props: { probeResults: makeProbeResults({ level4: { blocking: true } }) },
    })
    // Summary should still render
    expect(wrapper.find('[data-testid="probe-summary-view"]').exists()).toBe(true)
  })
})
