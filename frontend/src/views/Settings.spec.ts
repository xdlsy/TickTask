import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import Settings from './Settings.vue'

// Auto-mock the api client — individual tests configure the spies they need.
// We expose `api.agent` as a plain object so tests can stub `.test` / `.status`
// without needing the full client surface.
vi.mock('@/api/client', () => {
  const api: any = {
    getSettings: vi.fn(),
    updatePomodoroSettings: vi.fn(),
    updateAISettings: vi.fn(),
    previewImport: vi.fn(),
    applyImport: vi.fn(),
    clearAll: vi.fn(),
    agent: {
      status: vi.fn().mockResolvedValue({ data: { configured: false } }),
      test: vi.fn().mockResolvedValue({ data: { ok: true, provider: 'openai' } }),
    },
  }
  return { api }
})

vi.mock('element-plus', () => ({
  ElMessage: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
  ElMessageBox: { confirm: vi.fn(), prompt: vi.fn() },
}))

// Stub ImportWizard so the mount doesn't pull in the import flow.
vi.mock('@/components/settings/ImportWizard.vue', () => ({
  default: { name: 'ImportWizard', template: '<div />' },
}))

const mockSettingsResponse = (overrides: Partial<any> = {}) => ({
  data: {
    pomodoro: {
      work_duration: 1500,
      short_break_duration: 300,
      long_break_duration: 900,
      long_break_after: 4,
      auto_start_break: false,
      auto_start_work: false,
      enable_sound: true,
      buffer_ratio: 20,
      task_time_preferences: '{"management":"any","dev":"any"}',
    },
    ai: {
      provider: 'openai',
      base_url: 'https://api.openai.com/v1',
      model: 'gpt-4o-mini',
      api_key_set: true,
      api_key_preview: 'sk-ab****wxyz',
      ...overrides,
    },
  },
})

const elStubs = {
  'el-button': { template: '<button class="el-button"><slot/></button>', props: ['disabled', 'type', 'size', 'loading'] },
  'el-input': { template: '<input class="el-input" :value="modelValue" :type="type" :placeholder="placeholder" @input="$emit(\'update:modelValue\', $event.target.value)"/>', props: ['modelValue', 'type', 'showPassword', 'placeholder', 'size'] },
  'el-input-number': { template: '<input class="el-input-number" :value="modelValue"/>', props: ['modelValue', 'min', 'max', 'size'] },
  'el-select': { template: '<select class="el-select" :value="modelValue"><slot/></select>', props: ['modelValue', 'placeholder', 'size'] },
  'el-option': { template: '<option :value="value"><slot/></option>', props: ['value', 'label'] },
  'el-switch': { template: '<input type="checkbox" :checked="modelValue"/>', props: ['modelValue', 'size'] },
  'el-slider': { template: '<input type="range" :value="modelValue"/>', props: ['modelValue', 'min', 'max', 'step'] },
  'el-tag': { template: '<span class="el-tag"><slot/></span>', props: ['type', 'size', 'effect'] },
}

