import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useTimerStore } from '@/stores/timer'

describe('Timer Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
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
})