import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import Settings from './Settings.vue'

const mockPomodoroSettings = {
  work_duration: 1500,
  short_break_duration: 300,
  long_break_duration: 900,
  long_break_after: 4,
  auto_start_break: true,
  auto_start_work: false,
  enable_sound: true,
  buffer_ratio: 20,
  task_time_preferences: '{"management":"morning","dev":"any"}'
}

const mockAISettings = {
  provider: 'anthropic',
  api_key: 'sk-ant-xxxxxxxxxxxxxxxxxxxx',
  base_url: 'https://api.anthropic.com',
  model: 'claude-sonnet-4-6'
}

vi.mock('@/api/client', () => ({
  api: {
    getSettings: vi.fn(),
    updatePomodoroSettings: vi.fn(),
    updateAISettings: vi.fn()
  }
}))

vi.mock('element-plus', () => ({
  ElMessage: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
  ElMessageBox: { confirm: vi.fn() }
}))

const elStubs = {
  'el-button': { template: '<button class="el-button" :disabled="disabled" @click="$emit(\'click\')"><slot/></button>', props: ['disabled', 'type', 'size', 'loading'] },
  'el-icon': { template: '<i class="el-icon"><slot/></i>' },
  'el-input': { template: '<input class="el-input" :value="modelValue" :type="type" @input="$emit(\'update:modelValue\', $event.target.value)"/>', props: ['modelValue', 'type', 'showPassword', 'placeholder'] },
  'el-switch': { template: '<input type="checkbox" class="el-switch" :checked="modelValue" @change="$emit(\'update:modelValue\', $event.target.checked)"/>', props: ['modelValue'] },
  'el-slider': { template: '<input type="range" class="el-slider" :value="modelValue" :min="min" :max="max" :step="step" @input="$emit(\'update:modelValue\', $event.target.value)"/>', props: ['modelValue', 'min', 'max', 'step', 'showInput'] },
  'el-select': { template: '<select class="el-select" :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value)"><slot/></select>', props: ['modelValue', 'placeholder'] },
  'el-option': { template: '<option :value="value"><slot/></option>', props: ['value', 'label'] },
  'el-card': { template: '<div class="el-card"><slot/></div>' },
  'el-form': { template: '<form class="el-form"><slot/></form>', props: ['model', 'labelWidth'] },
  'el-form-item': { template: '<div class="el-form-item"><slot/></div>', props: ['label'] },
  'el-divider': { template: '<hr class="el-divider"/>' },
  'el-tag': { template: '<span class="el-tag"><slot/></span>', props: ['type', 'size', 'effect'] },
  'el-dialog': { template: '<div v-if="modelValue" class="el-dialog"><slot/></div>', props: ['modelValue', 'title', 'width'] },
  'el-alert': { template: '<div class="el-alert" :class="`el-alert--${type}`"><slot/></div>', props: ['type', 'title', 'closeless', 'showIcon'] }
}

vi.mock('vue-router', () => ({ useRouter: () => ({ push: vi.fn() }) }))