describe('Settings.vue — API Key input', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('loads settings and leaves the api_key input empty (with preview placeholder)', async () => {
    const { api } = await import('@/api/client')
    ;(api.getSettings as any).mockResolvedValue(mockSettingsResponse())

    const wrapper = mount(Settings, { global: { stubs: elStubs } })
    await flushPromises()

    // The api_key ref should be empty after load — preview is shown as
    // placeholder, NOT as the input value.
    expect((wrapper.vm as any).aiSettings.api_key).toBe('')
    expect((wrapper.vm as any).aiSettingsPreview).toBe('sk-ab****wxyz')
  })

  it('saves AI settings with empty api_key — backend preserves the existing key', async () => {
    const { api } = await import('@/api/client')
    ;(api.getSettings as any).mockResolvedValue(mockSettingsResponse())
    ;(api.updateAISettings as any).mockResolvedValue({ data: {} })

    const wrapper = mount(Settings, { global: { stubs: elStubs } })
    await flushPromises()

    // Change only the model, leave api_key empty.
    ;(wrapper.vm as any).aiSettings.model = 'gpt-4o'
    await (wrapper.vm as any).saveAISettings()
    await flushPromises()

    const sentBody = (api.updateAISettings as any).mock.calls[0][0]
    expect(sentBody.api_key).toBe('')
    expect(sentBody.model).toBe('gpt-4o')
  })

  it('testAIConnection sends form values via api.agent.test body without saving first', async () => {
    const { api } = await import('@/api/client')
    ;(api.getSettings as any).mockResolvedValue(mockSettingsResponse())
    ;(api.agent.test as any).mockResolvedValue({ data: { ok: true, provider: 'openai' } })

    const wrapper = mount(Settings, { global: { stubs: elStubs } })
    await flushPromises()

    ;(wrapper.vm as any).aiSettings.api_key = 'sk-typed-in-form'
    await (wrapper.vm as any).testAIConnection()
    await flushPromises()

    // Critical: must NOT call updateAISettings before testing — the whole
    // point is to allow testing a candidate key without committing it.
    expect(api.updateAISettings).not.toHaveBeenCalled()
    expect(api.agent.test).toHaveBeenCalledWith(
      expect.objectContaining({ api_key: 'sk-typed-in-form' }),
    )
  })

  it('testAIConnection with no typed key sends empty body to test the saved config', async () => {
    const { api } = await import('@/api/client')
    ;(api.getSettings as any).mockResolvedValue(mockSettingsResponse())

    const wrapper = mount(Settings, { global: { stubs: elStubs } })
    await flushPromises()

    // Fresh page load: form api_key is empty (GET /settings returns only the
    // masked preview), but a key IS stored.
    expect((wrapper.vm as any).aiSettings.api_key).toBe('')
    expect((wrapper.vm as any).aiSettingsPreview).toBe('sk-ab****wxyz')

    await (wrapper.vm as any).testAIConnection()
    await flushPromises()

    // No typed key → send an EMPTY body so the backend tests the SAVED config
    // via its nil-settings path. Sending the partial form (empty api_key)
    // would make the backend reject it as "AI 未配置".
    expect(api.agent.test).toHaveBeenCalledWith({})
  })
})

describe('Settings.vue — vendor presets', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('handleProviderChange fills baseURL + default model for a known vendor', async () => {
    const { api } = await import('@/api/client')
    ;(api.getSettings as any).mockResolvedValue(mockSettingsResponse())

    const wrapper = mount(Settings, { global: { stubs: elStubs } })
    await flushPromises()

    ;(wrapper.vm as any).aiSettings.provider = 'minimax'
    ;(wrapper.vm as any).handleProviderChange()

    expect((wrapper.vm as any).aiSettings.base_url).toBe('https://api.minimaxi.com/anthropic')
    expect((wrapper.vm as any).aiSettings.model).toBe('MiniMax-M3')
    expect((wrapper.vm as any).availableModels).toEqual([
      'MiniMax-M3',
      'MiniMax-M2.7',
      'MiniMax-M2.7-highspeed',
      'MiniMax-M2.5',
    ])
  })

  it('handleProviderChange clears baseURL + model for custom', async () => {
    const { api } = await import('@/api/client')
    ;(api.getSettings as any).mockResolvedValue(mockSettingsResponse())

    const wrapper = mount(Settings, { global: { stubs: elStubs } })
    await flushPromises()

    ;(wrapper.vm as any).aiSettings.provider = 'custom'
    ;(wrapper.vm as any).handleProviderChange()

    expect((wrapper.vm as any).aiSettings.base_url).toBe('')
    expect((wrapper.vm as any).aiSettings.model).toBe('')
    expect((wrapper.vm as any).availableModels).toEqual([])
  })

  it('renders the vendor presets as provider options', async () => {
    const { api } = await import('@/api/client')
    ;(api.getSettings as any).mockResolvedValue(mockSettingsResponse())

    const wrapper = mount(Settings, { global: { stubs: elStubs } })
    await flushPromises()

    // The el-option stub exposes the id via the <option> value attribute (the
    // `label` prop is not rendered as slot text by this stub). Vendor ids are
    // unique to the provider dropdown, so membership is a robust check.
    const values = wrapper.findAll('option').map((o) => o.attributes('value'))
    for (const expected of [
      'openai',
      'anthropic',
      'deepseek',
      'qwen',
      'zhipu',
      'moonshot',
      'minimax',
      'custom',
    ]) {
      expect(values).toContain(expected)
    }
  })
})
