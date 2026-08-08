// ToolCard.spec.ts
import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ElementPlus from 'element-plus'
import ToolCard from './ToolCard.vue'

const baseMsg: any = {
  id: 'm1', conversation_id: 'c1', role: 'tool_call', content: '',
  tool_name: 'list_tasks', tool_args: '{}', tool_status: 'succeeded', created_at: '',
}

describe('ToolCard', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('read succeeded shows green', () => {
    const w = mount(ToolCard, {
      props: { message: { ...baseMsg, tool_status: 'succeeded' } },
      global: { plugins: [ElementPlus] },
    })
    expect(w.classes()).toContain('read')
  })
  it('write pending shows yellow with confirm buttons', () => {
    const w = mount(ToolCard, {
      props: { message: { ...baseMsg, tool_name: 'create_task', tool_status: 'pending_confirmation' } },
      global: { plugins: [ElementPlus] },
    })
    expect(w.classes()).toContain('write')
    expect(w.find('[data-testid="confirm-btn"]').exists()).toBe(true)
  })
  it('dangerous pending shows red', () => {
    const w = mount(ToolCard, {
      props: { message: { ...baseMsg, tool_name: 'delete_task', tool_status: 'pending_confirmation' } },
      global: { plugins: [ElementPlus] },
    })
    expect(w.classes()).toContain('danger')
  })
  it('failed shows error text', () => {
    const w = mount(ToolCard, {
      props: { message: { ...baseMsg, tool_status: 'failed', tool_result: '{"error":"oops"}' } },
      global: { plugins: [ElementPlus] },
    })
    expect(w.text()).toContain('oops')
  })
})
