import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '@/api/client'
import type { ClassificationResult, PrioritySuggestion, ScheduleEvent, RescheduleResult, DailyInsights } from '@/types'

export const useAIStore = defineStore('ai', () => {
  // State
  const configured = ref(false)
  const loading = ref(false)
  const lastClassification = ref<ClassificationResult | null>(null)

  // Actions
  async function checkStatus() {
    try {
      const res = await api.getAIStatus()
      configured.value = res.data.configured
      return configured.value
    } catch (error) {
      console.error('Failed to check AI status:', error)
      configured.value = false
      return false
    }
  }

  async function classifyTask(taskId: string): Promise<ClassificationResult | null> {
    loading.value = true
    try {
      const res = await api.classifyTask(taskId)
      lastClassification.value = res.data
      return res.data
    } catch (error) {
      console.error('Failed to classify task:', error)
      throw error
    } finally {
      loading.value = false
    }
  }

  async function classifyTasks(taskIds: string[]): Promise<ClassificationResult[]> {
    loading.value = true
    try {
      const res = await api.classifyTasks(taskIds)
      return res.data.results
    } catch (error) {
      console.error('Failed to classify tasks:', error)
      throw error
    } finally {
      loading.value = false
    }
  }

  async function classifyTaskByText(title: string, description?: string): Promise<ClassificationResult | null> {
    loading.value = true
    try {
      const res = await api.classifyTaskByText(title, description)
      lastClassification.value = res.data
      return res.data
    } catch (error) {
      console.error('Failed to classify task by text:', error)
      throw error
    } finally {
      loading.value = false
    }
  }

  async function generateSchedule(startTime: string, endTime: string): Promise<ScheduleEvent[] | null> {
    loading.value = true
    try {
      const res = await api.generateSchedule(startTime, endTime)
      return res.data.events
    } catch (error) {
      console.error('Failed to generate schedule:', error)
      throw error
    } finally {
      loading.value = false
    }
  }

  async function rescheduleAfterInterrupt(data: {
    task_id: string
    completed_minutes?: number
    planned_minutes: number
    interrupt_reason?: string
    work_end_time: string
  }): Promise<RescheduleResult | null> {
    loading.value = true
    try {
      const res = await api.rescheduleAfterInterrupt(data)
      return res.data
    } catch (error) {
      console.error('Failed to reschedule after interrupt:', error)
      throw error
    } finally {
      loading.value = false
    }
  }

  async function getPrioritySuggestions(): Promise<PrioritySuggestion | null> {
    loading.value = true
    try {
      const res = await api.getPrioritySuggestions()
      return res.data
    } catch (error) {
      console.error('Failed to get priority suggestions:', error)
      throw error
    } finally {
      loading.value = false
    }
  }

  async function getDailyInsights(params?: {
    date?: string
    completed_pomodoros?: number
    total_focus_minutes?: number
    completed_tasks?: number
    total_interruptions?: number
    task_distribution?: string
  }): Promise<DailyInsights | null> {
    loading.value = true
    try {
      const res = await api.getDailyInsights(params)
      return res.data
    } catch (error) {
      console.error('Failed to get daily insights:', error)
      throw error
    } finally {
      loading.value = false
    }
  }

  return {
    // State
    configured,
    loading,
    lastClassification,
    // Actions
    checkStatus,
    classifyTask,
    classifyTasks,
    classifyTaskByText,
    generateSchedule,
    rescheduleAfterInterrupt,
    getPrioritySuggestions,
    getDailyInsights
  }
})
