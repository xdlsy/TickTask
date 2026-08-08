import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import ElementPlus from 'element-plus'
import App from './App.vue'

// Stub the websocket singleton so it doesn't try to connect under jsdom.
vi.mock('@/utils/websocket', () => ({
  wsClient: {
    connect: vi.fn(),
    on: vi.fn(),
    off: vi.fn(),
    send: vi.fn(),
  },
}))

// Mock API client so onMounted status checks don't fire real HTTP.
vi.mock('@/api/client', () => ({
  api: {
    getAIStatus: vi.fn().mockResolvedValue({ data: { configured: false } }),
    agent: {
      status: vi.fn().mockResolvedValue({
        data: { configured: false, supports_function_calling: false, provider: '' },
      }),
    },
  },
}))

// Stub AgentDrawer so we don't pull in the entire agent tree; verify the mount
// point exists by data-testid instead.
vi.mock('@/components/agent/AgentDrawer.vue', () => ({
  default: {
    name: 'AgentDrawer',
    template: '<div data-testid="agent-drawer-stub"></div>',
  },
}))

function buildRouter() {
  const stub = { template: '<div />' }
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', redirect: '/dashboard' },
      { path: '/dashboard', name: 'dashboard', component: stub },
      { path: '/timer', name: 'timer', component: stub },
      { path: '/tasks', name: 'tasks', component: stub },
      { path: '/schedule', name: 'schedule', component: stub },
      { path: '/analytics', name: 'analytics', component: stub },
      { path: '/work-log', name: 'work-log', component: stub },
      { path: '/settings', name: 'settings', component: stub },
    ],
  })
}

describe('App.vue — agent integration', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('renders the agent trigger button in the topbar', async () => {
    const router = buildRouter()
    await router.push('/dashboard')
    const w = mount(App, { global: { plugins: [router, ElementPlus] } })
    await flushPromises()
    expect(w.find('.agent-trigger').exists()).toBe(true)
    expect(w.find('[data-testid="agent-drawer-stub"]').exists()).toBe(true)
  })

  it('opens the agent drawer when the trigger is clicked', async () => {
    const router = buildRouter()
    await router.push('/dashboard')
    const w = mount(App, { global: { plugins: [router, ElementPlus] } })
    await flushPromises()
    const { useAgentStore } = await import('@/stores/agent')
    const store = useAgentStore()
    expect(store.isOpen).toBe(false)
    await w.find('.agent-trigger').trigger('click')
    expect(store.isOpen).toBe(true)
  })
})
