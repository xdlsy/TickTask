import { describe, it, expect, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import ElementPlus from 'element-plus'
import AgentDrawer from './AgentDrawer.vue'
import { useAgentStore } from '@/stores/agent'

// AgentDrawer calls useRouter(); install a real (in-memory) router like App.spec.ts.
function buildRouter() {
  const stub = { template: '<div />' }
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/dashboard', name: 'dashboard', component: stub },
      { path: '/settings', name: 'settings', component: stub },
    ],
  })
}

describe('AgentDrawer', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('does not render when closed', async () => {
    const router = buildRouter()
    await router.push('/dashboard')
    const w = mount(AgentDrawer, { global: { plugins: [router, ElementPlus] } })
    await flushPromises()
    expect(w.find('[data-testid="agent-drawer"]').exists()).toBe(false)
  })

  it('renders header/input/messages when open and configured', async () => {
    const router = buildRouter()
    await router.push('/dashboard')
    const w = mount(AgentDrawer, { global: { plugins: [router, ElementPlus] } })
    await flushPromises()
    const store = useAgentStore()
    store.openDrawer()
    store.status.configured = true
    await flushPromises()
    expect(w.find('[data-testid="agent-input"]').exists()).toBe(true)
    expect(w.find('.messages').exists()).toBe(true)
  })

  it('shows the not-configured hint instead of the input when unconfigured', async () => {
    const router = buildRouter()
    await router.push('/dashboard')
    const w = mount(AgentDrawer, { global: { plugins: [router, ElementPlus] } })
    await flushPromises()
    const store = useAgentStore()
    store.openDrawer()
    store.status.configured = false
    await flushPromises()
    expect(w.find('[data-testid="agent-input"]').exists()).toBe(false)
    expect(w.find('.not-configured-hint').exists()).toBe(true)
  })
})
