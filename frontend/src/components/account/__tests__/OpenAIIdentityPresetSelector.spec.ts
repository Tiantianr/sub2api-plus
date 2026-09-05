import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import OpenAIIdentityPresetSelector from '@/components/account/OpenAIIdentityPresetSelector.vue'
import {
  DEFAULT_PI_USER_AGENT,
  DEFAULT_CODEX_TUI_USER_AGENT
} from '@/constants/openaiIdentity'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

describe('OpenAIIdentityPresetSelector', () => {
  it('renders default labels when custom labels are not passed', () => {
    const wrapper = mount(OpenAIIdentityPresetSelector, {
      props: {
        modelValue: ''
      }
    })

    const buttons = wrapper.findAll('button')
    expect(buttons).toHaveLength(3)
    expect(buttons[0].text()).toBe('admin.settings.gatewayForwarding.channelDefault')
    expect(buttons[1].text()).toBe('admin.settings.gatewayForwarding.channelPiAgent')
    expect(buttons[2].text()).toBe('admin.settings.gatewayForwarding.channelCodexTui')
  })

  it('renders custom labels when provided', () => {
    const wrapper = mount(OpenAIIdentityPresetSelector, {
      props: {
        modelValue: '',
        defaultLabel: 'Default Channel',
        piLabel: 'Pi Simulation',
        codexTuiLabel: 'Codex TUI'
      }
    })

    const buttons = wrapper.findAll('button')
    expect(buttons[0].text()).toBe('Default Channel')
    expect(buttons[1].text()).toBe('Pi Simulation')
    expect(buttons[2].text()).toBe('Codex TUI')
  })

  it('emits update:modelValue with the exact Pi UA without version when clicking Pi button', async () => {
    const wrapper = mount(OpenAIIdentityPresetSelector, {
      props: {
        modelValue: ''
      }
    })

    const buttons = wrapper.findAll('button')
    await buttons[1].trigger('click')

    expect(wrapper.emitted('update:modelValue')).toBeTruthy()
    expect(wrapper.emitted('update:modelValue')![0]).toEqual([DEFAULT_PI_USER_AGENT])
  })

  it('emits update:modelValue with Codex TUI UA when clicking Codex TUI button', async () => {
    const wrapper = mount(OpenAIIdentityPresetSelector, {
      props: {
        modelValue: ''
      }
    })

    const buttons = wrapper.findAll('button')
    await buttons[2].trigger('click')

    expect(wrapper.emitted('update:modelValue')).toBeTruthy()
    expect(wrapper.emitted('update:modelValue')![0]).toEqual([DEFAULT_CODEX_TUI_USER_AGENT])
  })

  it('emits empty string when clicking Default button', async () => {
    const wrapper = mount(OpenAIIdentityPresetSelector, {
      props: {
        modelValue: DEFAULT_PI_USER_AGENT
      }
    })

    const buttons = wrapper.findAll('button')
    await buttons[0].trigger('click')

    expect(wrapper.emitted('update:modelValue')).toBeTruthy()
    expect(wrapper.emitted('update:modelValue')![0]).toEqual([''])
  })

  it('highlights the active preset button for Pi (including legacy formatted UA)', () => {
    const wrapper = mount(OpenAIIdentityPresetSelector, {
      props: {
        modelValue: 'pi/0.85.0 (darwin 24.1.0; arm64)'
      }
    })

    const buttons = wrapper.findAll('button')
    expect(buttons[1].classes()).toContain('bg-primary-100')
    expect(buttons[0].classes()).not.toContain('bg-primary-100')
    expect(buttons[2].classes()).not.toContain('bg-primary-100')
  })
})
