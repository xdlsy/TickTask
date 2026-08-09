import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ElementPlus from 'element-plus'
import AgentTurn from './AgentTurn.vue'
import type { AgentMessage } from '@/types'
import type { Turn } from './useAgentTurns'

function user(id: string, content: string): AgentMessage {
  return { id, conversation_id: 'c', role: 'user', content, created_at: '' }
}
function ast(id: string, content: string): AgentMessage {
  return { id, conversation_id: 'c', role: 'assistant', content, created_at: '' }
}
function tool(id: string): AgentMessage {
  return { id, conversation_id: 'c', role: 'tool_call', content: '', tool_name: 'list_tasks', tool_args: '{}', tool_status: 'succeeded', tool_result: '[1]', created_at: '' }
}

describe('AgentTurn', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('renders the user bubble on the right', () => {
    const turn: Turn = { id: 'u1', user: user('u1', 'hi'), segments: [] }
    const w = mount(AgentTurn, { props: { turn }, global: { plugins: [ElementPlus] } })
    expect(w.find('.msg.user').exists()).toBe(true)
    expect(w.text()).toContain('hi')
  })

  it('renders text + tool segments inside the agent block', () => {
    const turn: Turn = { id: 'u1', user: user('u1', 'hi'), segments: [
      { kind: 'text', message: ast('a1', '**ok**') },
      { kind: 'tool', message: tool('t1') },
    ] }
    const w = mount(AgentTurn, { props: { turn }, global: { plugins: [ElementPlus] } })
    expect(w.findComponent({ name: 'MarkdownText' }).exists() || w.find('.md').exists()).toBe(true)
    expect(w.find('[data-testid="tool-row"]').exists()).toBe(true)
  })

  it('shows the pulsing indicator when live has no text', () => {
    const turn: Turn = { id: 'u1', user: user('u1', 'hi'), segments: [], live: { text: '' } }
    const w = mount(AgentTurn, { props: { turn }, global: { plugins: [ElementPlus] } })
    expect(w.find('.pulse').exists()).toBe(true)
  })

  it('shows streaming text + caret when live has text', () => {
    const turn: Turn = { id: 'u1', user: user('u1', 'hi'), segments: [], live: { text: 'think' } }
    const w = mount(AgentTurn, { props: { turn }, global: { plugins: [ElementPlus] } })
    expect(w.find('.live-stream').exists()).toBe(true)
    expect(w.find('.caret').exists()).toBe(true)
  })
})
