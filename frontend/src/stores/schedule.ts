import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/api/client'
import type { ScheduleEvent, CreateScheduleDTO, UpdateScheduleDTO, MoveScheduleDTO, ViewMode } from '@/types'

export const useScheduleStore = defineStore('schedule', () => {
  const events = ref<ScheduleEvent[]>([])
  const loading = ref(false)
  const viewMode = ref<ViewMode>('week')
  const currentDate = ref(new Date())

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
    try {
      const res = await api.generateScheduleFromTasks(startTime, endTime)
      const generatedEvents = res.data.events as ScheduleEvent[]
      // 收集新生成事件的 task_id，用于去重
      const newTaskIds = new Set(generatedEvents.map(e => e.task_id).filter(Boolean))
      // 删除旧的任务日程，保留自定义日程（无 task_id）
      events.value = events.value.filter(e => !e.task_id || !newTaskIds.has(e.task_id))
      // 添加新日程
      events.value = [...events.value, ...generatedEvents]
      return generatedEvents
    } catch (error) {
      console.error('Failed to generate schedule:', error)
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

  return {
    events,
    loading,
    viewMode,
    currentDate,
    fetchSchedules,
    createSchedule,
    updateSchedule,
    deleteSchedule,
    moveSchedule,
    generateSchedule,
    setViewMode,
    setCurrentDate,
    goToPrevious,
    goToNext,
    goToToday
  }
})