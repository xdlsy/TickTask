import { describe, it, expect } from 'vitest'
import { summarizeTool, classifyPermission } from './toolFormatters'
import type { AgentMessage } from '@/types'

function msg(over: Partial<AgentMessage>): AgentMessage {
  return {
    id: 'm', conversation_id: 'c', role: 'tool_call', content: '',
    tool_name: 'list_tasks', tool_args: '{}', tool_status: 'succeeded', created_at: '',
    ...over,
  }
}

describe('classifyPermission', () => {
  it('delete_task is danger', () => {
    expect(classifyPermission('delete_task')).toBe('danger')
  })
  it('create_task is write', () => {
    expect(classifyPermission('create_task')).toBe('write')
  })
  it('list_tasks is read', () => {
    expect(classifyPermission('list_tasks')).toBe('read')
  })
  it('unknown tool defaults to read', () => {
    expect(classifyPermission('something_new')).toBe('read')
  })
})

describe('summarizeTool', () => {
  it('array result → count hint with noun', () => {
    const s = summarizeTool(msg({ tool_name: 'list_tasks', tool_result: '[{"id":1},{"id":2}]' }))
    expect(s.resultHint).toBe('2 条任务')
  })
  it('create_schedule array result uses 时段 noun', () => {
    const s = summarizeTool(msg({ tool_name: 'create_schedule', tool_result: '[1,2,3]' }))
    expect(s.resultHint).toBe('3 个时段')
  })
  it('create_task surfaces title as argHint', () => {
    const s = summarizeTool(msg({ tool_name: 'create_task', tool_args: '{"title":"写文档"}', tool_status: 'pending_confirmation' }))
    expect(s.argHint).toBe('title=写文档')
  })
  it('unknown tool falls back to first scalar arg + generic count', () => {
    const s = summarizeTool(msg({ tool_name: 'mystery_tool', tool_args: '{"q":"x"}', tool_result: '[1,2]' }))
    expect(s.argHint).toBe('q=x')
    expect(s.resultHint).toBe('2 项')
  })
  it('failed status yields no resultHint', () => {
    const s = summarizeTool(msg({ tool_name: 'list_tasks', tool_status: 'failed', tool_result: '{"error":"oops"}' }))
    expect(s.resultHint).toBeUndefined()
  })
  it('start_pomodoro succeeded → 完成 hint', () => {
    const s = summarizeTool(msg({ tool_name: 'start_pomodoro', tool_result: '{"started":true}' }))
    expect(s.resultHint).toBe('完成')
  })
  it('started tool with no result yields no resultHint', () => {
    const s = summarizeTool(msg({ tool_name: 'list_tasks', tool_status: 'started', tool_result: undefined }))
    expect(s.resultHint).toBeUndefined()
  })
  it('handles a missing tool_name defensively', () => {
    const s = summarizeTool(msg({ tool_name: undefined, tool_args: '{"title":"x"}' }))
    // no LABELS match → argHint falls back to first scalar arg by priority
    expect(s.argHint).toBe('title=x')
    expect(s.resultHint).toBeUndefined()
  })
  it('survives malformed tool_args JSON', () => {
    const s = summarizeTool(msg({ tool_name: 'list_tasks', tool_args: '{not json' }))
    expect(s.argHint).toBeUndefined()
  })
})
