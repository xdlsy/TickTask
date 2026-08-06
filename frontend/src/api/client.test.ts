import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import axios from 'axios'
import { api } from '@/api/client'

// Mock axios
vi.mock('axios', () => {
  const mockAxios = vi.fn()
  mockAxios.create = vi.fn(() => mockAxios)
  mockAxios.get = vi.fn()
  mockAxios.post = vi.fn()
  mockAxios.put = vi.fn()
  mockAxios.delete = vi.fn()
  mockAxios.patch = vi.fn()
  mockAxios.interceptors = {
    request: { use: vi.fn() },
    response: { use: vi.fn() }
  }
  return { default: mockAxios }
})

describe('Schedule API', () => {
  let mockAxios: ReturnType<typeof axios.create>

  beforeEach(() => {
    vi.clearAllMocks()
    mockAxios = axios.create()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('getSchedules', () => {
    it('should call GET /schedules without parameters', async () => {
      const mockResponse = {
        data: {
          events: [
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
        }
      }
      ;(mockAxios.get as ReturnType<typeof vi.fn>).mockResolvedValue(mockResponse)

      const result = await api.getSchedules()

      expect(mockAxios.get).toHaveBeenCalledWith('/schedules', { params: { start: undefined, end: undefined } })
      expect(result.data.events).toHaveLength(1)
      expect(result.data.events[0].title).toBe('测试日程')
    })

    it('should call GET /schedules with date range parameters', async () => {
      const mockResponse = {
        data: {
          events: []
        }
      }
      ;(mockAxios.get as ReturnType<typeof vi.fn>).mockResolvedValue(mockResponse)

      await api.getSchedules('2026-03-01', '2026-03-31')

      expect(mockAxios.get).toHaveBeenCalledWith('/schedules', {
        params: { start: '2026-03-01', end: '2026-03-31' }
      })
    })
  })

  describe('getSchedule', () => {
    it('should call GET /schedules/:id', async () => {
      const mockResponse = {
        data: {
          id: 'schedule-1',
          title: '单个日程',
          start: '2026-03-18T09:00:00Z',
          end: '2026-03-18T10:00:00Z',
          type: 'task',
          status: 'planned',
          color: '#3b82f6',
          allDay: false,
          editable: true
        }
      }
      ;(mockAxios.get as ReturnType<typeof vi.fn>).mockResolvedValue(mockResponse)

      const result = await api.getSchedule('schedule-1')

      expect(mockAxios.get).toHaveBeenCalledWith('/schedules/schedule-1')
      expect(result.data.id).toBe('schedule-1')
    })
  })

  describe('createSchedule', () => {
    it('should call POST /schedules with schedule data', async () => {
      const mockResponse = {
        data: {
          id: 'new-schedule',
          title: '新建日程',
          start_time: '2026-03-18T09:00:00Z',
          end_time: '2026-03-18T10:00:00Z',
          type: 'task',
          status: 'planned',
          color: '#3b82f6'
        }
      }
      ;(mockAxios.post as ReturnType<typeof vi.fn>).mockResolvedValue(mockResponse)

      const createData = {
        title: '新建日程',
        start_time: '2026-03-18T09:00:00Z',
        end_time: '2026-03-18T10:00:00Z',
        type: 'task' as const,
        color: '#3b82f6'
      }

      const result = await api.createSchedule(createData)

      expect(mockAxios.post).toHaveBeenCalledWith('/schedules', createData)
      expect(result.data.id).toBe('new-schedule')
    })

    it('should create schedule with task_id', async () => {
      const mockResponse = {
        data: {
          id: 'new-schedule',
          task_id: 'task-1',
          title: '关联任务的日程',
          start_time: '2026-03-18T09:00:00Z',
          end_time: '2026-03-18T10:00:00Z',
          type: 'task',
          status: 'planned',
          color: '#3b82f6'
        }
      }
      ;(mockAxios.post as ReturnType<typeof vi.fn>).mockResolvedValue(mockResponse)

      const createData = {
        task_id: 'task-1',
        title: '关联任务的日程',
        start_time: '2026-03-18T09:00:00Z',
        end_time: '2026-03-18T10:00:00Z',
        type: 'task' as const
      }

      await api.createSchedule(createData)

      expect(mockAxios.post).toHaveBeenCalledWith('/schedules', createData)
    })
  })

  describe('updateSchedule', () => {
    it('should call PUT /schedules/:id with update data', async () => {
      const mockResponse = {
        data: { message: 'schedule updated' }
      }
      ;(mockAxios.put as ReturnType<typeof vi.fn>).mockResolvedValue(mockResponse)

      const updateData = {
        title: '更新后的标题',
        color: '#ef4444'
      }

      await api.updateSchedule('schedule-1', updateData)

      expect(mockAxios.put).toHaveBeenCalledWith('/schedules/schedule-1', updateData)
    })

    it('should update schedule time', async () => {
      const mockResponse = {
        data: { message: 'schedule updated' }
      }
      ;(mockAxios.put as ReturnType<typeof vi.fn>).mockResolvedValue(mockResponse)

      const updateData = {
        start_time: '2026-03-18T14:00:00Z',
        end_time: '2026-03-18T15:00:00Z'
      }

      await api.updateSchedule('schedule-1', updateData)

      expect(mockAxios.put).toHaveBeenCalledWith('/schedules/schedule-1', updateData)
    })
  })

  describe('deleteSchedule', () => {
    it('should call DELETE /schedules/:id', async () => {
      const mockResponse = {
        data: { message: 'schedule deleted' }
      }
      ;(mockAxios.delete as ReturnType<typeof vi.fn>).mockResolvedValue(mockResponse)

      await api.deleteSchedule('schedule-1')

      expect(mockAxios.delete).toHaveBeenCalledWith('/schedules/schedule-1')
    })
  })

  describe('moveSchedule', () => {
    it('should call PUT /schedules/:id/move with new times', async () => {
      const mockResponse = {
        data: { message: 'schedule moved' }
      }
      ;(mockAxios.put as ReturnType<typeof vi.fn>).mockResolvedValue(mockResponse)

      const moveData = {
        start_time: '2026-03-18T14:00:00Z',
        end_time: '2026-03-18T15:00:00Z'
      }

      await api.moveSchedule('schedule-1', moveData)

      expect(mockAxios.put).toHaveBeenCalledWith('/schedules/schedule-1/move', moveData)
    })
  })

  describe('generateScheduleFromTasks', () => {
    it('should call POST /schedules/generate with time range', async () => {
      const mockResponse = {
        data: {
          events: [
            {
              id: 'gen-1',
              title: '生成的日程1',
              start: '2026-03-18T09:00:00Z',
              end: '2026-03-18T09:30:00Z',
              type: 'task',
              status: 'planned',
              color: '#3b82f6',
              allDay: false,
              editable: true
            }
          ]
        }
      }
      ;(mockAxios.post as ReturnType<typeof vi.fn>).mockResolvedValue(mockResponse)

      const result = await api.generateScheduleFromTasks('09:00', '18:00')

      expect(mockAxios.post).toHaveBeenCalledWith('/schedules/generate', {
        start_time: '09:00',
        end_time: '18:00'
      }, { timeout: 360000 })
      expect(result.data.events).toHaveLength(1)
    })

    it('should call POST /schedules/generate without parameters', async () => {
      const mockResponse = {
        data: { events: [] }
      }
      ;(mockAxios.post as ReturnType<typeof vi.fn>).mockResolvedValue(mockResponse)

      await api.generateScheduleFromTasks()

      expect(mockAxios.post).toHaveBeenCalledWith('/schedules/generate', {
        start_time: undefined,
        end_time: undefined
      }, { timeout: 360000 })
    })
  })
})

describe('Schedule API Error Handling', () => {
  let mockAxios: ReturnType<typeof axios.create>

  beforeEach(() => {
    vi.clearAllMocks()
    mockAxios = axios.create()
  })

  it('should handle 404 error for non-existent schedule', async () => {
    const error = new Error('Request failed with status code 404')
    ;(mockAxios.get as ReturnType<typeof vi.fn>).mockRejectedValue(error)

    await expect(api.getSchedule('non-existent-id')).rejects.toThrow('Request failed with status code 404')
  })

  it('should handle 400 error for invalid create data', async () => {
    const error = new Error('Request failed with status code 400')
    ;(mockAxios.post as ReturnType<typeof vi.fn>).mockRejectedValue(error)

    const invalidData = {
      title: '',
      start_time: '',
      end_time: '',
      type: 'task' as const
    }

    await expect(api.createSchedule(invalidData)).rejects.toThrow('Request failed with status code 400')
  })

  it('should handle network error', async () => {
    const error = new Error('Network Error')
    ;(mockAxios.get as ReturnType<typeof vi.fn>).mockRejectedValue(error)

    await expect(api.getSchedules()).rejects.toThrow('Network Error')
  })
})

describe('Data Import/Export API', () => {
  let mockAxios: ReturnType<typeof axios.create>

  beforeEach(() => {
    vi.clearAllMocks()
    mockAxios = axios.create()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('exportData', () => {
    it('should call GET /data/export with blob responseType', async () => {
      const mockResponse = { data: new Blob(['{}'], { type: 'application/json' }) }
      ;(mockAxios.get as ReturnType<typeof vi.fn>).mockResolvedValue(mockResponse)

      await api.exportData()

      expect(mockAxios.get).toHaveBeenCalledWith('/data/export', { responseType: 'blob' })
    })
  })

  describe('previewImport', () => {
    it('should POST /data/import/preview with FormData + multipart header', async () => {
      const mockResponse = {
        data: {
          schema_version: 1,
          schema_warning: '',
          modules: {}
        }
      }
      ;(mockAxios.post as ReturnType<typeof vi.fn>).mockResolvedValue(mockResponse)

      const file = new File(['{}'], 'b.json')
      await api.previewImport(file)

      const args = (mockAxios.post as ReturnType<typeof vi.fn>).mock.calls[0]
      expect(args[0]).toBe('/data/import/preview')
      expect(args[1]).toBeInstanceOf(FormData)
      expect(args[2]?.headers?.['Content-Type']).toContain('multipart/form-data')
    })
  })

  describe('applyImport', () => {
    it('should POST /data/import/apply with JSON body', async () => {
      const mockResponse = { data: { applied: {} } }
      ;(mockAxios.post as ReturnType<typeof vi.fn>).mockResolvedValue(mockResponse)

      const payload = {
        data: {
          tasks: [],
          sessions: [],
          schedules: [],
          work_logs: [],
          work_reports: [],
          settings: { pomodoro: {} as any, ai: {} as any }
        },
        modules: {}
      }
      await api.applyImport(payload)

      expect(mockAxios.post).toHaveBeenCalledWith('/data/import/apply', payload)
    })
  })
})