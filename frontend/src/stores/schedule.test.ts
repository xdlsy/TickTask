import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useScheduleStore } from '@/stores/schedule'
import { api } from '@/api/client'

// Mock the API client
vi.mock('@/api/client', () => ({
  api: {
    getSchedules: vi.fn(),
    getSchedule: vi.fn(),
    createSchedule: vi.fn(),
    updateSchedule: vi.fn(),
    deleteSchedule: vi.fn(),
    moveSchedule: vi.fn(),
    generateScheduleFromTasks: vi.fn()
  }
}))

describe('Schedule Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('State', () => {
    it('should have initial state', () => {
      const store = useScheduleStore()

      expect(store.events).toEqual([])
      expect(store.loading).toBe(false)
      expect(store.viewMode).toBe('week')
    })
  })

  describe('fetchSchedules', () => {
    it('should fetch schedules and update state', async () => {
      const mockEvents = [
        {
          id: '1',
          title: '测试日程',
          start: '2026-03-18T09:00:00Z',
          end: '2026-03-18T10:00:00Z',
          type: 'task',
          status: 'planned',
          color: '#3b82f6',
          allDay: false,
          editable: true
        }
      ]
      ;(api.getSchedules as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { events: mockEvents }
      })

      const store = useScheduleStore()
      await store.fetchSchedules('2026-03-01', '2026-03-31')

      expect(api.getSchedules).toHaveBeenCalledWith('2026-03-01', '2026-03-31')
      expect(store.events).toEqual(mockEvents)
      expect(store.loading).toBe(false)
    })

    it('should handle fetch error', async () => {
      const error = new Error('Network Error')
      ;(api.getSchedules as ReturnType<typeof vi.fn>).mockRejectedValue(error)

      const store = useScheduleStore()

      // fetchSchedules catches errors internally and doesn't re-throw
      await store.fetchSchedules()
      expect(store.loading).toBe(false)
      expect(store.events).toEqual([])
    })

    it('should set loading state during fetch', async () => {
      let resolveFn: (value: unknown) => void
      const promise = new Promise((resolve) => {
        resolveFn = resolve
      })
      ;(api.getSchedules as ReturnType<typeof vi.fn>).mockReturnValue(promise)

      const store = useScheduleStore()
      const fetchPromise = store.fetchSchedules()

      expect(store.loading).toBe(true)

      resolveFn!({ data: { events: [] } })
      await fetchPromise

      expect(store.loading).toBe(false)
    })
  })

  describe('createSchedule', () => {
    it('should create schedule and add to events', async () => {
      const newEvent = {
        id: 'new-1',
        title: '新建日程',
        start: '2026-03-18T09:00:00Z',
        end: '2026-03-18T10:00:00Z',
        type: 'task',
        status: 'planned',
        color: '#3b82f6',
        allDay: false,
        editable: true
      }
      ;(api.createSchedule as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: newEvent
      })

      const store = useScheduleStore()
      const createData = {
        title: '新建日程',
        start_time: '2026-03-18T09:00:00Z',
        end_time: '2026-03-18T10:00:00Z',
        type: 'task' as const
      }

      const result = await store.createSchedule(createData)

      expect(api.createSchedule).toHaveBeenCalledWith(createData)
      expect(result).toEqual(newEvent)
    })

    it('should handle create error', async () => {
      const error = new Error('Bad Request')
      ;(api.createSchedule as ReturnType<typeof vi.fn>).mockRejectedValue(error)

      const store = useScheduleStore()
      const createData = {
        title: '测试',
        start_time: '2026-03-18T09:00:00Z',
        end_time: '2026-03-18T10:00:00Z',
        type: 'task' as const
      }

      await expect(store.createSchedule(createData)).rejects.toThrow('Bad Request')
    })
  })

  describe('updateSchedule', () => {
    it('should update schedule', async () => {
      ;(api.updateSchedule as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { message: 'updated' }
      })

      const store = useScheduleStore()
      // Add an existing event
      store.events = [
        {
          id: '1',
          title: '旧标题',
          start: '2026-03-18T09:00:00Z',
          end: '2026-03-18T10:00:00Z',
          type: 'task',
          status: 'planned',
          color: '#3b82f6',
          allDay: false,
          editable: true
        }
      ]

      const updateData = { title: '新标题' }
      await store.updateSchedule('1', updateData)

      expect(api.updateSchedule).toHaveBeenCalledWith('1', updateData)
    })

    it('should handle update error', async () => {
      const error = new Error('Not Found')
      ;(api.updateSchedule as ReturnType<typeof vi.fn>).mockRejectedValue(error)

      const store = useScheduleStore()
      const updateData = { title: '新标题' }

      await expect(store.updateSchedule('non-existent', updateData)).rejects.toThrow('Not Found')
    })
  })

  describe('deleteSchedule', () => {
    it('should delete schedule and remove from events', async () => {
      ;(api.deleteSchedule as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { message: 'deleted' }
      })

      const store = useScheduleStore()
      store.events = [
        {
          id: '1',
          title: '要删除的日程',
          start: '2026-03-18T09:00:00Z',
          end: '2026-03-18T10:00:00Z',
          type: 'task',
          status: 'planned',
          color: '#3b82f6',
          allDay: false,
          editable: true
        },
        {
          id: '2',
          title: '保留的日程',
          start: '2026-03-18T11:00:00Z',
          end: '2026-03-18T12:00:00Z',
          type: 'task',
          status: 'planned',
          color: '#3b82f6',
          allDay: false,
          editable: true
        }
      ]

      await store.deleteSchedule('1')

      expect(api.deleteSchedule).toHaveBeenCalledWith('1')
      expect(store.events).toHaveLength(1)
      expect(store.events[0].id).toBe('2')
    })

    it('should handle delete error', async () => {
      const error = new Error('Not Found')
      ;(api.deleteSchedule as ReturnType<typeof vi.fn>).mockRejectedValue(error)

      const store = useScheduleStore()

      await expect(store.deleteSchedule('non-existent')).rejects.toThrow('Not Found')
    })
  })

  describe('moveSchedule', () => {
    it('should move schedule to new time', async () => {
      ;(api.moveSchedule as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { message: 'moved' }
      })

      const store = useScheduleStore()
      const moveData = {
        start_time: '2026-03-18T14:00:00Z',
        end_time: '2026-03-18T15:00:00Z'
      }

      await store.moveSchedule('1', moveData)

      expect(api.moveSchedule).toHaveBeenCalledWith('1', moveData)
    })
  })

  describe('generateSchedule', () => {
    it('should replace events and update store', async () => {
      const generatedEvents = [
        {
          id: 'gen-1',
          title: '生成的日程',
          start: '2026-03-18T09:00:00Z',
          end: '2026-03-18T09:30:00Z',
          type: 'task',
          status: 'planned',
          color: '#3b82f6',
          allDay: false,
          editable: true,
          task_id: 'task-1'
        }
      ]
      ;(api.generateScheduleFromTasks as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { events: generatedEvents }
      })

      const store = useScheduleStore()
      await store.generateSchedule('09:00', '18:00')

      expect(api.generateScheduleFromTasks).toHaveBeenCalledWith('09:00', '18:00')
      expect(store.events).toEqual(generatedEvents)
    })

    it('should handle generate error', async () => {
      const error = new Error('AI not configured')
      ;(api.generateScheduleFromTasks as ReturnType<typeof vi.fn>).mockRejectedValue(error)

      const store = useScheduleStore()

      await expect(store.generateSchedule()).rejects.toThrow('AI not configured')
    })
  })

  describe('setViewMode', () => {
    it('should change view mode', () => {
      const store = useScheduleStore()

      expect(store.viewMode).toBe('week')

      store.setViewMode('day')
      expect(store.viewMode).toBe('day')

      store.setViewMode('month')
      expect(store.viewMode).toBe('month')
    })
  })

  describe('navigation', () => {
    it('should navigate to previous week', () => {
      const store = useScheduleStore()
      const initialTime = store.currentDate.getTime()

      store.goToPrevious()

      const diffMs = initialTime - store.currentDate.getTime()
      const diffDays = diffMs / (1000 * 60 * 60 * 24)
      expect(diffDays).toBeCloseTo(7, 0)
    })

    it('should navigate to next week', () => {
      const store = useScheduleStore()
      const initialTime = store.currentDate.getTime()

      store.goToNext()

      const diffMs = store.currentDate.getTime() - initialTime
      const diffDays = diffMs / (1000 * 60 * 60 * 24)
      expect(diffDays).toBeCloseTo(7, 0)
    })

    it('should navigate to today', () => {
      const store = useScheduleStore()

      // First navigate away from today
      store.goToPrevious()
      store.goToPrevious()

      // Then go back to today
      store.goToToday()

      const today = new Date()
      expect(store.currentDate.toDateString()).toBe(today.toDateString())
    })

    it('should navigate by day in day view', () => {
      const store = useScheduleStore()
      store.setViewMode('day')

      const initialDate = new Date(store.currentDate)
      store.goToNext()

      const diffDays = Math.abs(store.currentDate.getDate() - initialDate.getDate())
      expect(diffDays).toBe(1)
    })

    it('should navigate by month in month view', () => {
      const store = useScheduleStore()
      store.setViewMode('month')

      const initialMonth = store.currentDate.getMonth()
      store.goToNext()

      const newMonth = store.currentDate.getMonth()
      // 月份应该增加1（或从11月跳到0月）
      expect((newMonth - initialMonth + 12) % 12).toBe(1)
    })

    it('should navigate previous by day in day view', () => {
      const store = useScheduleStore()
      store.setViewMode('day')

      const initialDate = new Date(store.currentDate)
      store.goToPrevious()

      const diffDays = Math.abs(store.currentDate.getDate() - initialDate.getDate())
      expect(diffDays).toBe(1)
    })

    it('should navigate previous by month in month view', () => {
      const store = useScheduleStore()
      store.setViewMode('month')

      const initialMonth = store.currentDate.getMonth()
      store.goToPrevious()

      const newMonth = store.currentDate.getMonth()
      // 月份应该减少1
      expect((initialMonth - newMonth + 12) % 12).toBe(1)
    })
  })

  describe('setCurrentDate', () => {
    it('should set current date', () => {
      const store = useScheduleStore()
      const newDate = new Date('2026-03-24')

      store.setCurrentDate(newDate)

      expect(store.currentDate.toDateString()).toBe(newDate.toDateString())
    })

    it('should not affect other state when setting date', () => {
      const store = useScheduleStore()
      store.setViewMode('month')

      store.setCurrentDate(new Date('2026-03-24'))

      expect(store.viewMode).toBe('month')
    })
  })

  describe('updateSchedule', () => {
    it('should update local event data after update', async () => {
      ;(api.updateSchedule as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { message: 'updated' }
      })

      const store = useScheduleStore()
      store.events = [
        {
          id: '1',
          title: '旧标题',
          start: '2026-03-18T09:00:00Z',
          end: '2026-03-18T10:00:00Z',
          type: 'task',
          status: 'planned',
          color: '#3b82f6',
          allDay: false,
          editable: true
        }
      ]

      await store.updateSchedule('1', {
        title: '新标题',
        color: '#ef4444'
      })

      expect(store.events[0].title).toBe('新标题')
      expect(store.events[0].color).toBe('#ef4444')
    })
  })

  describe('moveSchedule', () => {
    it('should update local event times after move', async () => {
      ;(api.moveSchedule as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { message: 'moved' }
      })

      const store = useScheduleStore()
      store.events = [
        {
          id: '1',
          title: '测试日程',
          start: '2026-03-18T09:00:00Z',
          end: '2026-03-18T10:00:00Z',
          type: 'task',
          status: 'planned',
          color: '#3b82f6',
          allDay: false,
          editable: true
        }
      ]

      await store.moveSchedule('1', {
        start_time: '2026-03-18T14:00:00Z',
        end_time: '2026-03-18T15:00:00Z'
      })

      expect(store.events[0].start).toBe('2026-03-18T14:00:00Z')
      expect(store.events[0].end).toBe('2026-03-18T15:00:00Z')
    })
  })

  describe('generateSchedule', () => {
    it('should deduplicate by task_id and preserve custom events', async () => {
      const existingTaskEvent = {
        id: 'old-task-1',
        title: '旧任务日程',
        start: '2026-03-18T09:00:00Z',
        end: '2026-03-18T10:00:00Z',
        type: 'task',
        status: 'planned',
        color: '#3b82f6',
        allDay: false,
        editable: true,
        task_id: 'task-1'
      }
      const customEvent = {
        id: 'custom-1',
        title: '手动创建的日程',
        start: '2026-03-18T14:00:00Z',
        end: '2026-03-18T15:00:00Z',
        type: 'custom',
        status: 'planned',
        color: '#9C9893',
        allDay: false,
        editable: true
      }

      const freshEvents = [
        {
          id: 'gen-1',
          title: '新生成的任务日程',
          start: '2026-03-18T11:00:00Z',
          end: '2026-03-18T12:00:00Z',
          type: 'task',
          status: 'planned',
          color: '#3b82f6',
          allDay: false,
          editable: true,
          task_id: 'task-1'
        }
      ]

      ;(api.generateScheduleFromTasks as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { events: freshEvents }
      })

      const store = useScheduleStore()
      store.events = [existingTaskEvent, customEvent]

      await store.generateSchedule('09:00', '18:00')

      // 旧的任务日程被新日程替换，手动日程保留
      expect(store.events.length).toBe(2)
      expect(store.events.find(e => e.id === 'gen-1')).toBeTruthy()
      expect(store.events.find(e => e.id === 'custom-1')).toBeTruthy()
      expect(store.events.find(e => e.id === 'old-task-1')).toBeFalsy()
    })

    it('should set loading state during generation', async () => {
      let genResolveFn: (value: unknown) => void
      const genPromise = new Promise((resolve) => {
        genResolveFn = resolve
      })
      ;(api.generateScheduleFromTasks as ReturnType<typeof vi.fn>).mockReturnValue(genPromise)

      const store = useScheduleStore()
      const resultPromise = store.generateSchedule('09:00', '18:00')

      expect(store.loading).toBe(true)

      genResolveFn!({ data: { events: [] } })
      await resultPromise

      expect(store.loading).toBe(false)
    })
  })
})