import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import Analytics from './Analytics.vue'

const mockAIStore = {
  configured: false,
  loading: false,
  checkStatus: vi.fn(),
  getDailyInsights: vi.fn()
}

vi.mock('@/stores/ai', () => ({
  useAIStore: () => mockAIStore
}))

vi.mock('@/api/client', () => ({
  api: {
    getAnalyticsSummary: vi.fn(),
    getAnalyticsTrend: vi.fn(),
    getAnalyticsDistribution: vi.fn(),
    getDailyInsights: vi.fn()
  }
}))

vi.mock('element-plus', () => ({
  ElMessage: { success: vi.fn(), error: vi.fn(), warning: vi.fn() }
}))

const elStubs = {
  'el-button': { template: '<button class="el-button" :disabled="disabled" @click="$emit(\'click\')"><slot/></button>', props: ['disabled', 'type', 'size', 'loading'] },
  'el-icon': { template: '<i class="el-icon"><slot/></i>' },
  'el-select': { template: '<select class="el-select" :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value)"><slot/></select>', props: ['modelValue'] },
  'el-option': { template: '<option :value="value"><slot/></option>', props: ['value', 'label'] },
  'el-card': { template: '<div class="el-card"><slot/></div>' },
  'el-skeleton': { template: '<div class="el-skeleton"/>' },
  'el-empty': { template: '<div class="el-empty"><slot/></div>' }
}

describe('Analytics.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    mockAIStore.configured = false
    mockAIStore.loading = false
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('initial render', () => {
    it('renders analytics page title', async () => {
      const { api } = await import('@/api/client')
      ;(api.getAnalyticsSummary as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { completed_pomodoros: 8, total_focus_time: 7200, completed_tasks: 5, created_tasks: 8 }
      })
      ;(api.getAnalyticsTrend as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { data: [{ date: '2026-05-25', focus_time: 7200, pomodoros: 8 }] }
      })
      ;(api.getAnalyticsDistribution as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { quadrant_stats: {}, task_stats: { total: 10, completed: 5, completion_rate: 0.5 } }
      })

      const wrapper = mount(Analytics, { global: { stubs: elStubs } })
      await new Promise(resolve => setTimeout(resolve, 50))

      expect(wrapper.text()).toContain('数据分析')
    })

    it('renders overview cards section', async () => {
      const { api } = await import('@/api/client')
      ;(api.getAnalyticsSummary as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { completed_pomodoros: 8, total_focus_time: 7200, completed_tasks: 5, created_tasks: 8 }
      })
      ;(api.getAnalyticsTrend as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { data: [] }
      })
      ;(api.getAnalyticsDistribution as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { quadrant_stats: {}, task_stats: { total: 10, completed: 5, completion_rate: 0.5 } }
      })

      const wrapper = mount(Analytics, { global: { stubs: elStubs } })
      await new Promise(resolve => setTimeout(resolve, 50))

      const cards = wrapper.findAll('.overview-card')
      expect(cards.length).toBeGreaterThanOrEqual(4)
    })

    it('renders page structure correctly', async () => {
      const { api } = await import('@/api/client')
      ;(api.getAnalyticsSummary as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { completed_pomodoros: 0, total_focus_time: 0, completed_tasks: 0, created_tasks: 0 }
      })
      ;(api.getAnalyticsTrend as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { data: [] }
      })
      ;(api.getAnalyticsDistribution as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { quadrant_stats: {}, task_stats: { total: 0, completed: 0, completion_rate: 0 } }
      })

      const wrapper = mount(Analytics, { global: { stubs: elStubs } })
      await new Promise(resolve => setTimeout(resolve, 50))

      expect(wrapper.find('.analytics-page').exists()).toBe(true)
      expect(wrapper.find('.page-header').exists()).toBe(true)
    })
  })

  describe('data display', () => {
    it('displays pomodoro count correctly', async () => {
      const { api } = await import('@/api/client')
      ;(api.getAnalyticsSummary as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { completed_pomodoros: 12, total_focus_time: 10800, completed_tasks: 6, created_tasks: 10 }
      })
      ;(api.getAnalyticsTrend as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { data: [] }
      })
      ;(api.getAnalyticsDistribution as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { quadrant_stats: {}, task_stats: { total: 10, completed: 6, completion_rate: 0.6 } }
      })

      const wrapper = mount(Analytics, { global: { stubs: elStubs } })
      await new Promise(resolve => setTimeout(resolve, 50))

      expect(wrapper.text()).toContain('12')
    })

    it('displays completion rate', async () => {
      const { api } = await import('@/api/client')
      ;(api.getAnalyticsSummary as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { completed_pomodoros: 5, total_focus_time: 4500, completed_tasks: 4, created_tasks: 8 }
      })
      ;(api.getAnalyticsTrend as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { data: [] }
      })
      ;(api.getAnalyticsDistribution as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { quadrant_stats: {}, task_stats: { total: 8, completed: 4, completion_rate: 0.5 } }
      })

      const wrapper = mount(Analytics, { global: { stubs: elStubs } })
      await new Promise(resolve => setTimeout(resolve, 50))

      expect(wrapper.text()).toContain('50')
    })
  })

  describe('AI insights section', () => {
    it('renders AI insights section when AI is configured', async () => {
      mockAIStore.configured = true
      const { api } = await import('@/api/client')
      ;(api.getAnalyticsSummary as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { completed_pomodoros: 8, total_focus_time: 7200, completed_tasks: 5, created_tasks: 8 }
      })
      ;(api.getAnalyticsTrend as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { data: [] }
      })
      ;(api.getAnalyticsDistribution as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { quadrant_stats: {}, task_stats: { total: 8, completed: 5, completion_rate: 0.625 } }
      })

      const wrapper = mount(Analytics, { global: { stubs: elStubs } })
      await new Promise(resolve => setTimeout(resolve, 50))

      expect(wrapper.find('.ai-insights-card').exists()).toBe(true)
    })

    it('does not render AI insights when AI is not configured', async () => {
      mockAIStore.configured = false
      const { api } = await import('@/api/client')
      ;(api.getAnalyticsSummary as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { completed_pomodoros: 0, total_focus_time: 0, completed_tasks: 0, created_tasks: 0 }
      })
      ;(api.getAnalyticsTrend as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { data: [] }
      })
      ;(api.getAnalyticsDistribution as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { quadrant_stats: {}, task_stats: { total: 0, completed: 0, completion_rate: 0 } }
      })

      const wrapper = mount(Analytics, { global: { stubs: elStubs } })
      await new Promise(resolve => setTimeout(resolve, 50))

      expect(wrapper.find('.ai-insights-card').exists()).toBe(false)
    })
  })

  describe('error handling', () => {
    it('handles API errors gracefully', async () => {
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
      const { api } = await import('@/api/client')
      ;(api.getAnalyticsSummary as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Network'))
      ;(api.getAnalyticsTrend as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Network'))
      ;(api.getAnalyticsDistribution as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Network'))

      const wrapper = mount(Analytics, { global: { stubs: elStubs } })
      await new Promise(resolve => setTimeout(resolve, 50))

      expect(wrapper.find('.analytics-page, .page-header').exists()).toBe(true)
      consoleSpy.mockRestore()
    })
  })
})
