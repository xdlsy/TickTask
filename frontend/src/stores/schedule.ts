import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/api/client'
import { wsClient } from '@/utils/websocket'
import type { ScheduleEvent, CreateScheduleDTO, UpdateScheduleDTO, MoveScheduleDTO, ViewMode, TerminalOutputMessage, TerminalStatusMessage, RevisionChange } from '@/types'

export interface TerminalLine {
  text: string
  isStderr: boolean
  timestamp: number
}

export const useScheduleStore = defineStore('schedule', () => {
  const events = ref<ScheduleEvent[]>([])
  const loading = ref(false)
  const viewMode = ref<ViewMode>('week')
  const currentDate = ref(new Date())
  const aiReasoning = ref('')
  const aiGenerating = ref(false)
  const cliToolName = ref('Python 调度引擎')
  const terminalOutput = ref<TerminalLine[]>([])
  const terminalStatus = ref<string>('') // '', 'started', 'completed', 'error'
  const terminalStatusMessage = ref('')
  const terminalStatusDetail = ref('')
  const revisionChanges = ref<RevisionChange[]>([])
  const revisionSummary = ref('')

  // WebSocket 监听
  let outputHandler: ((msg: any) => void) | null = null
  let statusHandler: ((msg: any) => void) | null = null

  function setupTerminalListener() {
    if (outputHandler) return
    outputHandler = (msg: TerminalOutputMessage) => {
      console.log('[terminal] output chunk:', msg.chunk?.substring(0, 50))
      terminalOutput.value.push({
        text: msg.chunk,
        isStderr: msg.is_stderr || false,
        timestamp: Date.now()
      })
    }
    statusHandler = (msg: TerminalStatusMessage) => {
      console.log('[terminal] status:', msg.status, msg.message)
      terminalStatus.value = msg.status
      terminalStatusMessage.value = msg.message || ''
      terminalStatusDetail.value = msg.detail || ''
      if (msg.status === 'completed' || msg.status === 'error') {
        setTimeout(() => {
          aiGenerating.value = false
        }, 2000)
      }
    }
    wsClient.on('terminal_output', outputHandler)
    wsClient.on('terminal_status', statusHandler)
    console.log('[terminal] listeners registered')
  }

  function cleanupTerminalListener() {
    if (outputHandler) {
      wsClient.off('terminal_output', outputHandler)
      outputHandler = null
    }
    if (statusHandler) {
      wsClient.off('terminal_status', statusHandler)
      statusHandler = null
    }
  }

  async function fetchSchedules(start?: string, end?: string) {
    loading.value = true
    try {
      const res = await api.getSchedules(start, end)
      events.value = res.data.events
    } catch (error) {
      console.error('Failed to fetch schedules:', error)
    } finally {
      loading.value = false
    }
  }

  async function createSchedule(data: CreateScheduleDTO) {
    try {
      const res = await api.createSchedule(data)
      events.value.push(res.data)
      return res.data
    } catch (error) {
      console.error('Failed to create schedule:', error)
      throw error
    }
  }

  async function updateSchedule(id: string, data: UpdateScheduleDTO) {
    try {
      await api.updateSchedule(id, data)
      const index = events.value.findIndex(e => e.id === id)
      if (index !== -1) {
        // 更新本地数据
        if (data.title) events.value[index].title = data.title
        if (data.start_time) events.value[index].start = data.start_time
        if (data.end_time) events.value[index].end = data.end_time
        if (data.status) events.value[index].status = data.status
        if (data.color) events.value[index].color = data.color
      }
    } catch (error) {
      console.error('Failed to update schedule:', error)
      throw error
    }
  }

  async function deleteSchedule(id: string) {
    try {
      await api.deleteSchedule(id)
      events.value = events.value.filter(e => e.id !== id)
    } catch (error) {
      console.error('Failed to delete schedule:', error)
      throw error
    }
  }

  async function moveSchedule(id: string, data: MoveScheduleDTO) {
    try {
      await api.moveSchedule(id, data)
      const index = events.value.findIndex(e => e.id === id)
      if (index !== -1) {
        events.value[index].start = data.start_time
        events.value[index].end = data.end_time
      }
    } catch (error) {
      console.error('Failed to move schedule:', error)
      throw error
    }
  }

  async function generateSchedule(startTime?: string, endTime?: string) {
    loading.value = true
    aiGenerating.value = true
    terminalOutput.value = []
    terminalStatus.value = ''
    terminalStatusMessage.value = ''
    terminalStatusDetail.value = ''
    setupTerminalListener()
    try {
      const res = await api.generateScheduleFromTasks(startTime, endTime)
      const generatedEvents = res.data.events as ScheduleEvent[]
      aiReasoning.value = (res.data as any).reasoning || ''
      // 收集新生成事件的 task_id，用于去重
      const newTaskIds = new Set(generatedEvents.map(e => e.task_id).filter(Boolean))
      // 删除旧的任务日程，保留自定义日程（无 task_id）
      events.value = events.value.filter(e => !e.task_id || !newTaskIds.has(e.task_id))
      // 添加新日程
      events.value = [...events.value, ...generatedEvents]
      return generatedEvents
    } catch (error: any) {
      // Only set error manually if WebSocket didn't already set a terminal state
      if (terminalStatus.value === '' || terminalStatus.value === 'started') {
        const errMsg = error?.response?.data?.error || (error as Error)?.message || 'AI 请求失败'
        terminalOutput.value.push({
          text: `错误: ${errMsg}`,
          isStderr: true,
          timestamp: Date.now()
        })
        terminalStatus.value = 'error'
        terminalStatusMessage.value = '日程生成失败'
        terminalStatusDetail.value = errMsg
      }
      // Keep terminal visible for user to read error/output
      setTimeout(() => {
        aiGenerating.value = false
      }, 5000)
      console.error('Failed to generate schedule:', error)
      throw error
    } finally {
      loading.value = false
    }
  }

  async function reviseSchedule(prompt: string) {
    loading.value = true
    aiGenerating.value = true
    cliToolName.value = '日程修订引擎'
    terminalOutput.value = []
    terminalStatus.value = ''
    terminalStatusMessage.value = ''
    terminalStatusDetail.value = ''
    revisionChanges.value = []
    revisionSummary.value = ''
    setupTerminalListener()
    try {
      const res = await api.reviseSchedule(prompt)
      revisionChanges.value = res.data.changes
      revisionSummary.value = res.data.summary
      aiReasoning.value = res.data.summary
      return res.data
    } catch (error: any) {
      if (terminalStatus.value === '' || terminalStatus.value === 'started') {
        const errMsg = error?.response?.data?.error || (error as Error)?.message || '修订请求失败'
        terminalOutput.value.push({
          text: `错误: ${errMsg}`,
          isStderr: true,
          timestamp: Date.now()
        })
        terminalStatus.value = 'error'
        terminalStatusMessage.value = '日程修订失败'
        terminalStatusDetail.value = errMsg
      }
      setTimeout(() => {
        aiGenerating.value = false
      }, 5000)
      console.error('Failed to revise schedule:', error)
      throw error
    } finally {
      loading.value = false
    }
  }

  async function applyRevision() {
    loading.value = true
    try {
      const res = await api.applyRevision()
      const appliedEvents = res.data.events as ScheduleEvent[]
      // Deduplicate by task_id, preserve custom events (no task_id)
      const newTaskIds = new Set(appliedEvents.map(e => e.task_id).filter(Boolean))
      events.value = events.value.filter(e => !e.task_id || !newTaskIds.has(e.task_id))
      events.value = [...events.value, ...appliedEvents]
      revisionChanges.value = []
      revisionSummary.value = ''
      return appliedEvents
    } catch (error) {
      console.error('Failed to apply revision:', error)
      throw error
    } finally {
      loading.value = false
    }
  }

  function setViewMode(mode: ViewMode) {
    viewMode.value = mode
  }

  function setCurrentDate(date: Date) {
    currentDate.value = date
  }

  function goToPrevious() {
    const newDate = new Date(currentDate.value)
    if (viewMode.value === 'day') {
      newDate.setDate(newDate.getDate() - 1)
    } else if (viewMode.value === 'week') {
      newDate.setDate(newDate.getDate() - 7)
    } else {
      newDate.setMonth(newDate.getMonth() - 1)
    }
    currentDate.value = newDate
  }

  function goToNext() {
    const newDate = new Date(currentDate.value)
    if (viewMode.value === 'day') {
      newDate.setDate(newDate.getDate() + 1)
    } else if (viewMode.value === 'week') {
      newDate.setDate(newDate.getDate() + 7)
    } else {
      newDate.setMonth(newDate.getMonth() + 1)
    }
    currentDate.value = newDate
  }

  function goToToday() {
    currentDate.value = new Date()
  }

  async function resetSchedules() {
    try {
      const res = await api.deleteAllSchedules()
      events.value = []
      return res.data.deleted
    } catch (error) {
      console.error('Failed to reset schedules:', error)
      throw error
    }
  }

  return {
    events,
    loading,
    viewMode,
    currentDate,
    aiReasoning,
    aiGenerating,
    cliToolName,
    terminalOutput,
    terminalStatus,
    terminalStatusMessage,
    terminalStatusDetail,
    revisionChanges,
    revisionSummary,
    fetchSchedules,
    createSchedule,
    updateSchedule,
    deleteSchedule,
    moveSchedule,
    generateSchedule,
    reviseSchedule,
    applyRevision,
    setupTerminalListener,
    cleanupTerminalListener,
    setViewMode,
    setCurrentDate,
    goToPrevious,
    goToNext,
    goToToday,
    resetSchedules
  }
})