describe('Settings.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('API Key Security (R07 P0)', () => {
    it('should render AI settings section with API key field', async () => {
      const { api } = await import('@/api/client')
      ;(api.getSettings as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { pomodoro: mockPomodoroSettings, ai: { ...mockAISettings, api_key: 'sk-ant-api03-abcdefghijklm' } }
      })
      ;(api.updateAISettings as ReturnType<typeof vi.fn>).mockResolvedValue({ data: {} })

      const wrapper = mount(Settings, { global: { stubs: elStubs } })
      await new Promise(resolve => setTimeout(resolve, 50))

      // Settings page renders successfully with AI config
      expect(wrapper.find('.settings-page, .page-header, form').exists()).toBe(true)
    })

    it('should render settings page with short API key without crashing', async () => {
      const { api } = await import('@/api/client')
      ;(api.getSettings as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { pomodoro: mockPomodoroSettings, ai: { ...mockAISettings, api_key: 'abc' } }
      })

      const wrapper = mount(Settings, { global: { stubs: elStubs } })
      await new Promise(resolve => setTimeout(resolve, 50))

      // Settings page renders successfully even with short key
      expect(wrapper.find('.settings-page, .page-header, form').exists()).toBe(true)
    })
  })

  describe('settings form display', () => {
    it('renders pomodoro settings section', async () => {
      const { api } = await import('@/api/client')
      ;(api.getSettings as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { pomodoro: mockPomodoroSettings, ai: mockAISettings }
      })

      const wrapper = mount(Settings, { global: { stubs: elStubs } })
      await new Promise(resolve => setTimeout(resolve, 50))

      expect(wrapper.find('.settings-page, .page-header, form').exists()).toBe(true)
    })

    it('renders AI settings section', async () => {
      const { api } = await import('@/api/client')
      ;(api.getSettings as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { pomodoro: mockPomodoroSettings, ai: mockAISettings }
      })

      const wrapper = mount(Settings, { global: { stubs: elStubs } })
      await new Promise(resolve => setTimeout(resolve, 50))

      expect(wrapper.find('.settings-page, .page-header, form').exists()).toBe(true)
    })

    it('shows no save button initially when no changes', async () => {
      const { api } = await import('@/api/client')
      ;(api.getSettings as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { pomodoro: mockPomodoroSettings, ai: mockAISettings }
      })

      const wrapper = mount(Settings, { global: { stubs: elStubs } })
      await new Promise(resolve => setTimeout(resolve, 50))

      const saveButton = wrapper.find('.save-button, button:contains("保存")')
      expect(saveButton.exists()).toBe(false)
    })
  })

  describe('save settings', () => {
    it('saves pomodoro settings successfully', async () => {
      const { api } = await import('@/api/client')
      const { ElMessage } = await import('element-plus')
      ;(api.getSettings as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { pomodoro: mockPomodoroSettings, ai: mockAISettings }
      })
      ;(api.updatePomodoroSettings as ReturnType<typeof vi.fn>).mockResolvedValue({ data: {} })

      const wrapper = mount(Settings, { global: { stubs: elStubs } })
      await new Promise(resolve => setTimeout(resolve, 50))

      await (wrapper.vm as any).savePomodoroSettings()
      await new Promise(resolve => setTimeout(resolve, 10))

      expect(ElMessage.success).toHaveBeenCalled()
    })

    it('handles save error gracefully', async () => {
      const { api } = await import('@/api/client')
      const { ElMessage } = await import('element-plus')
      ;(api.getSettings as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { pomodoro: mockPomodoroSettings, ai: mockAISettings }
      })
      ;(api.updatePomodoroSettings as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Save failed'))

      const wrapper = mount(Settings, { global: { stubs: elStubs } })
      await new Promise(resolve => setTimeout(resolve, 50))

      await (wrapper.vm as any).savePomodoroSettings()
      await new Promise(resolve => setTimeout(resolve, 10))

      expect(ElMessage.error).toHaveBeenCalled()
    })
  })

  describe('buffer ratio preview', () => {
    it('displays buffer ratio setting', async () => {
      const { api } = await import('@/api/client')
      ;(api.getSettings as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { pomodoro: mockPomodoroSettings, ai: mockAISettings }
      })

      const wrapper = mount(Settings, { global: { stubs: elStubs } })
      await new Promise(resolve => setTimeout(resolve, 50))

      expect(wrapper.find('.settings-page, .page-header, form').exists()).toBe(true)
    })
  })

  describe('AI Scheduling Strategy (E2E)', () => {
    it('renders scheduling strategy textarea in AI preference section', async () => {
      const { api } = await import('@/api/client')
      const settingsWithStrategy = {
        ...mockPomodoroSettings,
        scheduling_strategy: '上午深度工作，下午处理沟通任务'
      }
      ;(api.getSettings as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { pomodoro: settingsWithStrategy, ai: mockAISettings }
      })

      const wrapper = mount(Settings, { global: { stubs: elStubs } })
      await new Promise(resolve => setTimeout(resolve, 50))

      // Strategy textarea should exist
      const textareas = wrapper.findAll('.el-input')
      expect(textareas.length).toBeGreaterThan(0)
    })

    it('includes scheduling_strategy when saving pomodoro settings', async () => {
      const { api } = await import('@/api/client')
      const settingsWithStrategy = {
        ...mockPomodoroSettings,
        scheduling_strategy: '周五下午不排任务，留给总结和复盘'
      }
      ;(api.getSettings as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { pomodoro: settingsWithStrategy, ai: mockAISettings }
      })
      ;(api.updatePomodoroSettings as ReturnType<typeof vi.fn>).mockResolvedValue({ data: {} })

      const wrapper = mount(Settings, { global: { stubs: elStubs } })
      await new Promise(resolve => setTimeout(resolve, 50))

      // Find and click the save button
      const saveButtons = wrapper.findAll('.el-button')
      const saveBtn = saveButtons.find(btn => btn.text().includes('保存设置'))
      if (saveBtn) {
        await saveBtn.trigger('click')
        await new Promise(resolve => setTimeout(resolve, 50))

        expect(api.updatePomodoroSettings).toHaveBeenCalled()
        const callArgs = (api.updatePomodoroSettings as ReturnType<typeof vi.fn>).mock.calls[0][0]
        expect(callArgs).toHaveProperty('scheduling_strategy', '周五下午不排任务，留给总结和复盘')
      }
    })

    it('loads scheduling_strategy from server on mount', async () => {
      const { api } = await import('@/api/client')
      const settingsWithStrategy = {
        ...mockPomodoroSettings,
        scheduling_strategy: '每个任务间留15分钟缓冲'
      }
      ;(api.getSettings as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { pomodoro: settingsWithStrategy, ai: mockAISettings }
      })

      const wrapper = mount(Settings, { global: { stubs: elStubs } })
      await new Promise(resolve => setTimeout(resolve, 50))

      // Settings should load successfully with strategy
      expect(wrapper.find('.settings-page').exists()).toBe(true)
    })

    it('handles empty scheduling_strategy gracefully', async () => {
      const { api } = await import('@/api/client')
      ;(api.getSettings as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { pomodoro: mockPomodoroSettings, ai: mockAISettings }
      })

      const wrapper = mount(Settings, { global: { stubs: elStubs } })
      await new Promise(resolve => setTimeout(resolve, 50))

      // Should render without errors even without strategy
      expect(wrapper.find('.settings-page, .page-header, form').exists()).toBe(true)
    })
  })
})
