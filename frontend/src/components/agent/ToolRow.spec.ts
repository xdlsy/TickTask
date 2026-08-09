import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ElementPlus from 'element-plus'
import ToolRow from './ToolRow.vue'
import { useAgentStore } from '@/stores/agent'

const baseMsg: any = {
  id: 'm1', conversation_id: 'c1', role: 'tool_call', content: '',
  tool_name: 'list_tasks', tool_args: '{}', tool_status: 'succeeded', created_at: '',
}

describe('ToolRow', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('read succeeded gets read class', () => {
    const w = mount(ToolRow, { props: { message: { ...baseMsg, tool_status: 'succeeded' } }, global: { plugins: [ElementPlus] } })
    expect(w.classes()).toContain('read')
  })
  it('write pending shows write class + inline confirm buttons', () => {
    const w = mount(ToolRow, { props: { message: { ...baseMsg, tool_name: 'create_task', tool_args: '{"title":"x"}', tool_status: 'pending_confirmation' } }, global: { plugins: [ElementPlus] } })
    expect(w.classes()).toContain('write')
    expect(w.find('[data-testid="tool-confirm-approve"]').exists()).toBe(true)
  })
  it('danger pending shows danger class', () => {
    const w = mount(ToolRow, { props: { message: { ...baseMsg, tool_name: 'delete_task', tool_status: 'pending_confirmation' } }, global: { plugins: [ElementPlus] } })
    expect(w.classes()).toContain('danger')
  })
  it('failed surfaces the error message', () => {
    const w = mount(ToolRow, { props: { message: { ...baseMsg, tool_status: 'failed', tool_result: '{"error":"oops"}' } }, global: { plugins: [ElementPlus] } })
    expect(w.text()).toContain('oops')
  })
  it('shows smart summary (count hint) for a succeeded read', () => {
    const w = mount(ToolRow, { props: { message: { ...baseMsg, tool_name: 'list_tasks', tool_result: '[1,2,3]' } }, global: { plugins: [ElementPlus] } })
    expect(w.text()).toContain('3 条任务')
  })
  it('toggles JSON detail on chevron click', async () => {
    const w = mount(ToolRow, { props: { message: { ...baseMsg, tool_args: '{"a":1}' } }, global: { plugins: [ElementPlus] } })
    expect(w.find('[data-testid="tool-json"]').exists()).toBe(false)
    await w.find('[data-testid="tool-toggle"]').trigger('click')
    expect(w.find('[data-testid="tool-json"]').exists()).toBe(true)
  })
  it('approve click calls store.confirmToolCall', async () => {
    const store = useAgentStore()
    const spy = vi.spyOn(store, 'confirmToolCall').mockResolvedValue(undefined as never)
    const w = mount(ToolRow, { props: { message: { ...baseMsg, id: 'tc1', tool_name: 'create_task', tool_args: '{"title":"x"}', tool_status: 'pending_confirmation' } }, global: { plugins: [ElementPlus] } })
    await w.find('[data-testid="tool-confirm-approve"]').trigger('click')
    expect(spy).toHaveBeenCalledWith('tc1', 'approve')
  })
})
