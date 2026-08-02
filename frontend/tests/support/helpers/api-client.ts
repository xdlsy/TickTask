import type { APIRequestContext } from '@playwright/test'
import type {
  Task,
  PomodoroSession,
  ScheduleEvent,
  CreateScheduleDTO,
  UpdateScheduleDTO,
  MoveScheduleDTO,
  DailySummary,
  TrendData,
  DistributionStats,
  AIStatus,
  AISettings,
  PomodoroSettings,
  ClassificationResult,
  TaskTimeStats,
  ReviseResponse,
  PomodoroByTaskResult,
  PomodoroTrendsResult,
} from '../../../src/types'

/**
 * Typed API client for E2E tests.
 * Uses Playwright's request context for direct backend calls.
 */
export class ApiClient {
  constructor(private request: APIRequestContext) {}

  // --- Tasks ---

  async getTasks(): Promise<Task[]> {
    const res = await this.request.get('/api/tasks')
    return res.json()
  }

  async getTasksByQuadrant(): Promise<Record<string, Task[]>> {
    const res = await this.request.get('/api/tasks/quadrant')
    return res.json()
  }

  async getTask(id: string): Promise<Task> {
    const res = await this.request.get(`/api/tasks/${id}`)
    return res.json()
  }

  async createTask(data: Partial<Task> & { title: string }): Promise<Task> {
    const res = await this.request.post('/api/tasks', { data })
    return res.json()
  }

  async updateTask(id: string, data: Partial<Task>): Promise<{ message: string }> {
    const res = await this.request.put(`/api/tasks/${id}`, { data })
    return res.json()
  }

  async deleteTask(id: string): Promise<{ message: string }> {
    const res = await this.request.delete(`/api/tasks/${id}`)
    return res.json()
  }

  async moveTask(id: string, quadrant: number): Promise<{ message: string }> {
    const res = await this.request.patch(`/api/tasks/${id}/move`, { data: { quadrant } })
    return res.json()
  }

  // --- Timer / Sessions ---

  async getActiveSession(): Promise<PomodoroSession | null> {
    const res = await this.request.get('/api/sessions/active')
    const json = await res.json()
    return (json as { session: PomodoroSession | null }).session ?? json
  }

  async getRecentSessions(limit?: number): Promise<PomodoroSession[]> {
    const res = await this.request.get('/api/sessions/recent', {
      params: { limit },
    })
    return res.json()
  }

  async getTodayTaskStats(): Promise<TaskTimeStats[]> {
    const res = await this.request.get('/api/sessions/today-stats')
    return res.json()
  }

  async createSession(data: { task_id?: string; type: string; duration?: number }): Promise<PomodoroSession> {
    const res = await this.request.post('/api/sessions', { data })
    return res.json()
  }

  async controlSession(id: string, action: string, interruptReason?: string): Promise<{ message: string }> {
    const res = await this.request.patch(`/api/sessions/${id}/control`, {
      data: { action, interrupt_reason: interruptReason },
    })
    return res.json()
  }

  // --- Schedule ---

  async getSchedules(start?: string, end?: string): Promise<ScheduleEvent[]> {
    const res = await this.request.get('/api/schedules', {
      params: { start, end },
    })
    const json = await res.json()
    return (json as { events: ScheduleEvent[] }).events ?? json
  }

  async createSchedule(data: CreateScheduleDTO): Promise<ScheduleEvent> {
    const res = await this.request.post('/api/schedules', { data })
    return res.json()
  }

  async updateSchedule(id: string, data: UpdateScheduleDTO): Promise<{ message: string }> {
    const res = await this.request.put(`/api/schedules/${id}`, { data })
    return res.json()
  }

  async deleteSchedule(id: string): Promise<{ message: string }> {
    const res = await this.request.delete(`/api/schedules/${id}`)
    return res.json()
  }

  async moveSchedule(id: string, data: MoveScheduleDTO): Promise<{ message: string }> {
    const res = await this.request.put(`/api/schedules/${id}/move`, { data })
    return res.json()
  }

  async deleteAllSchedules(): Promise<{ deleted: number }> {
    const res = await this.request.delete('/api/schedules')
    return res.json()
  }

  async generateSchedule(startTime?: string, endTime?: string): Promise<ScheduleEvent[]> {
    const res = await this.request.post('/api/schedules/generate', {
      data: { start_time: startTime, end_time: endTime },
      timeout: 360000,
    })
    const json = await res.json()
    return (json as { events: ScheduleEvent[] }).events ?? json
  }

  async reviseSchedule(prompt: string): Promise<ReviseResponse> {
    const res = await this.request.post('/api/schedules/revise', {
      data: { prompt },
      timeout: 360000,
    })
    return res.json()
  }

  async applyRevision(): Promise<{ applied: boolean; events: ScheduleEvent[] }> {
    const res = await this.request.post('/api/schedules/revise/apply')
    return res.json()
  }

  // --- Analytics ---

  async getAnalyticsSummary(date?: string): Promise<DailySummary> {
    const res = await this.request.get('/api/analytics/summary', {
      params: { date },
    })
    return res.json()
  }

  async getAnalyticsTrend(days?: number): Promise<TrendData> {
    const res = await this.request.get('/api/analytics/trend', {
      params: { days },
    })
    return res.json()
  }

  async getAnalyticsDistribution(start?: string, end?: string): Promise<DistributionStats> {
    const res = await this.request.get('/api/analytics/distribution', {
      params: { start, end },
    })
    return res.json()
  }

  async getPomodoroByTask(period?: string): Promise<PomodoroByTaskResult> {
    const res = await this.request.get('/api/analytics/pomodoro-by-task', {
      params: { period },
    })
    return res.json()
  }

  async getPomodoroTrends(period?: string): Promise<PomodoroTrendsResult> {
    const res = await this.request.get('/api/analytics/pomodoro-trends', {
      params: { period },
    })
    return res.json()
  }

  // --- AI ---

  async getAIStatus(): Promise<AIStatus> {
    const res = await this.request.get('/api/ai/status')
    return res.json()
  }

  async classifyTask(taskId: string): Promise<ClassificationResult> {
    const res = await this.request.post('/api/ai/classify', { data: { task_id: taskId } })
    return res.json()
  }

  async batchClassifyTasks(taskIds: string[]): Promise<Record<string, ClassificationResult>> {
    const res = await this.request.post('/api/ai/classify/batch', { data: { task_ids: taskIds } })
    const json = await res.json()
    return (json as { results: Record<string, ClassificationResult> }).results ?? json
  }

  async classifyTaskText(title: string, description?: string): Promise<ClassificationResult> {
    const res = await this.request.post('/api/ai/classify-task-text', { data: { title, description } })
    return res.json()
  }

  // --- Settings ---

  async getSettings(): Promise<{ pomodoro: PomodoroSettings; ai: AISettings }> {
    const res = await this.request.get('/api/settings')
    return res.json()
  }

  async getPomodoroSettings(): Promise<PomodoroSettings> {
    const { pomodoro } = await this.getSettings()
    return pomodoro
  }

  async updatePomodoroSettings(settings: Partial<PomodoroSettings>): Promise<{ message: string }> {
    const res = await this.request.put('/api/settings/pomodoro', { data: settings })
    return res.json()
  }

  async updateAISettings(settings: Partial<AISettings>): Promise<{ message: string }> {
    const res = await this.request.put('/api/settings/ai', { data: settings })
    return res.json()
  }
}
