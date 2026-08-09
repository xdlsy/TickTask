import { describe, it, expect } from 'vitest'
import { groupIntoTurns } from './useAgentTurns'
import type { AgentMessage } from '@/types'

function user(id: string, content: string): AgentMessage {
  return { id, conversation_id: 'c', role: 'user', content, created_at: '' }
}
function ast(id: string, content: string): AgentMessage {
  return { id, conversation_id: 'c', role: 'assistant', content, created_at: '' }
}
function tool(id: string, name: string): AgentMessage {
  return { id, conversation_id: 'c', role: 'tool_call', content: '', tool_name: name, tool_args: '{}', tool_status: 'succeeded', created_at: '' }
}

describe('groupIntoTurns', () => {
  it('groups a user message with its following assistant + tool messages', () => {
    const turns = groupIntoTurns([user('u1', 'hi'), ast('a1', 'ok'), tool('t1', 'list_tasks')], '', false)
    expect(turns).toHaveLength(1)
    expect(turns[0].user?.id).toBe('u1')
    expect(turns[0].segments.map((s) => s.kind)).toEqual(['text', 'tool'])
  })

  it('starts a new turn at the next user message', () => {
    const turns = groupIntoTurns([user('u1', 'hi'), ast('a1', 'ok'), user('u2', 'again')], '', false)
    expect(turns).toHaveLength(2)
    expect(turns[1].user?.id).toBe('u2')
  })

  it('skips empty assistant segments (no empty bubble)', () => {
    const turns = groupIntoTurns([user('u1', 'hi'), ast('a1', '')], '', false)
    expect(turns[0].segments).toHaveLength(0)
  })

  it('attaches live streaming text to the last turn', () => {
    const turns = groupIntoTurns([user('u1', 'hi')], 'think', true)
    expect(turns[0].live?.text).toBe('think')
  })

  it('shows a live (empty) indicator when thinking but no text yet', () => {
    const turns = groupIntoTurns([user('u1', 'hi')], '', true)
    expect(turns[0].live?.text).toBe('')
  })

  it('no live when neither streaming nor thinking', () => {
    const turns = groupIntoTurns([user('u1', 'hi'), ast('a1', 'done')], '', false)
    expect(turns[0].live).toBeUndefined()
  })
})