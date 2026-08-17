import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import SettingsPage from './SettingsPage.vue'

vi.stubGlobal('fetch', vi.fn(async () => ({ ok: true, json: async () => ({ items: [] }) })))

describe('SettingsPage', () => {
  it('shows server-side role boundaries', () => {
    const wrapper = mount(SettingsPage)
    expect(wrapper.text()).toContain('安全主管')
    expect(wrapper.text()).toContain('审计查看者')
  })
})
