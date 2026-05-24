import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAIStore } from '@/stores/ai'
import { api } from '@/api/client'

vi.mock('@/api/client', () => ({
  api: {
    getAIStatus: vi.fn(),
    classifyTask: vi.fn(),
    classifyTasks: vi.fn(),
    classifyTaskByText: vi.fn(),
    generateSchedule: vi.fn(),
    rescheduleAfterInterrupt: vi.fn(),
    getPrioritySuggestions: vi.fn(),
    getDailyInsights: vi.fn()
  }
}))

describe('AI Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('initial state', () => {
    it('initializes as not configured', () => {
      const store = useAIStore()
      expect(store.configured).toBe(false)
    })

    it('initializes with loading false', () => {
      const store = useAIStore()
      expect(store.loading).toBe(false)
    })

    it('initializes with no last classification', () => {
      const store = useAIStore()
      expect(store.lastClassification).toBeNull()
    })
  })

  describe('checkStatus', () => {
    it('should set configured to true when API returns configured', async () => {
      const store = useAIStore()
      ;(api.getAIStatus as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { configured: true }
      })

      const result = await store.checkStatus()

      expect(result).toBe(true)
      expect(store.configured).toBe(true)
    })

    it('should set configured to false when API returns not configured', async () => {
      const store = useAIStore()
      store.configured = true
      ;(api.getAIStatus as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { configured: false }
      })

      await store.checkStatus()

      expect(store.configured).toBe(false)
    })

    it('should set configured to false on error', async () => {
      const store = useAIStore()
      store.configured = true
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
      ;(api.getAIStatus as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Network'))

      const result = await store.checkStatus()

      expect(result).toBe(false)
      expect(store.configured).toBe(false)
      consoleSpy.mockRestore()
    })
  })

  describe('classifyTask', () => {
    it('should classify a single task and store result', async () => {
      const store = useAIStore()
      const classification = {
        task_id: 'task-1',
        important: true,
        urgent: false,
        quadrant: 2,
        reason: 'AI analysis'
      }
      ;(api.classifyTask as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: classification
      })

      const result = await store.classifyTask('task-1')

      expect(api.classifyTask).toHaveBeenCalledWith('task-1')
      expect(result).toEqual(classification)
      expect(store.lastClassification).toEqual(classification)
      expect(store.loading).toBe(false)
    })

    it('should propagate errors', async () => {
      const store = useAIStore()
      ;(api.classifyTask as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Failed'))

      await expect(store.classifyTask('task-1')).rejects.toThrow('Failed')
      expect(store.loading).toBe(false)
    })
  })

  describe('classifyTasks', () => {
    it('should classify multiple tasks', async () => {
      const store = useAIStore()
      const results = [
        { task_id: '1', important: true, urgent: true, quadrant: 1, reason: 'a' },
        { task_id: '2', important: false, urgent: false, quadrant: 4, reason: 'b' }
      ]
      ;(api.classifyTasks as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { results }
      })

      const result = await store.classifyTasks(['1', '2'])

      expect(api.classifyTasks).toHaveBeenCalledWith(['1', '2'])
      expect(result).toEqual(results)
      expect(store.loading).toBe(false)
    })

    it('should propagate errors', async () => {
      const store = useAIStore()
      ;(api.classifyTasks as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Failed'))

      await expect(store.classifyTasks(['1'])).rejects.toThrow('Failed')
      expect(store.loading).toBe(false)
    })
  })

  describe('generateSchedule', () => {
    it('should generate schedule and return events', async () => {
      const store = useAIStore()
      const events = [
        {
          id: 'e1',
          title: 'Generated',
          start: '2026-05-21T09:00:00Z',
          end: '2026-05-21T09:30:00Z',
          type: 'task' as const,
          status: 'planned' as const,
          color: '#3b82f6',
          allDay: false,
          editable: true
        }
      ]
      ;(api.generateSchedule as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { events }
      })

      const result = await store.generateSchedule('09:00', '18:00')

      expect(api.generateSchedule).toHaveBeenCalledWith('09:00', '18:00')
      expect(result).toEqual(events)
      expect(store.loading).toBe(false)
    })

    it('should propagate errors', async () => {
      const store = useAIStore()
      ;(api.generateSchedule as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Failed'))

      await expect(store.generateSchedule('09:00', '18:00')).rejects.toThrow('Failed')
      expect(store.loading).toBe(false)
    })
  })

  describe('rescheduleAfterInterrupt', () => {
    const rescheduleData = {
      task_id: 'task-1',
      completed_minutes: 10,
      planned_minutes: 25,
      interrupt_reason: 'meeting',
      work_end_time: '18:00'
    }

    const rescheduleResult = {
      adjusted_schedule: [
        {
          task_id: 'task-1',
          title: 'Interrupted Task',
          start_time: '2026-05-21T14:00:00Z',
          end_time: '2026-05-21T14:15:00Z',
          adjustment: 'shortened',
          reason: 'Remaining 15 min from original 25 min'
        },
        {
          task_id: 'task-2',
          title: 'Next Task',
          start_time: '2026-05-21T14:15:00Z',
          end_time: '2026-05-21T14:40:00Z',
          adjustment: 'postponed',
          reason: 'Previous task interrupted'
        }
      ],
      summary: 'Moved 2 tasks, shortened 1 task due to interruption'
    }

    it('should call reschedule API and return adjusted schedule', async () => {
      const store = useAIStore()
      ;(api.rescheduleAfterInterrupt as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: rescheduleResult
      })

      const result = await store.rescheduleAfterInterrupt(rescheduleData)

      expect(api.rescheduleAfterInterrupt).toHaveBeenCalledWith(rescheduleData)
      expect(result).toEqual(rescheduleResult)
      expect(result!.adjusted_schedule).toHaveLength(2)
      expect(result!.adjusted_schedule[0].adjustment).toBe('shortened')
      expect(result!.adjusted_schedule[1].adjustment).toBe('postponed')
      expect(store.loading).toBe(false)
    })

    it('should set loading true during reschedule request', async () => {
      const store = useAIStore()
      let resolveFn: (value: unknown) => void
      const promise = new Promise((resolve) => { resolveFn = resolve })
      ;(api.rescheduleAfterInterrupt as ReturnType<typeof vi.fn>).mockReturnValue(promise)

      const resultPromise = store.rescheduleAfterInterrupt(rescheduleData)
      expect(store.loading).toBe(true)

      resolveFn!({ data: rescheduleResult })
      await resultPromise
      expect(store.loading).toBe(false)
    })

    it('should propagate reschedule API errors', async () => {
      const store = useAIStore()
      ;(api.rescheduleAfterInterrupt as ReturnType<typeof vi.fn>).mockRejectedValue(
        new Error('AI service unavailable')
      )

      await expect(store.rescheduleAfterInterrupt(rescheduleData)).rejects.toThrow(
        'AI service unavailable'
      )
      expect(store.loading).toBe(false)
    })

    it('should handle minimal reschedule params (without optional fields)', async () => {
      const store = useAIStore()
      const result = {
        adjusted_schedule: [],
        summary: 'No adjustments needed'
      }
      ;(api.rescheduleAfterInterrupt as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: result
      })

      const response = await store.rescheduleAfterInterrupt({
        task_id: 'task-1',
        planned_minutes: 25,
        work_end_time: '18:00'
      })

      expect(api.rescheduleAfterInterrupt).toHaveBeenCalledWith({
        task_id: 'task-1',
        planned_minutes: 25,
        work_end_time: '18:00'
      })
      expect(response).toEqual(result)
    })
  })

  describe('classifyTaskByText', () => {
    it('should classify task text and return result', async () => {
      const store = useAIStore()
      const classification = {
        task_id: '',
        important: true,
        urgent: false,
        quadrant: 2,
        reason: 'Not urgent but important for long-term goals'
      }
      ;(api.classifyTaskByText as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: classification
      })

      const result = await store.classifyTaskByText('重构用户模块')

      expect(api.classifyTaskByText).toHaveBeenCalledWith('重构用户模块', undefined)
      expect(result).toEqual(classification)
      expect(store.lastClassification).toEqual(classification)
      expect(store.loading).toBe(false)
    })

    it('should accept optional description parameter', async () => {
      const store = useAIStore()
      ;(api.classifyTaskByText as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { task_id: '', important: false, urgent: true, quadrant: 3, reason: 'urgent' }
      })

      await store.classifyTaskByText('修复登录bug', '用户无法登录')
      expect(api.classifyTaskByText).toHaveBeenCalledWith('修复登录bug', '用户无法登录')
    })

    it('should propagate errors', async () => {
      const store = useAIStore()
      ;(api.classifyTaskByText as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Failed'))

      await expect(store.classifyTaskByText('test')).rejects.toThrow('Failed')
      expect(store.loading).toBe(false)
    })
  })

  describe('getDailyInsights', () => {
    const insights = {
      productivity_score: 78,
      peak_hours: '09:00-11:00',
      achievements: ['Completed 8 pomodoros', 'Finished 3 tasks'],
      suggestions: ['Try grouping similar tasks', 'Add buffer after meetings'],
      motivation: 'You are doing great! Keep the momentum.'
    }

    it('should fetch daily insights with full params', async () => {
      const store = useAIStore()
      ;(api.getDailyInsights as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: insights
      })

      const result = await store.getDailyInsights({
        date: '2026-05-21',
        completed_pomodoros: 8,
        total_focus_minutes: 200,
        completed_tasks: 3,
        total_interruptions: 2,
        task_distribution: '{"1":2,"2":1}'
      })

      expect(result).toEqual(insights)
      expect(result!.productivity_score).toBe(78)
      expect(result!.achievements).toHaveLength(2)
      expect(result!.suggestions).toHaveLength(2)
      expect(store.loading).toBe(false)
    })

    it('should handle no params (default behavior)', async () => {
      const store = useAIStore()
      ;(api.getDailyInsights as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: insights
      })

      await store.getDailyInsights()

      expect(api.getDailyInsights).toHaveBeenCalledWith(undefined)
      expect(store.loading).toBe(false)
    })

    it('should propagate errors', async () => {
      const store = useAIStore()
      ;(api.getDailyInsights as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Failed'))

      await expect(store.getDailyInsights()).rejects.toThrow('Failed')
      expect(store.loading).toBe(false)
    })
  })

  describe('getPrioritySuggestions', () => {
    it('should get priority suggestions', async () => {
      const store = useAIStore()
      const suggestion = { priority_order: ['task-1', 'task-2', 'task-3'] }
      ;(api.getPrioritySuggestions as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: suggestion
      })

      const result = await store.getPrioritySuggestions()

      expect(result).toEqual(suggestion)
      expect(store.loading).toBe(false)
    })

    it('should propagate errors', async () => {
      const store = useAIStore()
      ;(api.getPrioritySuggestions as ReturnType<typeof vi.fn>).mockRejectedValue(new Error('Failed'))

      await expect(store.getPrioritySuggestions()).rejects.toThrow('Failed')
      expect(store.loading).toBe(false)
    })
  })
})
