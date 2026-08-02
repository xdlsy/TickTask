export type Quadrant = 1 | 2 | 3 | 4

export type TaskStatus = 'todo' | 'in_progress' | 'completed' | 'cancelled'

export interface Task {
  id: string
  title: string
  description: string
  quadrant: Quadrant
  is_important: boolean
  is_urgent: boolean
  status: TaskStatus
  estimated_time: number
  deadline: string | null
  start_date: string | null
  due_date: string | null
  is_recurring: boolean
  recurrence_pattern: string
  recurrence_day: number
  preferred_start_time: string
  preferred_end_time: string
  tags: string[]
  order: number
  created_at: string
  updated_at: string
  completed_at: string | null
}

export type SessionType = 'work' | 'short_break' | 'long_break'

export type SessionStatus = 'pending' | 'running' | 'paused' | 'completed' | 'abandoned'

export interface PomodoroSession {
  id: string
  task_id: string | null
  type: SessionType
  status: SessionStatus
  start_time: string
  end_time: string | null
  planned_duration: number
  actual_duration: number | null
  interruptions: number
  interrupt_reason: string | null
  created_at: string
}

export interface PomodoroSettings {
  work_duration: number
  short_break_duration: number
  long_break_duration: number
  long_break_after: number
  auto_start_break: boolean
  auto_start_work: boolean
  enable_sound: boolean
  buffer_ratio: number
  task_time_preferences: string
  scheduling_strategy: string
}

export interface QuadrantInfo {
  id: Quadrant
  name: string
  description: string
  color: string
}

export const QUADRANT_INFO: Record<Quadrant, QuadrantInfo> = {
  1: { id: 1, name: '重要且紧急', description: '立即执行', color: '#ef4444' },
  2: { id: 2, name: '重要不紧急', description: '计划安排', color: '#f59e0b' },
  3: { id: 3, name: '紧急不重要', description: '委托他人', color: '#3b82f6' },
  4: { id: 4, name: '不重要不紧急', description: '删除/减少', color: '#6b7280' }
}

// WebSocket 消息类型
export type WSMessageType =
  | 'timer_tick'
  | 'timer_complete'
  | 'session_state'
  | 'task_updated'
  | 'terminal_output'
  | 'terminal_status'
  | 'error'

export interface TimerTickMessage {
  type: 'timer_tick'
  session_id: string
  remaining: number
  total: number
  percentage: number
}

export interface SessionStateMessage {
  type: 'session_state'
  id: string
  status: SessionStatus
}

export interface TimerCompleteMessage {
  type: 'timer_complete'
  session_id: string
}

export interface TerminalOutputMessage {
  type: 'terminal_output'
  chunk: string
  is_stderr?: boolean
}

export interface TerminalStatusMessage {
  type: 'terminal_status'
  status: string
  message?: string
  detail?: string
}

export type WSMessage = TimerTickMessage | SessionStateMessage | TimerCompleteMessage | TerminalOutputMessage | TerminalStatusMessage

// AI 相关类型
export interface ClassificationResult {
  task_id: string
  important: boolean
  urgent: boolean
  quadrant: number
  reason: string
}

export interface ScheduleItem {
  task_id: string
  title: string
  start_time: string
  end_time: string
  pomodoro_count: number
}

export interface DailySchedule {
  schedule: ScheduleItem[]
}

export interface PrioritySuggestion {
  priority_order: string[]
}

export interface AIStatus {
  configured: boolean
}

export interface AISettings {
  provider: string
  api_key: string
  base_url: string
  model: string
  cli_tool: string
}

// 任务时间统计
export interface TaskTimeStats {
  task_id: string
  task_title: string
  session_count: number
  total_time: number // 秒
}

// 数据分析相关类型
export interface DailySummary {
  completed_pomodoros: number
  total_focus_time: number
  completed_tasks: number
  created_tasks: number
}

export interface TrendDataPoint {
  date: string
  focus_time: number
  pomodoros: number
}

export interface TrendData {
  data: TrendDataPoint[]
}

export interface QuadrantStats {
  total: number
  completed: number
}

