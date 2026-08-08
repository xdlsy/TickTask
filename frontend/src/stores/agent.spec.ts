import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAgentStore } from './agent'

vi.mock('@/api/client', () => ({
  api: {
    agent: {
      listConversations: vi.fn().mockResolvedValue({ items: [], total: 0 }),
      createConversation: vi.fn().mockResolvedValue({ id: 'c1', title: 'New' }),
      chat: vi.fn(),
      runTool: vi.fn().mockResolvedValue({ result: 'ok' }),
      confirm: vi.fn(),
      status: vi.fn().mockResolvedValue({ configured: true, supports_function_calling: true, provider: 'openai' }),
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
})
