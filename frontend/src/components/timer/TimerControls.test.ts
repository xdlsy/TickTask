import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import TimerControls from './TimerControls.vue'

const store = vi.hoisted(() => ({
  currentSession: null as { status: string } | null,
  isRunning: false,
  isPaused: false,
  createSession: vi.fn(),
  controlSession: vi.fn()
}))

vi.mock('@/stores/timer', () => ({
  useTimerStore: () => store
}))

const elMsg = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  warning: vi.fn()
}))

const elMsgBox = vi.hoisted(() => ({
  confirm: vi.fn()
}))

vi.mock('element-plus', () => ({
  ElMessage: elMsg,
  ElMessageBox: elMsgBox
}))

describe('TimerControls', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    store.currentSession = null
    store.isRunning = false
    store.isPaused = false
    vi.clearAllMocks()
  })

  describe('start button visibility', () => {
    it('shows start button when no session', () => {
      const wrapper = mount(TimerControls)
      expect(wrapper.find('.start-btn').exists()).toBe(true)
    })

    it('shows start button when session is completed', () => {
      store.currentSession = { status: 'completed' }
      const wrapper = mount(TimerControls)
      expect(wrapper.find('.start-btn').exists()).toBe(true)
    })

    it('shows start button when session is abandoned', () => {
      store.currentSession = { status: 'abandoned' }
      const wrapper = mount(TimerControls)
      expect(wrapper.find('.start-btn').exists()).toBe(true)
    })

    it('hides start button when session is active', () => {
      store.currentSession = { status: 'running' }
      const wrapper = mount(TimerControls)
      expect(wrapper.find('.start-btn').exists()).toBe(false)
    })
  })

  describe('action buttons', () => {
    it('shows pause button when running', () => {
      store.currentSession = { status: 'running' }
      store.isRunning = true
      const wrapper = mount(TimerControls)
      expect(wrapper.find('.pause-btn').exists()).toBe(true)
    })

    it('shows resume button when paused', () => {
      store.currentSession = { status: 'paused' }
      store.isPaused = true
      const wrapper = mount(TimerControls)
      expect(wrapper.find('.resume-btn').exists()).toBe(true)
    })

    it('shows complete and abandon buttons when running', () => {
      store.currentSession = { status: 'running' }
      store.isRunning = true
      const wrapper = mount(TimerControls)
      expect(wrapper.find('.complete-btn').exists()).toBe(true)
      expect(wrapper.find('.abandon-btn').exists()).toBe(true)
    })

    it('shows complete and abandon buttons when paused', () => {
      store.currentSession = { status: 'paused' }
      store.isPaused = true
      const wrapper = mount(TimerControls)
      expect(wrapper.find('.complete-btn').exists()).toBe(true)
      expect(wrapper.find('.abandon-btn').exists()).toBe(true)
    })

    it('never shows pause and resume buttons simultaneously', () => {
      store.currentSession = { status: 'running' }
      store.isRunning = true
      store.isPaused = false
      const wrapper = mount(TimerControls)
      expect(wrapper.find('.pause-btn').exists()).toBe(true)
      expect(wrapper.find('.resume-btn').exists()).toBe(false)
    })
  })

  describe('quick action buttons', () => {
    it('always shows quick action buttons when no session', () => {
      const wrapper = mount(TimerControls)
      expect(wrapper.find('.quick-btn.short').exists()).toBe(true)
      expect(wrapper.find('.quick-btn.long').exists()).toBe(true)
    })

    it('always shows quick action buttons when session is running', () => {
      store.currentSession = { status: 'running' }
      store.isRunning = true
      const wrapper = mount(TimerControls)
      expect(wrapper.find('.quick-btn.short').exists()).toBe(true)
      expect(wrapper.find('.quick-btn.long').exists()).toBe(true)
    })

    it('always shows quick action buttons when session is paused', () => {
      store.currentSession = { status: 'paused' }
      store.isPaused = true
      const wrapper = mount(TimerControls)
      expect(wrapper.find('.quick-btn.short').exists()).toBe(true)
      expect(wrapper.find('.quick-btn.long').exists()).toBe(true)
    })
  })

  describe('startWork', () => {
    it('calls createSession with null and work mode on start', async () => {
      store.createSession = vi.fn().mockResolvedValue(undefined)
      const wrapper = mount(TimerControls)

      await wrapper.find('.start-btn').trigger('click')

      expect(store.createSession).toHaveBeenCalledWith(null, 'work')
    })

    it('shows error on startWork failure', async () => {
      store.createSession = vi.fn().mockRejectedValue(new Error('fail'))
      const wrapper = mount(TimerControls)

      await wrapper.find('.start-btn').trigger('click')

      expect(elMsg.error).toHaveBeenCalledWith('启动计时器失败')
    })
  })

  describe('startShortBreak', () => {
    it('calls createSession with short_break mode', async () => {
      store.createSession = vi.fn().mockResolvedValue(undefined)
      const wrapper = mount(TimerControls)

      await wrapper.find('.quick-btn.short').trigger('click')

      expect(store.createSession).toHaveBeenCalledWith(null, 'short_break')
    })

    it('shows error on short_break failure', async () => {
      store.createSession = vi.fn().mockRejectedValue(new Error('fail'))
      const wrapper = mount(TimerControls)

      await wrapper.find('.quick-btn.short').trigger('click')

      expect(elMsg.error).toHaveBeenCalledWith('启动短休息失败')
    })
  })

  describe('startLongBreak', () => {
    it('calls createSession with long_break mode', async () => {
      store.createSession = vi.fn().mockResolvedValue(undefined)
      const wrapper = mount(TimerControls)

      await wrapper.find('.quick-btn.long').trigger('click')

      expect(store.createSession).toHaveBeenCalledWith(null, 'long_break')
    })

    it('shows error on long_break failure', async () => {
      store.createSession = vi.fn().mockRejectedValue(new Error('fail'))
      const wrapper = mount(TimerControls)

      await wrapper.find('.quick-btn.long').trigger('click')

      expect(elMsg.error).toHaveBeenCalledWith('启动长休息失败')
    })
  })

  describe('pause', () => {
    it('calls controlSession with pause', async () => {
      store.currentSession = { status: 'running' }
      store.isRunning = true
      store.controlSession = vi.fn().mockResolvedValue(undefined)
      const wrapper = mount(TimerControls)

      await wrapper.find('.pause-btn').trigger('click')

      expect(store.controlSession).toHaveBeenCalledWith('pause')
    })

    it('shows error on pause failure', async () => {
      store.currentSession = { status: 'running' }
      store.isRunning = true
      store.controlSession = vi.fn().mockRejectedValue(new Error('fail'))
      const wrapper = mount(TimerControls)

      await wrapper.find('.pause-btn').trigger('click')

      expect(elMsg.error).toHaveBeenCalledWith('暂停失败')
    })
  })

  describe('resume', () => {
    it('calls controlSession with resume', async () => {
      store.currentSession = { status: 'paused' }
      store.isPaused = true
      store.controlSession = vi.fn().mockResolvedValue(undefined)
      const wrapper = mount(TimerControls)

      await wrapper.find('.resume-btn').trigger('click')

      expect(store.controlSession).toHaveBeenCalledWith('resume')
    })

    it('shows error on resume failure', async () => {
      store.currentSession = { status: 'paused' }
      store.isPaused = true
      store.controlSession = vi.fn().mockRejectedValue(new Error('fail'))
      const wrapper = mount(TimerControls)

      await wrapper.find('.resume-btn').trigger('click')

      expect(elMsg.error).toHaveBeenCalledWith('继续失败')
    })
  })

  describe('complete', () => {
    it('calls controlSession with complete and shows success', async () => {
      store.currentSession = { status: 'running' }
      store.isRunning = true
      store.controlSession = vi.fn().mockResolvedValue(undefined)
      const wrapper = mount(TimerControls)

      await wrapper.find('.complete-btn').trigger('click')

      expect(store.controlSession).toHaveBeenCalledWith('complete')
      expect(elMsg.success).toHaveBeenCalledWith('番茄完成!')
    })

    it('shows error on complete failure', async () => {
      store.currentSession = { status: 'running' }
      store.isRunning = true
      store.controlSession = vi.fn().mockRejectedValue(new Error('fail'))
      const wrapper = mount(TimerControls)

      await wrapper.find('.complete-btn').trigger('click')

      expect(elMsg.error).toHaveBeenCalledWith('完成失败')
    })
  })

  describe('abandon', () => {
    it('shows confirm dialog then calls controlSession with abandon and other reason', async () => {
      store.currentSession = { status: 'running' }
      store.isRunning = true
      store.controlSession = vi.fn().mockResolvedValue(undefined)
      elMsgBox.confirm = vi.fn().mockResolvedValue('confirm')
      const wrapper = mount(TimerControls)

      await wrapper.find('.abandon-btn').trigger('click')

      expect(elMsgBox.confirm).toHaveBeenCalledWith(
        '放弃当前计时？AI 将记录此次打断并调整后续排程',
        '放弃计时',
        { confirmButtonText: '确认放弃', cancelButtonText: '返回', type: 'warning' }
      )
      expect(store.controlSession).toHaveBeenCalledWith('abandon', 'other')
    })

    it('does not call controlSession when user cancels confirm dialog', async () => {
      store.currentSession = { status: 'running' }
      store.isRunning = true
      store.controlSession = vi.fn().mockResolvedValue(undefined)
      elMsgBox.confirm = vi.fn().mockRejectedValue('cancel')
      const wrapper = mount(TimerControls)

      await wrapper.find('.abandon-btn').trigger('click')

      expect(elMsgBox.confirm).toHaveBeenCalled()
      expect(store.controlSession).not.toHaveBeenCalled()
      expect(elMsg.error).not.toHaveBeenCalled()
    })

    it('shows error on abandon after confirm when API fails', async () => {
      store.currentSession = { status: 'running' }
      store.isRunning = true
      store.controlSession = vi.fn().mockRejectedValue(new Error('API fail'))
      elMsgBox.confirm = vi.fn().mockResolvedValue('confirm')
      const wrapper = mount(TimerControls)

      await wrapper.find('.abandon-btn').trigger('click')

      expect(elMsgBox.confirm).toHaveBeenCalled()
      expect(elMsg.error).toHaveBeenCalledWith('放弃失败')
    })
  })
})
