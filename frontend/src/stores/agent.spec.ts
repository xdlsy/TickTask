import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAgentStore } from './agent'

vi.mock('@/api/client', () => ({
  api: {
    agent: {
      listConversations: vi.fn().mockResolvedValue({ data: { items: [], total: 0 } }),
      createConversation: vi.fn().mockResolvedValue({ data: { id: 'c1', title: 'New' } }),
      chat: vi.fn(),
      runTool: vi.fn().mockResolvedValue({ data: { result: 'ok' } }),
      confirm: vi.fn(),
      status: vi.fn().mockResolvedValue({ data: { configured: true, supports_function_calling: true, provider: 'openai' } }),
    },
  },
}))

describe('useAgentStore', () => {
  beforeEach(() => setActivePinia(createPinia()))
  it('opens/closes drawer', () => {
    const s = useAgentStore()
    expect(s.isOpen).toBe(false)
    s.openDrawer()
    expect(s.isOpen).toBe(true)
  })
  it('appends streaming tokens', () => {
    const s = useAgentStore()
    s.onAgentMessage({ type: 'agent_message', conversation_id: 'c1', message_id: 'm1', delta_text: 'a' })
    s.onAgentMessage({ type: 'agent_message', conversation_id: 'c1', message_id: 'm1', delta_text: 'b' })
    expect(s.streamingText).toBe('ab')
  })
  it('runTool calls API', async () => {
    const s = useAgentStore()
    const r = await s.runTool('classify_task', { task_id: '12' })
    expect(r).toBe('ok')
  })

  it('flushes an assistant segment when message_id changes', () => {
    const s = useAgentStore()
    s.onAgentMessage({ type: 'agent_message', conversation_id: 'c1', message_id: 'm1', delta_text: 'hello' })
    s.onAgentMessage({ type: 'agent_message', conversation_id: 'c1', message_id: 'm2', delta_text: 'world' })
    expect(s.messages).toHaveLength(1)
    expect(s.messages[0]).toMatchObject({ id: 'm1', role: 'assistant', content: 'hello' })
    expect(s.streamingText).toBe('world')
  })

  it('materializes a read tool started -> succeeded (no message_id, matched by order)', () => {
    const s = useAgentStore()
    s.onAgentMessage({ type: 'agent_message', conversation_id: 'c1', message_id: 'm1', delta_text: 'reading' })
    s.onAgentTool({ type: 'agent_tool', conversation_id: 'c1', tool_name: 'list_tasks', args: {}, status: 'started' })
    s.onAgentTool({ type: 'agent_tool', conversation_id: 'c1', tool_name: 'list_tasks', args: {}, status: 'succeeded', result: [{ id: 1 }, { id: 2 }] })
    const tools = s.messages.filter((m) => m.tool_name === 'list_tasks')
    expect(tools).toHaveLength(1)
    expect(tools[0].tool_status).toBe('succeeded')
    expect(tools[0].tool_result).toContain('2') // serialized array
  })

  it('upserts a write tool by message_id and clears pendingConfirm on resolution', () => {
    const s = useAgentStore()
    s.onAgentMessage({ type: 'agent_message', conversation_id: 'c1', message_id: 'm1', delta_text: 'go' })
    s.onAgentTool({ type: 'agent_tool', conversation_id: 'c1', message_id: 'tc1', tool_name: 'create_task', args: { title: 'x' }, status: 'pending_confirmation', preview: { title: 'x' } })
    expect(s.pendingConfirm?.messageId).toBe('tc1')
    s.onAgentTool({ type: 'agent_tool', conversation_id: 'c1', message_id: 'tc1', tool_name: 'create_task', args: { title: 'x' }, status: 'succeeded', result: { id: 9 } })
    const tc = s.messages.find((m) => m.id === 'tc1')
    expect(tc?.tool_status).toBe('succeeded')
    expect(s.pendingConfirm).toBeNull()
  })

  it('flushes remaining streaming text on agent_done', () => {
    const s = useAgentStore()
    s.onAgentMessage({ type: 'agent_message', conversation_id: 'c1', message_id: 'm1', delta_text: 'final' })
    s.onAgentDone({ type: 'agent_done', conversation_id: 'c1', finish_reason: 'stop' })
    expect(s.streamingText).toBe('')
    expect(s.messages.find((m) => m.id === 'm1')).toMatchObject({ role: 'assistant', content: 'final' })
    expect(s.isThinking).toBe(false)
  })
})
