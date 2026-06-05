import axios from 'axios'
import type { Task, PomodoroSession, ClassificationResult, PrioritySuggestion, AIStatus, PomodoroSettings, AISettings, TaskTimeStats, DailySummary, TrendData, DistributionStats, ScheduleEvent, CreateScheduleDTO, UpdateScheduleDTO, MoveScheduleDTO, RescheduleResult, DailyInsights, ReviseResponse } from '@/types'

const client = axios.create({
  baseURL: '/api',
  timeout: 60000, // AI 接口需要更长超时
  headers: {
    'Content-Type': 'application/json'
  }
})

// 请求拦截器
client.interceptors.request.use(
  config => {
    return config
  },
  error => {
    return Promise.reject(error)
  }
)

// 响应拦截器
client.interceptors.response.use(
  response => response,
  error => {
    console.error('API error:', error)
    return Promise.reject(error)
  }
)

export const api = {
  // 任务相关
  getTasks: () => client.get<Task[]>('/tasks'),
  getTasksByQuadrant: () => client.get<Record<string, Task[]>>('/tasks/quadrant'),
  getTask: (id: string) => client.get<Task>(`/tasks/${id}`),
  createTask: (data: any) => client.post<Task>('/tasks', data),
  updateTask: (id: string, data: any) => client.put(`/tasks/${id}`, data),
  deleteTask: (id: string) => client.delete(`/tasks/${id}`),
  moveTask: (id: string, quadrant: number) => client.patch(`/tasks/${id}/move`, { quadrant }),

  // 计时器相关
  getActiveSession: () => client.get<PomodoroSession | { session: PomodoroSession | null }>('/sessions/active'),
  getRecentSessions: (limit?: number) => client.get<PomodoroSession[]>('/sessions/recent', {
    params: { limit }
  }),
  getTodayTaskStats: () => client.get<TaskTimeStats[]>('/sessions/today-stats'),
  createSession: (data: any) => client.post<PomodoroSession>('/sessions', data),
  controlSession: (id: string, action: string, interrupt_reason?: string) => client.patch(`/sessions/${id}/control`, { action, interrupt_reason }),

  // AI 智能功能
  getAIStatus: () => client.get<AIStatus>('/ai/status'),
  classifyTask: (taskId: string) => client.post<ClassificationResult>('/ai/classify', { task_id: taskId }),
  classifyTasks: (taskIds: string[]) => client.post<{ results: ClassificationResult[] }>('/ai/classify/batch', { task_ids: taskIds }),
  classifyTaskByText: (title: string, description?: string) => client.post<ClassificationResult>('/ai/classify-task-text', { title, description }),
  generateSchedule: (startTime: string, endTime: string) => client.post<{ events: ScheduleEvent[] }>('/ai/schedule', { start_time: startTime, end_time: endTime }),
  rescheduleAfterInterrupt: (data: { task_id: string; completed_minutes?: number; planned_minutes: number; interrupt_reason?: string; work_end_time: string }) => client.post<RescheduleResult>('/ai/reschedule-after-interrupt', data),
  getPrioritySuggestions: () => client.get<PrioritySuggestion>('/ai/priority'),
  getDailyInsights: (params?: { date?: string; completed_pomodoros?: number; total_focus_minutes?: number; completed_tasks?: number; total_interruptions?: number; task_distribution?: string }) => client.get<DailyInsights>('/ai/daily-insights', { params }),

  // 设置相关
  getSettings: () => client.get<{ pomodoro: PomodoroSettings; ai: AISettings }>('/settings'),
  updatePomodoroSettings: (settings: PomodoroSettings) => client.put('/settings/pomodoro', settings),
  updateAISettings: (settings: AISettings) => client.put('/settings/ai', settings),

  // 数据分析
  getAnalyticsSummary: (date?: string) => client.get<DailySummary>('/analytics/summary', { params: { date } }),
  getAnalyticsTrend: (days?: number) => client.get<TrendData>('/analytics/trend', { params: { days } }),
  getAnalyticsDistribution: (start?: string, end?: string) => client.get<DistributionStats>('/analytics/distribution', { params: { start, end } }),

  // 日程相关
  getSchedules: (start?: string, end?: string) => client.get<{ events: ScheduleEvent[] }>('/schedules', { params: { start, end } }),
  getSchedule: (id: string) => client.get<ScheduleEvent>(`/schedules/${id}`),
  createSchedule: (data: CreateScheduleDTO) => client.post<ScheduleEvent>('/schedules', data),
  updateSchedule: (id: string, data: UpdateScheduleDTO) => client.put(`/schedules/${id}`, data),
  deleteSchedule: (id: string) => client.delete(`/schedules/${id}`),
  moveSchedule: (id: string, data: MoveScheduleDTO) => client.put(`/schedules/${id}/move`, data),
  generateScheduleFromTasks: (startTime?: string, endTime?: string) => client.post<{ events: ScheduleEvent[] }>('/schedules/generate', { start_time: startTime, end_time: endTime }, { timeout: 360000 }), // AI 整周生成可能需要 3-5 分钟
  deleteAllSchedules: () => client.delete<{ deleted: number }>('/schedules'),
  reviseSchedule: (prompt: string) => client.post<ReviseResponse>('/schedules/revise', { prompt }, { timeout: 360000 }),
  applyRevision: () => client.post<{ applied: boolean; events: ScheduleEvent[] }>('/schedules/revise/apply')
}