export interface DistributionStats {
  quadrant_stats: Record<number, QuadrantStats>
  task_stats: {
    total: number
    completed: number
    completion_rate: number
  }
}

// 日程相关类型
export type ScheduleType = 'task' | 'pomodoro' | 'break' | 'custom'
export type ScheduleStatus = 'planned' | 'in_progress' | 'completed' | 'cancelled'

export interface ScheduleEvent {
  id: string
  title: string
  start: string
  end: string
  type: ScheduleType
  status: ScheduleStatus
  color: string
  task_id?: string
  allDay: boolean
  editable: boolean
  ai_adjusted: boolean
  adjustment_type: string
}

// AI 重排程结果
export interface AdjustedItem {
  task_id: string
  title: string
  start_time: string
  end_time: string
  adjustment: string
  reason: string
}

export interface RescheduleResult {
  adjusted_schedule: AdjustedItem[]
  summary: string
}

// 修订日程
export interface RevisionChange {
  type: 'moved' | 'added' | 'removed'
  title: string
  original_start?: string
  original_end?: string
  new_start?: string
  new_end?: string
}

export interface ReviseResponse {
  applied: boolean
  summary: string
  changes: RevisionChange[]
  events: ScheduleEvent[]
}

// AI 每日洞察
export interface DailyInsights {
  productivity_score: number
  peak_hours: string
  achievements: string[]
  suggestions: string[]
  motivation: string
}

export interface CreateScheduleDTO {
  task_id?: string
  title: string
  description?: string
  start_time: string
  end_time: string
  type: ScheduleType
  color?: string
}

export interface UpdateScheduleDTO {
  title?: string
  description?: string
  start_time?: string
  end_time?: string
  status?: ScheduleStatus
  color?: string
}

export interface MoveScheduleDTO {
  start_time: string
  end_time: string
}

export type ViewMode = 'day' | 'week' | 'month'

// 番茄钟-任务关联类型
export type PomodoroStatus = 'not_started' | 'in_progress' | 'completed' | 'exceeded'

export interface TaskResponse extends Task {
  planned_pomodoros: number
  completed_pomodoros: number
  pomodoro_status: PomodoroStatus
}

export interface PomodoroByTaskItem {
  task_id: string
  task_title: string
  planned_pomodoros: number
  completed_pomodoros: number
  total_focus_minutes: number
  status: string
}

export interface PomodoroByTaskResult {
  tasks: PomodoroByTaskItem[]
}

export interface PomodoroTrendDay {
  date: string
  planned: number
  actual: number
  completed_tasks: number
  exceeded_tasks: number
}

export interface PomodoroTrendsResult {
  days: PomodoroTrendDay[]
}

// ── 工作日志 ──

export interface WorkItem {
  id: string
  work_log_id: string
  seq: number
  title: string
  content: string
  problem_solved: string
  result: string
  impact: string
}

export interface WorkLog {
  id: string
  date: string
  summary: string
  raw_brain_dump: string
  created_at: string
  updated_at: string
  items: WorkItem[]
}

export type WorkReportType = 'weekly' | 'monthly' | 'halfyear' | 'yearly'

export interface WorkReport {
  id: string
  type: WorkReportType
  period_key: string
  start_date: string
  end_date: string
  summary_json: string
  missing_days: string
  created_at: string
  updated_at: string
}

export interface TodayContext {
  date: string
  completed_tasks: Array<{ id: string; title: string }>
  pomodoro_sessions: Array<{
    id: string
    task_id: string
    task_title: string
    started_at: string
    minutes: number
  }>
  pomodoro_summary: { count: number; total_minutes: number }
}

export interface StructuredWorkLog {
  items: Array<{
    title: string
    content: string
    problem_solved: string
    result: string
    impact: string
  }>
  summary: string
}

export interface SaveWorkLogInput {
  date: string
  summary: string
  raw_brain_dump: string
  items: Array<{
    seq: number
    title: string
    content: string
    problem_solved: string
    result: string
    impact: string
  }>
}

export interface ReportSummary {
  core_work: string
  main_progress: string
  open_issues: string
  next_focus: string
}
