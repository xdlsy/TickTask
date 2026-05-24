import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { PomodoroSession, SessionType } from '@/types'
import { api } from '@/api/client'
import { wsClient } from '@/utils/websocket'
import type { WSMessage, TimerTickMessage, SessionStateMessage, TimerCompleteMessage } from '@/types'

export const useTimerStore = defineStore('timer', () => {
  // State
  const currentSession = ref<PomodoroSession | null>(null)
  const remainingTime = ref(0)
  const totalTime = ref(0)
  const recentSessions = ref<PomodoroSession[]>([])

  // Computed
  const isRunning = computed(() => currentSession.value?.status === 'running')
  const isPaused = computed(() => currentSession.value?.status === 'paused')
  const percentage = computed(() => {
    if (totalTime.value === 0) return 0
    return Math.round((remainingTime.value / totalTime.value) * 100)
  })
  const currentMode = computed((): SessionType => currentSession.value?.type || 'work')

  // Actions
  async function fetchActiveSession() {
    try {
      const res = await api.getActiveSession()
      const data = res.data
      if (data && typeof data === 'object') {
        if ('session' in data) {
          // Response is { session: PomodoroSession | null }
          const session = data.session
          if (session) {
            currentSession.value = session
            totalTime.value = session.planned_duration
            remainingTime.value = calculateRemainingTime(session)
          } else {
            currentSession.value = null
          }
        } else {
          // Response is PomodoroSession directly - use type assertion since backend returns it this way
          const session = data as unknown as PomodoroSession
          currentSession.value = session
          totalTime.value = session.planned_duration
          remainingTime.value = calculateRemainingTime(session)
        }
      } else {
        currentSession.value = null
      }
    } catch (error) {
      console.error('Failed to fetch active session:', error)
    }
  }

  async function createSession(taskId: string | null = null, type: SessionType = 'work', duration?: number) {
    try {
      const res = await api.createSession({
        task_id: taskId,
        type,
        duration
      })
      currentSession.value = res.data
      totalTime.value = res.data.planned_duration
      remainingTime.value = totalTime.value
      return res.data
    } catch (error) {
      console.error('Failed to create session:', error)
      throw error
    }
  }

  async function controlSession(action: 'pause' | 'resume' | 'complete' | 'abandon', interruptReason?: string) {
    if (!currentSession.value) return

    try {
      await api.controlSession(currentSession.value.id, action, interruptReason)

      // 更新本地状态
      if (action === 'pause') {
        currentSession.value.status = 'paused'
      } else if (action === 'complete') {
        currentSession.value.status = 'completed'
        const now = new Date()
        currentSession.value.end_time = now.toISOString()
      } else if (action === 'abandon') {
        currentSession.value.status = 'abandoned'
        if (interruptReason) {
          currentSession.value.interrupt_reason = interruptReason
        }
        currentSession.value = null
      }

      await fetchRecentSessions()
    } catch (error) {
      console.error('Failed to control session:', error)
      throw error
    }
  }

  async function fetchRecentSessions(limit = 20) {
    try {
      const res = await api.getRecentSessions(limit)
      recentSessions.value = res.data
    } catch (error) {
      console.error('Failed to fetch recent sessions:', error)
    }
  }

  // WebSocket 消息处理
  function setupWebSocket() {
    wsClient.on('timer_tick', handleTimerTick)
    wsClient.on('session_state', handleSessionState)
    wsClient.on('timer_complete', handleTimerComplete)
  }

  function handleTimerTick(message: WSMessage) {
    const tick = message as TimerTickMessage
    if (currentSession.value?.id === tick.session_id) {
      remainingTime.value = tick.remaining
      totalTime.value = tick.total
    }
  }

  function handleSessionState(message: WSMessage) {
    const state = message as SessionStateMessage
    if (currentSession.value?.id === state.id) {
      currentSession.value.status = state.status
    }
  }

  function handleTimerComplete(message: WSMessage) {
    const complete = message as TimerCompleteMessage
    if (currentSession.value?.id === complete.session_id) {
      currentSession.value.status = 'completed'
      const now = new Date()
      currentSession.value.end_time = now.toISOString()
      fetchRecentSessions()
    }
  }

  function calculateRemainingTime(session: PomodoroSession): number {
    const start = new Date(session.start_time).getTime()
    const elapsed = Math.floor((Date.now() - start) / 1000)
    return Math.max(0, session.planned_duration - elapsed)
  }

  return {
    // State
    currentSession,
    remainingTime,
    totalTime,
    recentSessions,
    // Computed
    isRunning,
    isPaused,
    percentage,
    currentMode,
    // Actions
    fetchActiveSession,
    createSession,
    controlSession,
    fetchRecentSessions,
    setupWebSocket
  }
})
