import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import TimerDisplay from './TimerDisplay.vue'

const mockTimerStore = {
  currentSession: null as { task_id?: string } | null,
  isRunning: false,
  isPaused: false,
  remainingTime: 0,
  percentage: 0,
  currentMode: 'work' as string
}

const mockTaskStore = {
  tasks: [] as Array<{ id: string; title: string }>
}

vi.mock('@/stores/timer', () => ({
  useTimerStore: () => mockTimerStore
}))

vi.mock('@/stores/task', () => ({
  useTaskStore: () => mockTaskStore
}))

vi.mock('@/utils/time', () => ({
  formatTime: vi.fn((s: number) => `00:${String(s).padStart(2, '0')}`)
}))

describe('TimerDisplay', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockTimerStore.currentSession = null
    mockTimerStore.isRunning = false
    mockTimerStore.isPaused = false
    mockTimerStore.remainingTime = 0
    mockTimerStore.percentage = 0
    mockTimerStore.currentMode = 'work'
    mockTaskStore.tasks = []
  })

  describe('rendering', () => {
    it('renders formatted time', () => {
      mockTimerStore.remainingTime = 1500

      const wrapper = mount(TimerDisplay, { props: { size: 200 } })

      expect(wrapper.find('.timer-time').text()).toBe('00:1500')
    })

    it('renders default label when no session', () => {
      const wrapper = mount(TimerDisplay, { props: { size: 200 } })

      expect(wrapper.find('.timer-label').text()).toBe('准备开始')
    })

    it('renders running label', () => {
      mockTimerStore.isRunning = true
      mockTimerStore.currentSession = { task_id: 't1' }

      const wrapper = mount(TimerDisplay, { props: { size: 200 } })

      expect(wrapper.find('.timer-label').text()).toBe('专注中...')
    })

    it('renders paused label', () => {
      mockTimerStore.isPaused = true
      mockTimerStore.currentSession = { task_id: 't1' }

      const wrapper = mount(TimerDisplay, { props: { size: 200 } })

      expect(wrapper.find('.timer-label').text()).toBe('已暂停')
    })

    it('renders timer label for session not running or paused', () => {
      mockTimerStore.currentSession = { task_id: 't1' }

      const wrapper = mount(TimerDisplay, { props: { size: 200 } })

      expect(wrapper.find('.timer-label').text()).toBe('计时器')
    })
  })

  describe('currentTask', () => {
    it('renders task name when current session has matching task', () => {
      mockTimerStore.currentSession = { task_id: 'task-1' }
      mockTaskStore.tasks = [
        { id: 'task-1', title: '设计文档' },
        { id: 'task-2', title: '代码审查' }
      ]

      const wrapper = mount(TimerDisplay, { props: { size: 200 } })

      expect(wrapper.find('.task-name').text()).toBe('设计文档')
    })

    it('does not render task name when no matching task', () => {
      mockTimerStore.currentSession = { task_id: 'task-99' }
      mockTaskStore.tasks = [{ id: 'task-1', title: '设计文档' }]

      const wrapper = mount(TimerDisplay, { props: { size: 200 } })

      expect(wrapper.find('.task-name').exists()).toBe(false)
    })

    it('does not render task name when no session', () => {
      mockTaskStore.tasks = [{ id: 'task-1', title: '设计文档' }]

      const wrapper = mount(TimerDisplay, { props: { size: 200 } })

      expect(wrapper.find('.task-name').exists()).toBe(false)
    })
  })

  describe('size prop', () => {
    it('uses default size 260', () => {
      const wrapper = mount(TimerDisplay)

      const circle = wrapper.find('.timer-circle')
      expect(circle.attributes('style')).toContain('width: 260px')
      expect(circle.attributes('style')).toContain('height: 260px')
    })

    it('applies custom size', () => {
      const wrapper = mount(TimerDisplay, { props: { size: 320 } })

      const circle = wrapper.find('.timer-circle')
      expect(circle.attributes('style')).toContain('width: 320px')
      expect(circle.attributes('style')).toContain('height: 320px')
    })
  })

  describe('computed properties', () => {
    it('computes radius from size', () => {
      mockTimerStore.remainingTime = 300
      const wrapper = mount(TimerDisplay, { props: { size: 260 } })

      expect(wrapper.vm.radius).toBe(120)
    })

    it('computes circumference from radius', () => {
      mockTimerStore.remainingTime = 300
      const wrapper = mount(TimerDisplay, { props: { size: 260 } })

      expect(wrapper.vm.circumference).toBeCloseTo(2 * Math.PI * 120, 1)
    })

    it('computes strokeDashoffset from percentage', () => {
      mockTimerStore.percentage = 50
      const wrapper = mount(TimerDisplay, { props: { size: 260 } })

      const expected = wrapper.vm.circumference - 0.5 * wrapper.vm.circumference
      expect(wrapper.vm.strokeDashoffset).toBe(expected)
    })
  })

  describe('SVG icon', () => {
    it('renders clock icon for work mode', () => {
      mockTimerStore.currentMode = 'work'
      mockTimerStore.currentSession = { task_id: 't1' }

      const wrapper = mount(TimerDisplay, { props: { size: 200 } })

      expect(wrapper.find('.timer-icon').html()).toContain('M12 6v6l4 2')
    })

    it('renders coffee icon for short_break mode', () => {
      mockTimerStore.currentMode = 'short_break'
      mockTimerStore.currentSession = { task_id: 't1' }

      const wrapper = mount(TimerDisplay, { props: { size: 200 } })

      expect(wrapper.find('.timer-icon').html()).toContain('M3 8h14v9a4')
    })

    it('renders moon icon for long_break mode', () => {
      mockTimerStore.currentMode = 'long_break'
      mockTimerStore.currentSession = { task_id: 't1' }

      const wrapper = mount(TimerDisplay, { props: { size: 200 } })

      expect(wrapper.find('.timer-icon').html()).toContain('M17.5 19H9a7')
    })
  })

  describe('color', () => {
    it('uses primary color for work mode', () => {
      mockTimerStore.currentMode = 'work'
      mockTimerStore.currentSession = { task_id: 't1' }

      const wrapper = mount(TimerDisplay, { props: { size: 200 } })

      expect(wrapper.vm.color).toBe('var(--accent-primary)')
    })

    it('uses sage color for short_break mode', () => {
      mockTimerStore.currentMode = 'short_break'
      mockTimerStore.currentSession = { task_id: 't1' }

      const wrapper = mount(TimerDisplay, { props: { size: 200 } })

      expect(wrapper.vm.color).toBe('var(--accent-sage)')
    })

    it('uses gold color for long_break mode', () => {
      mockTimerStore.currentMode = 'long_break'
      mockTimerStore.currentSession = { task_id: 't1' }

      const wrapper = mount(TimerDisplay, { props: { size: 200 } })

      expect(wrapper.vm.color).toBe('var(--accent-gold)')
    })
  })
})
