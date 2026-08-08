import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { nextTick } from 'vue'
import { useRouter } from 'vue-router'
import Dashboard from './Dashboard.vue'
import type { Task } from '@/types'

const mockTasks: Task[] = [
  {
    id: 'task-1',
    title: '完成代码审查',
    description: '',
    quadrant: 1,
    is_important: true,
    is_urgent: true,
    status: 'todo',
    estimated_time: 1500,
    deadline: null,
    tags: [],
    order: 0,
    created_at: '2026-05-25T08:00:00Z',
    updated_at: '2026-05-25T08:00:00Z',
    completed_at: null
  },
  {
    id: 'task-2',
    title: '重构用户模块',
    description: '',
    quadrant: 2,
    is_important: true,
    is_urgent: false,
    status: 'completed',
    estimated_time: 3600,
    deadline: null,
    tags: [],
    order: 1,
    created_at: '2026-05-25T07:00:00Z',
    updated_at: '2026-05-25T10:00:00Z',
    completed_at: '2026-05-25T10:00:00Z'
  }
]

const mockSessions = [
  {
    id: 's1', task_id: 'task-2', type: 'work' as const, status: 'completed' as const,
    start_time: '2026-05-25T09:00:00Z', end_time: '2026-05-25T09:25:00Z',
    planned_duration: 1500, actual_duration: 1500, interruptions: 0,
    interrupt_reason: null, created_at: '2026-05-25T09:00:00Z'
  }
]

const mockTaskStore = {
  tasks: [] as Task[],
  loading: false,
  fetchTasks: vi.fn(),
  fetchTasksByQuadrant: vi.fn()
}

const mockTimerStore = {
  recentSessions: [] as any[],
  fetchRecentSessions: vi.fn(),
  createSession: vi.fn()
}

const mockAgentStore = {
  status: { configured: false },
  openDrawer: vi.fn(),
  runTool: vi.fn(),
}

vi.mock('@/stores/task', () => ({ useTaskStore: () => mockTaskStore }))
vi.mock('@/stores/timer', () => ({ useTimerStore: () => mockTimerStore }))
vi.mock('@/stores/agent', () => ({ useAgentStore: () => mockAgentStore }))

vi.mock('element-plus', () => ({
  ElMessage: { success: vi.fn(), error: vi.fn(), warning: vi.fn() }
}))

const elStubs = {
  'el-button': { template: '<button class="el-button" @click="$emit(\'click\')"><slot/></button>' },
  'el-icon': { template: '<i class="el-icon"><slot/></i>' },
  'el-time-select': { template: '<input class="el-time-select" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)"/>', props: ['modelValue', 'placeholder', 'start', 'step', 'end'] }
}

vi.mock('vue-router', () => {
  const push = vi.fn()
  return { useRouter: () => ({ push }) }
})

