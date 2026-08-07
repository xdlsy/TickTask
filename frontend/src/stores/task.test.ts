import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useTaskStore } from '@/stores/task'
import { api } from '@/api/client'
import type { Task } from '@/types'

const mockTask: Task = {
  id: 'task-1',
  title: '测试任务',
  description: '描述',
  quadrant: 1,
  is_important: true,
  is_urgent: true,
  status: 'todo',
  estimated_time: 1800,
  deadline: null,
  tags: [],
  order: 0,
  created_at: '2026-05-21T09:00:00Z',
  updated_at: '2026-05-21T09:00:00Z',
  completed_at: null
}

vi.mock('@/api/client', () => ({
  api: {
    getTasks: vi.fn(),
    getTasksByQuadrant: vi.fn(),
    createTask: vi.fn(),
    updateTask: vi.fn(),
    deleteTask: vi.fn(),
    moveTask: vi.fn()
  }
}))

describe('Task Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('initial state', () => {
    it('should have correct initial state', () => {
      const store = useTaskStore()
      expect(store.tasks).toEqual([])
      expect(store.loading).toBe(false)
      expect(store.tasksByQuadrant).toEqual({ 1: [], 2: [], 3: [], 4: [] })
    })
  })

  describe('getQuadrantName', () => {
    it('should return correct name for quadrant 1', () => {
      const store = useTaskStore()
      expect(store.getQuadrantName(1)).toBe('重要且紧急')
    })

    it('should return correct name for quadrant 4', () => {
      const store = useTaskStore()
      expect(store.getQuadrantName(4)).toBe('不重要不紧急')
    })
  })

  describe('getQuadrantColor', () => {
    it('should return correct color for each quadrant', () => {
      const store = useTaskStore()
      expect(store.getQuadrantColor(1)).toBe('#D86F54')
      expect(store.getQuadrantColor(2)).toBe('#E6A23C')
      expect(store.getQuadrantColor(3)).toBe('#7FA8C0')
      expect(store.getQuadrantColor(4)).toBe('#8A8273')
    })
  })

  describe('fetchTasks', () => {
    it('should fetch tasks and update state', async () => {
      const store = useTaskStore()
      ;(api.getTasks as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: [mockTask]
      })

      await store.fetchTasks()

      expect(store.tasks).toHaveLength(1)
      expect(store.tasks[0].id).toBe('task-1')
      expect(store.loading).toBe(false)
    })

    it('should set loading state during fetch', async () => {
      let resolveFn: (value: unknown) => void
      const promise = new Promise((resolve) => {
        resolveFn = resolve
      })
      ;(api.getTasks as ReturnType<typeof vi.fn>).mockReturnValue(promise)

      const store = useTaskStore()
      const fetchPromise = store.fetchTasks()

      expect(store.loading).toBe(true)

      resolveFn!({ data: [] })
      await fetchPromise

      expect(store.loading).toBe(false)
    })

    it('should handle fetch error', async () => {
      const store = useTaskStore()
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
      ;(api.getTasks as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Network'))

      await store.fetchTasks()

      expect(store.loading).toBe(false)
      expect(consoleSpy).toHaveBeenCalled()
      consoleSpy.mockRestore()
    })
  })

  describe('fetchTasksByQuadrant', () => {
    it('should fetch and organize tasks by quadrant', async () => {
      const store = useTaskStore()
      ;(api.getTasksByQuadrant as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { 1: [mockTask], 2: [], 3: [], 4: [] }
      })

      await store.fetchTasksByQuadrant()

      expect(store.tasksByQuadrant[1]).toHaveLength(1)
      expect(store.loading).toBe(false)
    })

    it('should fill in missing quadrants', async () => {
      const store = useTaskStore()
      ;(api.getTasksByQuadrant as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { 1: [mockTask] }
      })

      await store.fetchTasksByQuadrant()

      expect(store.tasksByQuadrant[2]).toEqual([])
      expect(store.tasksByQuadrant[3]).toEqual([])
      expect(store.tasksByQuadrant[4]).toEqual([])
    })

    it('should handle error', async () => {
      const store = useTaskStore()
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
      ;(api.getTasksByQuadrant as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Network'))

      await store.fetchTasksByQuadrant()

      expect(store.loading).toBe(false)
      consoleSpy.mockRestore()
    })
  })

  describe('createTask', () => {
    it('should create task and prepend to list', async () => {
      const store = useTaskStore()
      store.tasks = [{ ...mockTask, id: 'existing' }]
      ;(api.createTask as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { ...mockTask, id: 'new-task' }
      })
      ;(api.getTasksByQuadrant as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { 1: [], 2: [], 3: [], 4: [] }
      })

      const result = await store.createTask({
        title: '新任务',
        quadrant: 1
      })

      expect(result.id).toBe('new-task')
      expect(store.tasks[0].id).toBe('new-task')
      expect(store.tasks).toHaveLength(2)
    })

    it('should propagate create error', async () => {
      const store = useTaskStore()
      ;(api.createTask as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Bad Request'))

      await expect(store.createTask({ title: 'test', quadrant: 2 })).rejects.toThrow('Bad Request')
    })
  })

  describe('updateTask', () => {
    it('should update task and sync local state', async () => {
      const store = useTaskStore()
      store.tasks = [{ ...mockTask }]
      ;(api.updateTask as ReturnType<typeof vi.fn>).mockResolvedValue({ data: {} })
      ;(api.getTasksByQuadrant as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { 1: [], 2: [], 3: [], 4: [] }
      })

      await store.updateTask('task-1', { title: 'Updated', status: 'completed' })

      expect(api.updateTask).toHaveBeenCalledWith('task-1', {
        title: 'Updated',
        status: 'completed'
      })
      expect(store.tasks[0].title).toBe('Updated')
    })

    it('should not modify tasks if ID not found', async () => {
      const store = useTaskStore()
      store.tasks = [{ ...mockTask }]
      ;(api.updateTask as ReturnType<typeof vi.fn>).mockResolvedValue({ data: {} })
      ;(api.getTasksByQuadrant as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { 1: [], 2: [], 3: [], 4: [] }
      })

      await store.updateTask('non-existent', { title: 'X' })

      expect(store.tasks[0].title).toBe('测试任务')
    })

    it('should propagate update error', async () => {
      const store = useTaskStore()
      ;(api.updateTask as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Not Found'))

      await expect(store.updateTask('x', { title: 'Y' })).rejects.toThrow('Not Found')
    })
  })

  describe('deleteTask', () => {
    it('should delete task and update local state', async () => {
      const store = useTaskStore()
      store.tasks = [{ ...mockTask }, { ...mockTask, id: 'task-2' }]
      ;(api.deleteTask as ReturnType<typeof vi.fn>).mockResolvedValue({ data: {} })
      ;(api.getTasksByQuadrant as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { 1: [], 2: [], 3: [], 4: [] }
      })

      await store.deleteTask('task-1')

      expect(api.deleteTask).toHaveBeenCalledWith('task-1')
      expect(store.tasks).toHaveLength(1)
      expect(store.tasks[0].id).toBe('task-2')
    })

    it('should propagate delete error', async () => {
      const store = useTaskStore()
      ;(api.deleteTask as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Not Found'))

      await expect(store.deleteTask('x')).rejects.toThrow('Not Found')
    })
  })

  describe('moveTask', () => {
    it('should move task to target quadrant', async () => {
      const store = useTaskStore()
      ;(api.moveTask as ReturnType<typeof vi.fn>).mockResolvedValue({ data: {} })
      ;(api.getTasksByQuadrant as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { 1: [], 2: [], 3: [], 4: [] }
      })

      await store.moveTask('task-1', 2)

      expect(api.moveTask).toHaveBeenCalledWith('task-1', 2)
    })

    it('should propagate move error', async () => {
      const store = useTaskStore()
      ;(api.moveTask as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Not Found'))

      await expect(store.moveTask('x', 3)).rejects.toThrow('Not Found')
    })
  })

  describe('markCompleted', () => {
    it('should mark task as completed via updateTask', async () => {
      const store = useTaskStore()
      store.tasks = [{ ...mockTask }]
      ;(api.updateTask as ReturnType<typeof vi.fn>).mockResolvedValue({ data: {} })
      ;(api.getTasksByQuadrant as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { 1: [], 2: [], 3: [], 4: [] }
      })

      await store.markCompleted('task-1')

      expect(api.updateTask).toHaveBeenCalledWith('task-1', { status: 'completed' })
    })
  })
})
