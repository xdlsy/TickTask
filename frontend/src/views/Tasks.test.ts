import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import TasksView from './Tasks.vue'

const taskStore = vi.hoisted(() => ({
  fetchTasks: vi.fn().mockResolvedValue(undefined),
  fetchTasksByQuadrant: vi.fn().mockResolvedValue(undefined),
  createTask: vi.fn().mockResolvedValue(undefined)
}))

const mockPush = vi.hoisted(() => vi.fn())
const mockReplace = vi.hoisted(() => vi.fn())
const mockRoute = vi.hoisted(() => ({ query: {} }))

vi.mock('@/stores/task', () => ({
  useTaskStore: () => taskStore
}))

vi.mock('@element-plus/icons-vue', () => ({
  Plus: { template: '<span />' }
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mockPush, replace: mockReplace }),
  useRoute: () => mockRoute
}))

const QuadrantViewStub = {
  name: 'QuadrantView',
  template: '<div />',
  setup() {
    function onAddTask() {}
    return { onAddTask }
  }
}

const ListViewStub = {
  name: 'ListView',
  template: '<div />',
  setup() {
    function onAddTask() {}
    return { onAddTask }
  }
}

const elStubs = {
  'el-button': true,
  'el-icon': true,
  QuadrantView: QuadrantViewStub,
  ListView: ListViewStub,
  TaskForm: true
}

describe('Tasks View', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    // Reset mockRoute query before each test
    mockRoute.query = {}
    // Ensure mockResolvedValue persists after clearAllMocks
    taskStore.fetchTasks = vi.fn().mockResolvedValue(undefined)
    taskStore.fetchTasksByQuadrant = vi.fn().mockResolvedValue(undefined)
    taskStore.createTask = vi.fn().mockResolvedValue(undefined)
  })

  describe('rendering', () => {
    it('renders the page title', () => {
      const wrapper = mount(TasksView, { global: { stubs: elStubs } })
      expect(wrapper.find('h1').text()).toBe('任务管理')
    })

    it('renders QuadrantView by default', () => {
      const wrapper = mount(TasksView, { global: { stubs: elStubs } })
      expect(wrapper.findComponent({ name: 'QuadrantView' }).exists()).toBe(true)
      expect(wrapper.findComponent({ name: 'ListView' }).exists()).toBe(false)
    })

    it('switches to ListView when viewMode is list', async () => {
      const wrapper = mount(TasksView, { global: { stubs: elStubs } })
      wrapper.vm.viewMode = 'list'
      await wrapper.vm.$nextTick()
      expect(wrapper.findComponent({ name: 'ListView' }).exists()).toBe(true)
      expect(wrapper.findComponent({ name: 'QuadrantView' }).exists()).toBe(false)
    })
  })

  describe('onMounted', () => {
    it('fetches tasks on mount', async () => {
      mount(TasksView, { global: { stubs: elStubs } })
      // onMounted is async, wait for microtasks
      await vi.waitFor(() => {
        expect(taskStore.fetchTasks).toHaveBeenCalled()
        expect(taskStore.fetchTasksByQuadrant).toHaveBeenCalled()
      })
    })

    it('triggers add task and clears query param when add=true', async () => {
      mockRoute.query = { add: 'true' }
      mount(TasksView, { global: { stubs: elStubs } })
      await vi.waitFor(() => {
        expect(mockReplace).toHaveBeenCalledWith({ query: {} })
      })
    })

    it('does not open add form when query param is absent', async () => {
      const wrapper = mount(TasksView, { global: { stubs: elStubs } })
      await vi.waitFor(() => {
        expect(taskStore.fetchTasksByQuadrant).toHaveBeenCalled()
      })
      expect(wrapper.vm.showForm).toBe(false)
      expect(mockReplace).not.toHaveBeenCalled()
    })
  })

  describe('onAddTask', () => {
    it('falls back to showing form when ref not available', async () => {
      const wrapper = mount(TasksView, { global: { stubs: elStubs } })
      // Clear refs so it falls through to the else branch
      ;(wrapper.vm as any).quadrantViewRef = null
      ;(wrapper.vm as any).listViewRef = null
      wrapper.vm.onAddTask()
      expect(wrapper.vm.showForm).toBe(true)
      expect(wrapper.vm.editingTask).toBeNull()
    })
  })

  describe('onSaveTask', () => {
    it('creates task and closes form', async () => {
      const wrapper = mount(TasksView, { global: { stubs: elStubs } })
      wrapper.vm.showForm = true
      await wrapper.vm.onSaveTask({ title: 'New Task' })
      expect(taskStore.createTask).toHaveBeenCalledWith({ title: 'New Task' })
      expect(wrapper.vm.showForm).toBe(false)
    })
  })

  describe('view mode toggle', () => {
    it('toggles view mode on button click', async () => {
      const wrapper = mount(TasksView, { global: { stubs: elStubs } })
      const buttons = wrapper.findAll('.view-btn')
      await buttons[1].trigger('click')
      expect(wrapper.vm.viewMode).toBe('list')
      await buttons[0].trigger('click')
      expect(wrapper.vm.viewMode).toBe('quadrant')
    })

    it('applies active class to current view', async () => {
      const wrapper = mount(TasksView, { global: { stubs: elStubs } })
      expect(wrapper.findAll('.view-btn')[0].classes()).toContain('active')
      await wrapper.findAll('.view-btn')[1].trigger('click')
      expect(wrapper.findAll('.view-btn')[1].classes()).toContain('active')
    })
  })
})
