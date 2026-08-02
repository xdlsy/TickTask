import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useTimerStore } from '@/stores/timer'
import { api } from '@/api/client'
import { wsClient } from '@/utils/websocket'
import type { PomodoroSession } from '@/types'

const mockSession: PomodoroSession = {
  id: 'session-1',
  task_id: 'task-1',
  type: 'work',
  status: 'running',
  start_time: '2026-05-21T09:00:00Z',
  end_time: null,
  planned_duration: 1500,
  actual_duration: null,
  interruptions: 0,
  created_at: '2026-05-21T09:00:00Z'
}

vi.mock('@/api/client', () => ({
  api: {
    getActiveSession: vi.fn(),
    createSession: vi.fn(),
    controlSession: vi.fn(),
    getRecentSessions: vi.fn()
  }
}))

vi.mock('@/utils/websocket', () => ({
  wsClient: { on: vi.fn() }
}))

describe('Timer Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-05-21T09:10:00Z'))
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('initial state', () => {
    it('should have correct initial state', () => {
      const store = useTimerStore()
      expect(store.currentSession).toBeNull()
      expect(store.remainingTime).toBe(0)
      expect(store.totalTime).toBe(0)
      expect(store.recentSessions).toEqual([])
    })

    it('isRunning should be false when no session', () => {
      const store = useTimerStore()
      expect(store.isRunning).toBe(false)
    })

    it('isPaused should be false when no session', () => {
      const store = useTimerStore()
      expect(store.isPaused).toBe(false)
    })

    it('percentage should be 0 when totalTime is 0', () => {
      const store = useTimerStore()
      expect(store.percentage).toBe(0)
    })

    it('currentMode should default to work', () => {
      const store = useTimerStore()
      expect(store.currentMode).toBe('work')
    })
  })

  describe('computed properties with session', () => {
    it('isRunning should reflect session status', () => {
      const store = useTimerStore()
      store.currentSession = { ...mockSession, status: 'running' }
      expect(store.isRunning).toBe(true)
    })

    it('isPaused should reflect session status', () => {
      const store = useTimerStore()
      store.currentSession = { ...mockSession, status: 'paused' }
      expect(store.isPaused).toBe(true)
    })

    it('percentage should calculate correctly', () => {
      const store = useTimerStore()
      store.totalTime = 1800
      store.remainingTime = 900
      expect(store.percentage).toBe(50)
    })

    it('currentMode should return session type', () => {
      const store = useTimerStore()
      store.currentSession = { ...mockSession, type: 'short_break' }
      expect(store.currentMode).toBe('short_break')
    })
  })

  describe('fetchActiveSession', () => {
    it('should handle session-in-data response format', async () => {
      const store = useTimerStore()
      ;(api.getActiveSession as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { session: { ...mockSession } }
      })

      await store.fetchActiveSession()

      expect(store.currentSession).toBeTruthy()
      expect(store.totalTime).toBe(1500)
    })

    it('should handle session-is-null in data', async () => {
      const store = useTimerStore()
      store.currentSession = { ...mockSession }
      ;(api.getActiveSession as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { session: null }
      })

      await store.fetchActiveSession()

      expect(store.currentSession).toBeNull()
    })

    it('should handle direct session response', async () => {
      const store = useTimerStore()
      ;(api.getActiveSession as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { ...mockSession }
      })

      await store.fetchActiveSession()

      expect(store.currentSession).toBeTruthy()
    })

    it('should handle null/empty response', async () => {
      const store = useTimerStore()
      ;(api.getActiveSession as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: null
      })

      await store.fetchActiveSession()

      expect(store.currentSession).toBeNull()
    })

    it('should handle fetch error gracefully', async () => {
      const store = useTimerStore()
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
      ;(api.getActiveSession as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Network'))

      await store.fetchActiveSession()

      expect(consoleSpy).toHaveBeenCalled()
      consoleSpy.mockRestore()
    })
  })

  describe('createSession', () => {
    it('should create session with default params', async () => {
      const store = useTimerStore()
      ;(api.createSession as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { ...mockSession, planned_duration: 1800 }
      })

      const result = await store.createSession()

      expect(api.createSession).toHaveBeenCalledWith({
        task_id: null,
        type: 'work',
        duration: undefined
      })
      expect(result.planned_duration).toBe(1800)
      expect(store.totalTime).toBe(1800)
      expect(store.remainingTime).toBe(1800)
    })

    it('should create session with full params', async () => {
      const store = useTimerStore()
      ;(api.createSession as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { ...mockSession, type: 'short_break', planned_duration: 300 }
      })

      await store.createSession('task-1', 'short_break', 300)

      expect(api.createSession).toHaveBeenCalledWith({
        task_id: 'task-1',
        type: 'short_break',
        duration: 300
      })
    })

    it('should propagate errors', async () => {
      const store = useTimerStore()
      ;(api.createSession as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Bad Request'))

      await expect(store.createSession()).rejects.toThrow('Bad Request')
    })
  })

  describe('controlSession', () => {
    it('should do nothing if no current session', async () => {
      const store = useTimerStore()
      ;(api.controlSession as ReturnType<typeof vi.fn>).mockResolvedValue({ data: {} })

      await store.controlSession('pause')

      expect(api.controlSession).not.toHaveBeenCalled()
    })

    it('should pause session and update local state', async () => {
      const store = useTimerStore()
      store.currentSession = { ...mockSession, status: 'running' }
      ;(api.controlSession as ReturnType<typeof vi.fn>).mockResolvedValue({ data: {} })
      ;(api.getRecentSessions as ReturnType<typeof vi.fn>).mockResolvedValue({ data: [] })

      await store.controlSession('pause')

      expect(api.controlSession).toHaveBeenCalledWith('session-1', 'pause', undefined)
      expect(store.currentSession?.status).toBe('paused')
    })

    it('should complete session and set end_time', async () => {
      const store = useTimerStore()
      store.currentSession = { ...mockSession, status: 'running' }
      ;(api.controlSession as ReturnType<typeof vi.fn>).mockResolvedValue({ data: {} })
      ;(api.getRecentSessions as ReturnType<typeof vi.fn>).mockResolvedValue({ data: [] })

      await store.controlSession('complete')

      expect(store.currentSession?.status).toBe('completed')
      expect(store.currentSession?.end_time).toBe('2026-05-21T09:10:00.000Z')
    })

    it('should abandon session and clear state', async () => {
      const store = useTimerStore()
      store.currentSession = { ...mockSession, status: 'running' }
      ;(api.controlSession as ReturnType<typeof vi.fn>).mockResolvedValue({ data: {} })
      ;(api.getRecentSessions as ReturnType<typeof vi.fn>).mockResolvedValue({ data: [] })

      await store.controlSession('abandon')

      expect(store.currentSession).toBeNull()
    })

    it('should propagate control errors', async () => {
      const store = useTimerStore()
      store.currentSession = { ...mockSession }
      ;(api.controlSession as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Not Found'))

      await expect(store.controlSession('pause')).rejects.toThrow('Not Found')
    })
  })

  describe('fetchRecentSessions', () => {
    it('should fetch and store recent sessions', async () => {
      const store = useTimerStore()
      const sessions: PomodoroSession[] = [{ ...mockSession, id: 'r1', status: 'completed' }]
      ;(api.getRecentSessions as ReturnType<typeof vi.fn>).mockResolvedValue({ data: sessions })

      await store.fetchRecentSessions(10)

      expect(api.getRecentSessions).toHaveBeenCalledWith(10)
      expect(store.recentSessions).toHaveLength(1)
    })

    it('should use default limit of 20', async () => {
      const store = useTimerStore()
      ;(api.getRecentSessions as ReturnType<typeof vi.fn>).mockResolvedValue({ data: [] })

      await store.fetchRecentSessions()

      expect(api.getRecentSessions).toHaveBeenCalledWith(20)
    })

    it('should handle fetch error gracefully', async () => {
      const store = useTimerStore()
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
      ;(api.getRecentSessions as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Network'))

      await store.fetchRecentSessions()

      expect(consoleSpy).toHaveBeenCalled()
      consoleSpy.mockRestore()
    })
  })

  describe('WebSocket setup', () => {
    it('should register WebSocket listeners', () => {
      const store = useTimerStore()
      store.setupWebSocket()

      expect(wsClient.on).toHaveBeenCalledWith('timer_tick', expect.any(Function))
      expect(wsClient.on).toHaveBeenCalledWith('session_state', expect.any(Function))
      expect(wsClient.on).toHaveBeenCalledWith('timer_complete', expect.any(Function))
    })
  })

  describe('interrupt handling', () => {
    it('should abandon session with interrupt reason', async () => {
      const store = useTimerStore()
      store.currentSession = { ...mockSession, status: 'running' }
      ;(api.controlSession as ReturnType<typeof vi.fn>).mockResolvedValue({ data: {} })
      ;(api.getRecentSessions as ReturnType<typeof vi.fn>).mockResolvedValue({ data: [] })

      await store.controlSession('abandon', 'meeting')

      expect(api.controlSession).toHaveBeenCalledWith('session-1', 'abandon', 'meeting')
      expect(store.currentSession).toBeNull()
    })

    it('should abandon session with call interrupt reason', async () => {
      const store = useTimerStore()
      store.currentSession = { ...mockSession, status: 'running' }
      ;(api.controlSession as ReturnType<typeof vi.fn>).mockResolvedValue({ data: {} })
      ;(api.getRecentSessions as ReturnType<typeof vi.fn>).mockResolvedValue({ data: [] })

      await store.controlSession('abandon', 'call')

      expect(api.controlSession).toHaveBeenCalledWith('session-1', 'abandon', 'call')
    })

    it('should abandon session with urgent interrupt reason', async () => {
      const store = useTimerStore()
      store.currentSession = { ...mockSession, status: 'running' }
      ;(api.controlSession as ReturnType<typeof vi.fn>).mockResolvedValue({ data: {} })
      ;(api.getRecentSessions as ReturnType<typeof vi.fn>).mockResolvedValue({ data: [] })

      await store.controlSession('abandon', 'urgent')

      expect(api.controlSession).toHaveBeenCalledWith('session-1', 'abandon', 'urgent')
    })

    it('should abandon session with other interrupt reason', async () => {
      const store = useTimerStore()
      store.currentSession = { ...mockSession, status: 'running' }
      ;(api.controlSession as ReturnType<typeof vi.fn>).mockResolvedValue({ data: {} })
      ;(api.getRecentSessions as ReturnType<typeof vi.fn>).mockResolvedValue({ data: [] })

      await store.controlSession('abandon', 'other')

      expect(api.controlSession).toHaveBeenCalledWith('session-1', 'abandon', 'other')
    })

    it('should clear currentSession after abandon regardless of reason', async () => {
      const store = useTimerStore()
      store.currentSession = { ...mockSession, status: 'running' }
      store.remainingTime = 900
      ;(api.controlSession as ReturnType<typeof vi.fn>).mockResolvedValue({ data: {} })
      ;(api.getRecentSessions as ReturnType<typeof vi.fn>).mockResolvedValue({ data: [] })

      await store.controlSession('abandon', 'meeting')

      expect(store.currentSession).toBeNull()
    })
  })

  describe('remainingTime calculation (via fetchActiveSession)', () => {
    it('should calculate remaining time based on elapsed time', async () => {
      const store = useTimerStore()
      // session started 10 min ago, planned 30 min => 20 min remaining = 1200s
      ;(api.getActiveSession as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { ...mockSession, planned_duration: 1800, start_time: '2026-05-21T09:00:00Z' }
      })

      await store.fetchActiveSession()

      expect(store.remainingTime).toBe(1200)
    })

    it('should floor remaining time at 0', async () => {
      const store = useTimerStore()
      // session started 10 min ago, planned 5 min => elapsed > planned => 0
      ;(api.getActiveSession as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { ...mockSession, planned_duration: 300, start_time: '2026-05-21T09:00:00Z' }
      })

      await store.fetchActiveSession()

      expect(store.remainingTime).toBe(0)
    })
  })
})
