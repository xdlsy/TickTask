import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Task, TaskResponse, Quadrant } from '@/types'
import { api } from '@/api/client'
import { QUADRANT_INFO } from '@/types'

export const useTaskStore = defineStore('task', () => {
  // State
  const tasks = ref<TaskResponse[]>([])
  const tasksByQuadrant = ref<Record<Quadrant, TaskResponse[]>>({
    1: [],
    2: [],
    3: [],
    4: []
  })
  const loading = ref(false)

  // Actions
  async function fetchTasks() {
    loading.value = true
    try {
      const res = await api.getTasks()
      tasks.value = res.data
    } catch (error) {
      console.error('Failed to fetch tasks:', error)
    } finally {
      loading.value = false
    }
  }

  async function fetchTasksByQuadrant() {
    loading.value = true
    try {
      const res = await api.getTasksByQuadrant()
      tasksByQuadrant.value = res.data as Record<Quadrant, TaskResponse[]>
      // 确保四个象限都存在
      for (const q of [1, 2, 3, 4] as Quadrant[]) {
        if (!tasksByQuadrant.value[q]) {
          tasksByQuadrant.value[q] = []
        }
      }
    } catch (error) {
      console.error('Failed to fetch tasks by quadrant:', error)
    } finally {
      loading.value = false
    }
  }

  async function createTask(data: {
    title: string
    description?: string
    quadrant: Quadrant
    is_important?: boolean
    is_urgent?: boolean
    estimated_time?: number
    deadline?: string
    start_date?: string | null
    due_date?: string | null
    is_recurring?: boolean
    recurrence_pattern?: string
    preferred_start_time?: string | null
    preferred_end_time?: string | null
    tags?: string[]
  }) {
    try {
      const res = await api.createTask(data)
      tasks.value.unshift({ ...res.data, planned_pomodoros: 0, completed_pomodoros: 0, pomodoro_status: 'not_started' })
      await fetchTasksByQuadrant()
      return res.data
    } catch (error) {
      console.error('Failed to create task:', error)
      throw error
    }
  }

  async function updateTask(id: string, data: Partial<Task | TaskResponse>) {
    try {
      await api.updateTask(id, data)
      // 更新本地状态
      const index = tasks.value.findIndex(t => t.id === id)
      if (index !== -1) {
        tasks.value[index] = { ...tasks.value[index], ...data }
      }
      await fetchTasksByQuadrant()
    } catch (error) {
      console.error('Failed to update task:', error)
      throw error
    }
  }

  async function deleteTask(id: string) {
    try {
      await api.deleteTask(id)
      tasks.value = tasks.value.filter(t => t.id !== id)
      await fetchTasksByQuadrant()
    } catch (error) {
      console.error('Failed to delete task:', error)
      throw error
    }
  }

  async function moveTask(id: string, targetQuadrant: Quadrant) {
    try {
      await api.moveTask(id, targetQuadrant)
      await fetchTasksByQuadrant()
    } catch (error) {
      console.error('Failed to move task:', error)
      throw error
    }
  }

  async function markCompleted(id: string) {
    return updateTask(id, { status: 'completed' })
  }

  function getQuadrantName(quadrant: Quadrant): string {
    return QUADRANT_INFO[quadrant].name
  }

  function getQuadrantColor(quadrant: Quadrant): string {
    return QUADRANT_INFO[quadrant].color
  }

  return {
    // State
    tasks,
    tasksByQuadrant,
    loading,
    // Actions
    fetchTasks,
    fetchTasksByQuadrant,
    createTask,
    updateTask,
    deleteTask,
    moveTask,
    markCompleted,
    // Helpers
    getQuadrantName,
    getQuadrantColor
  }
})
