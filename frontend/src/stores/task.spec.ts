import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useTaskStore } from '@/stores/task'
import type { Task, Quadrant } from '@/types'

// Mock API client
const mockTasks: Task[] = [
  {
    id: 'task-1',
    title: '重要且紧急',
    description: '描述1',
    quadrant: 1,
    is_important: true,
    is_urgent: true,
    status: 'todo',
    estimated_time: 30,
    deadline: null,
    tags: [],
    order: 0,
    created_at: '2024-03-10T08:00:00Z',
    updated_at: '2024-03-10T08:00:00Z',
    completed_at: null
  },
  {
    id: 'task-2',
    title: '重要不紧急',
    description: '描述2',
    quadrant: 2,
    is_important: true,
    is_urgent: false,
    status: 'todo',
    estimated_time: 60,
    deadline: null,
    tags: [],
    order: 0,
    created_at: '2024-03-10T08:00:00Z',
    updated_at: '2024-03-10T08:00:00Z',
    completed_at: null
  }
]

describe('Task Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('initializes with empty tasks', () => {
    const store = useTaskStore()
    expect(store.tasks).toEqual([])
  })

  it('initializes with empty quadrants', () => {
    const store = useTaskStore()
    expect(store.tasksByQuadrant).toEqual({
      1: [],
      2: [],
      3: [],
      4: []
    })
  })

  it('has loading state', () => {
    const store = useTaskStore()
    expect(store.loading).toBe(false)
  })

  it('getQuadrantName returns correct names', () => {
    const store = useTaskStore()

    expect(store.getQuadrantName(1)).toBe('重要且紧急')
    expect(store.getQuadrantName(2)).toBe('重要不紧急')
    expect(store.getQuadrantName(3)).toBe('紧急不重要')
    expect(store.getQuadrantName(4)).toBe('不重要不紧急')
  })

  it('getQuadrantColor returns correct colors', () => {
    const store = useTaskStore()

    expect(store.getQuadrantColor(1)).toBe('#ef4444') // red
    expect(store.getQuadrantColor(2)).toBe('#f59e0b') // yellow
    expect(store.getQuadrantColor(3)).toBe('#3b82f6') // blue
    expect(store.getQuadrantColor(4)).toBe('#6b7280') // gray
  })
})