describe('Dashboard', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-05-25T10:00:00Z'))
    mockTaskStore.tasks = [...mockTasks]
    mockTaskStore.loading = false
    mockTaskStore.fetchTasks.mockResolvedValue(undefined)
    mockTimerStore.recentSessions = [...mockSessions]
    mockTimerStore.fetchRecentSessions.mockResolvedValue(undefined)
    mockTimerStore.createSession.mockResolvedValue({ planned_duration: 1500 })
    mockAgentStore.status.configured = false
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('initial render', () => {
    it('renders the dashboard title', () => {
      const wrapper = mount(Dashboard, { global: { stubs: elStubs } })
      expect(wrapper.find('h1').text()).toBe('仪表盘')
    })

    it('renders all stat cards', () => {
      const wrapper = mount(Dashboard, { global: { stubs: elStubs } })
      const statCards = wrapper.findAll('.stat-card')
      expect(statCards).toHaveLength(4)
    })

    it('renders quick action buttons', () => {
      const wrapper = mount(Dashboard, { global: { stubs: elStubs } })
      const buttons = wrapper.findAll('.quick-actions .el-button')
      expect(buttons.length).toBeGreaterThanOrEqual(2)
    })
  })

  describe('stats calculation', () => {
    it('calculates pending tasks count from todo items', async () => {
      mockTaskStore.tasks = [
        { ...mockTasks[0], status: 'todo' },
        { ...mockTasks[0], id: 't3', status: 'todo' },
        { ...mockTasks[1], status: 'completed' }
      ]
      const wrapper = mount(Dashboard, { global: { stubs: elStubs } })
      await nextTick()

      const statValues = wrapper.findAll('.stat-value')
      expect(statValues[3].text()).toBe('2') // pending tasks = 2
    })

    it('shows zero stats when no tasks or sessions', async () => {
      mockTaskStore.tasks = []
      mockTimerStore.recentSessions = []
      const wrapper = mount(Dashboard, { global: { stubs: elStubs } })
      await nextTick()

      const statValues = wrapper.findAll('.stat-value')
      expect(statValues[0].text()).toBe('0') // pomodoros
      expect(statValues[1].text()).toBe('0m') // focus time
      expect(statValues[2].text()).toBe('0') // completed
    })
  })

  describe('empty state', () => {
    it('shows empty state when no recent tasks', async () => {
      mockTaskStore.tasks = []
      const wrapper = mount(Dashboard, { global: { stubs: elStubs } })
      await nextTick()

      expect(wrapper.find('.empty-state').exists()).toBe(true)
      expect(wrapper.find('.empty-state p').text()).toBe('暂无任务')
    })

    it('shows create task button in empty state', async () => {
      mockTaskStore.tasks = []
      const wrapper = mount(Dashboard, { global: { stubs: elStubs } })
      await nextTick()

      const emptyButton = wrapper.find('.empty-state .el-button')
      expect(emptyButton.exists()).toBe(true)
    })
  })

  describe('task list', () => {
    it('renders recent tasks (max 5)', async () => {
      const wrapper = mount(Dashboard, { global: { stubs: { ...elStubs, TaskCard: { template: '<div class="task-card-mock">{{ task.title }}</div>', props: ['task'] } } } })
      await nextTick()

      const cards = wrapper.findAll('.task-card-mock')
      expect(cards).toHaveLength(2)
    })

    it('navigates to tasks with add param when empty state button is clicked', async () => {
      mockTaskStore.tasks = []
      const wrapper = mount(Dashboard, { global: { stubs: elStubs } })
      await nextTick()

      await wrapper.find('.empty-state .el-button').trigger('click')
      const { push } = useRouter()
      expect(push).toHaveBeenCalledWith('/tasks?add=true')
    })
  })

  describe('AI section', () => {
    it('shows priority card when AI is configured', async () => {
      mockAgentStore.status.configured = true
      const wrapper = mount(Dashboard, { global: { stubs: elStubs } })
      await nextTick()

      expect(wrapper.find('.priority-card').exists()).toBe(true)
    })
  })

  describe('formatDuration', () => {
    it('formats seconds to minutes', () => {
      const wrapper = mount(Dashboard, { global: { stubs: elStubs } })
      const vm = wrapper.vm as any
      expect(vm.formatDuration(60)).toBe('1m')
      expect(vm.formatDuration(1500)).toBe('25m')
    })

    it('formats seconds to hours and minutes', () => {
      const wrapper = mount(Dashboard, { global: { stubs: elStubs } })
      const vm = wrapper.vm as any
      expect(vm.formatDuration(3600)).toBe('1h')
      expect(vm.formatDuration(5400)).toBe('1h 30m')
    })
  })

  describe('onMounted data fetching', () => {
    it('fetches tasks and recent sessions on mount', async () => {
      mount(Dashboard, { global: { stubs: elStubs } })
      await nextTick()
      await nextTick()

      expect(mockTaskStore.fetchTasks).toHaveBeenCalled()
      expect(mockTimerStore.fetchRecentSessions).toHaveBeenCalled()
    })
  })

  describe('quick actions', () => {
    it('navigates to tasks with add param when 创建任务 button is clicked', async () => {
      const wrapper = mount(Dashboard, { global: { stubs: elStubs } })
      await nextTick()

      const buttons = wrapper.findAll('.quick-actions .el-button')
      // First button is 创建任务 (primary type)
      await buttons[0].trigger('click')
      const { push } = useRouter()
      expect(push).toHaveBeenCalledWith('/tasks?add=true')
    })

    it('creates session and navigates to timer when 开始番茄 button is clicked', async () => {
      const wrapper = mount(Dashboard, { global: { stubs: elStubs } })
      await nextTick()

      const buttons = wrapper.findAll('.quick-actions .el-button')
      // Second button is 开始番茄
      await buttons[1].trigger('click')

      expect(mockTimerStore.createSession).toHaveBeenCalledWith(null, 'work')
      await nextTick()
      const { push } = useRouter()
      expect(push).toHaveBeenCalledWith('/timer')
    })
  })
})
