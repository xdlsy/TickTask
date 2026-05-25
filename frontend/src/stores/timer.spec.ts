import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useTimerStore } from '@/stores/timer'

const mockApi = vi.hoisted(() => ({
  getActiveSession: vi.fn(),
  getRecentSessions: vi.fn(),
  createSession: vi.fn(),
  controlSession: vi.fn()
}))

vi.mock('@/api/client', () => ({
  api: mockApi
}))

describe('Timer Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    mockApi.getActiveSession.mockResolvedValue({ data: null })
    mockApi.getRecentSessions.mockResolvedValue({ data: [] })
  })

  it('initializes with no current session', () => {
    const store = useTimerStore()
    expect(store.currentSession).toBeNull()
  })

  it('initializes with zero remaining time', () => {
    const store = useTimerStore()
    expect(store.remainingTime).toBe(0)
  })

  it('initializes with zero total time', () => {
    const store = useTimerStore()
    expect(store.totalTime).toBe(0)
  })

  it('isRunning is false initially', () => {
    const store = useTimerStore()
    expect(store.isRunning).toBe(false)
  })

  it('isPaused is false initially', () => {
    const store = useTimerStore()
    expect(store.isPaused).toBe(false)
  })

  it('percentage is 0 when total time is 0', () => {
    const store = useTimerStore()
    expect(store.percentage).toBe(0)
  })

  it('recentSessions is empty initially', () => {
    const store = useTimerStore()
    expect(store.recentSessions).toEqual([])
  })

  describe('controlSession', () => {
    it('sets remainingTime to 0 on complete', async () => {
      const store = useTimerStore()
      store.currentSession = {
        id: 's1',
        task_id: 't1',
        type: 'work',
        status: 'running',
        planned_duration: 1500,
        actual_duration: 0,
        start_time: new Date().toISOString(),
        end_time: null,
        interrupt_reason: null,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
      }
      store.remainingTime = 900

      mockApi.controlSession.mockResolvedValue({ data: {} })

      await store.controlSession('complete')

      expect(store.remainingTime).toBe(0)
    })

    it('sets remainingTime to 0 on abandon', async () => {
      const store = useTimerStore()
      store.currentSession = {
        id: 's1',
        task_id: 't1',
        type: 'work',
        status: 'running',
        planned_duration: 1500,
        actual_duration: 0,
        start_time: new Date().toISOString(),
        end_time: null,
        interrupt_reason: null,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
      }
      store.remainingTime = 600

      mockApi.controlSession.mockResolvedValue({ data: {} })

      await store.controlSession('abandon')

      expect(store.remainingTime).toBe(0)
    })
  })
})