import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import AgentDrawer from './AgentDrawer.vue'
import ElementPlus from 'element-plus'

describe('AgentDrawer', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('does not render when closed', () => {
    const w = mount(AgentDrawer, { global: { plugins: [ElementPlus] } })
    expect(w.find('[data-testid="agent-drawer"]').exists()).toBe(false)
  })

  it('renders header/input/messages when open', async () => {
    const w = mount(AgentDrawer, { global: { plugins: [ElementPlus] } })
    await w.vm.$nextTick()
    // Set isOpen via store
    const { useAgentStore } = await import('@/stores/agent')
    useAgentStore().openDrawer()
    await w.vm.$nextTick()
    expect(w.find('[data-testid="agent-input"]').exists()).toBe(true)
  })
})
