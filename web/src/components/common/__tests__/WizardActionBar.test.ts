// @vitest-environment jsdom
import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import WizardActionBar from '../WizardActionBar.vue'

describe('WizardActionBar', () => {
  it('renders sticky layout and disabled reason', () => {
    const wrapper = mount(WizardActionBar, {
      global: { plugins: [ElementPlus] },
      props: {
        activeStep: 1,
        totalSteps: 3,
        canNext: false,
        primaryLabel: 'Next',
        nextDisabledReason: 'Select an item',
      },
    })
    expect(wrapper.find('[data-testid="wizard-action-bar"]').classes()).toContain('layout-sticky-top')
    expect(wrapper.find('[data-testid="wizard-action-bar"]').classes()).toContain('is-sticky')
    expect(wrapper.find('[data-testid="wizard-disabled-reason"]').text()).toBe('Select an item')
  })

  it('disables primary action while loading', () => {
    const wrapper = mount(WizardActionBar, {
      global: { plugins: [ElementPlus] },
      props: {
        activeStep: 2,
        totalSteps: 3,
        canNext: true,
        primaryLabel: 'Save',
        primaryLoading: true,
      },
    })
    expect(wrapper.html()).toContain('Save')
    expect(wrapper.find('[data-testid="wizard-action-bar"]').exists()).toBe(true)
  })

  it('renders secondary actions for final wizard steps', () => {
    const wrapper = mount(WizardActionBar, {
      global: { plugins: [ElementPlus] },
      props: {
        activeStep: 2,
        totalSteps: 3,
        primaryLabel: 'Save and Check',
        secondaryActions: [{ key: 'save-only', label: 'Save Only' }],
      },
    })
    expect(wrapper.text()).toContain('Save Only')
    expect(wrapper.text()).toContain('Save and Check')
  })
})
