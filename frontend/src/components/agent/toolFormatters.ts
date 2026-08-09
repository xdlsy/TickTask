import type { AgentMessage, ToolStatus } from '@/types'

export type Permission = 'read' | 'write' | 'danger'

// Mirrors backend/internal/agent/tools/*.go Permission values.
const DANGER_TOOLS = new Set(['delete_task', 'delete_schedule', 'apply_schedule_revision'])
const WRITE_TOOLS = new Set([
  'create_task', 'update_task', 'move_task',
  'generate_schedule', 'update_schedule', 'create_schedule', 'revise_schedule',
  'start_pomodoro', 'stop_pomodoro', 'control_pomodoro',
  'save_worklog', 'generate_work_report', 'update_worklog', 'update_worklog_summary', 'add_worklog_entry',
])

export function classifyPermission(toolName?: string): Permission {
  if (!toolName) return 'read'
  if (DANGER_TOOLS.has(toolName)) return 'danger'
  if (WRITE_TOOLS.has(toolName)) return 'write'
  return 'read'
}

// Per-tool display overrides: which arg to surface, and the noun for a count.
const LABELS: Record<string, { argKey?: string; noun?: string; doneHint?: string }> = {
  list_tasks: { noun: '条任务' },
  list_schedule: { noun: '个时段' },
  list_worklogs: { noun: '篇日志' },
  create_schedule: { noun: '个时段' },
  generate_schedule: { noun: '个时段' },
  create_task: { argKey: 'title' },
  update_task: { argKey: 'title' },
  delete_task: { argKey: 'task_id' },
  start_pomodoro: { doneHint: '完成' },
}

const ARG_PRIORITY = ['title', 'name', 'date', 'task_id', 'id', 'date_str']

function parseArgs(m: AgentMessage): Record<string, unknown> {
  try { return JSON.parse(m.tool_args || '{}') } catch { return {} }
}
function parseResult(m: AgentMessage): unknown {
  try { return JSON.parse(m.tool_result || '') } catch { return m.tool_result || undefined }
}

function firstScalarArg(args: Record<string, unknown>): { key: string; value: string } | undefined {
  for (const k of ARG_PRIORITY) {
    if (args[k] != null) return { key: k, value: String(args[k]) }
  }
  for (const [k, v] of Object.entries(args)) {
    if (typeof v === 'string' || typeof v === 'number') return { key: k, value: String(v) }
  }
  return undefined
}

export interface ToolSummary {
  argHint?: string
  resultHint?: string
}

export function summarizeTool(m: AgentMessage): ToolSummary {
  const args = parseArgs(m)
  const result = parseResult(m)
  const cfg = LABELS[m.tool_name || '']
  const status: ToolStatus | undefined = m.tool_status

  // argHint
  let argHint: string | undefined
  if (cfg?.argKey && args[cfg.argKey] != null) {
    argHint = `${cfg.argKey}=${args[cfg.argKey]}`
  } else {
    const s = firstScalarArg(args)
    if (s) argHint = `${s.key}=${s.value}`
  }

  // resultHint (only on success — failures are surfaced by ToolRow via tool_result)
  let resultHint: string | undefined
  if (status === 'succeeded') {
    if (cfg?.doneHint) {
      resultHint = cfg.doneHint
    } else if (Array.isArray(result)) {
      resultHint = `${result.length} ${cfg?.noun || '项'}`
    } else if (result && typeof result === 'object' && 'count' in (result as Record<string, unknown>)) {
      resultHint = `${(result as Record<string, unknown>).count} ${cfg?.noun || '项'}`
    }
  }

  return { argHint, resultHint }
}
