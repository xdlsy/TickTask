import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ListView from './ListView.vue'

const taskStore = vi.hoisted(() => ({
  tasks: [
    { id: 't1', title: 'Task 1', status: 'todo', quadrant: 1, description: 'desc1', deadline: '2026-06-01T00:00:00Z', estimated_time: 30, created_at: '2026-05-20T00:00:00Z', tags: [] },
    { id: 't2', title: 'Task 2', status: 'completed', quadrant: 2, description: '', deadline: '2026-05-15T00:00:00Z', estimated_time: 0, created_at: '2026-05-21T00:00:00Z', tags: [] },
    { id: 't3', title: 'Task 3', status: 'in_progress', quadrant: 3, description: 'desc3', deadline: null, estimated_time: 60, created_at: '2026-05-22T00:00:00Z', tags: [] },
    { id: 't4', title: 'Task 4', status: 'todo', quadrant: 1, description: '', deadline: null, estimated_time: 0, created_at: '2026-05-19T00:00:00Z', tags: [] }
  ],
  updateTask: vi.fn(),
  markCompleted: vi.fn(),
  createTask: vi.fn(),
  deleteTask: vi.fn()
}))

vi.mock('@/stores/task', () => ({
  useTaskStore: () => taskStore
}))

const timerStore = vi.hoisted(() => ({
  createSession: vi.fn()
}))

vi.mock('@/stores/timer', () => ({
  useTimerStore: () => timerStore
}))

const routerPush = vi.hoisted(() => vi.fn())
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: routerPush })
}))

const elMsg = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  warning: vi.fn()
}))

vi.mock('element-plus', () => ({
  ElMessage: elMsg
}))

const elStubs = {
  'el-select': true,
  'el-option': true,
  'el-dropdown': true,
  'el-dropdown-menu': true,
  'el-dropdown-item': true,
  TaskForm: true
}

describe('ListView', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  describe('filteredTasks', () => {
    it('returns all tasks by default', () => {
      const wrapper = mount(ListView, { global: { stubs: elStubs } })
      expect(wrapper.vm.filteredTasks).toHaveLength(4)
    })

    it('filters by status', async () => {
      const wrapper = mount(ListView, { global: { stubs: elStubs } })
      wrapper.vm.statusFilter = 'completed'
      await wrapper.vm.$nextTick()
      expect(wrapper.vm.filteredTasks).toHaveLength(1)
      expect(wrapper.vm.filteredTasks[0].id).toBe('t2')
    })

    it('filters by quadrant', async () => {
      const wrapper = mount(ListView, { global: { stubs: elStubs } })
      wrapper.vm.quadrantFilter = '1'
      await wrapper.vm.$nextTick()
      expect(wrapper.vm.filteredTasks).toHaveLength(2)
    })

    it('sorts by created_at descending', () => {
      const wrapper = mount(ListView, { global: { stubs: elStubs } })
      wrapper.vm.sortBy = 'created'
      const ids = wrapper.vm.filteredTasks.map((t: any) => t.id)
      expect(ids[0]).toBe('t3')
      expect(ids[3]).toBe('t4')
    })

    it('sorts by quadrant ascending', () => {
      const wrapper = mount(ListView, { global: { stubs: elStubs } })
      wrapper.vm.sortBy = 'quadrant'
      const quads = wrapper.vm.filteredTasks.map((t: any) => t.quadrant)
      expect(quads).toEqual([1, 1, 2, 3])
    })

    it('sorts by deadline with nulls last', () => {
      const wrapper = mount(ListView, { global: { stubs: elStubs } })
      wrapper.vm.sortBy = 'deadline'
      const ids = wrapper.vm.filteredTasks.map((t: any) => t.id)
      expect(ids[0]).toBe('t2')
      expect(ids[3]).toBe('t4')
    })

    it('renders empty state when no tasks match', async () => {
      const wrapper = mount(ListView, { global: { stubs: elStubs } })
      wrapper.vm.statusFilter = 'nonexistent' as any
      await wrapper.vm.$nextTick()
      expect(wrapper.find('.empty-state').exists()).toBe(true)
    })
  })

  describe('helpers', () => {
    it('quadrantLabel returns correct label', () => {
      const wrapper = mount(ListView, { global: { stubs: elStubs } })
      expect(wrapper.vm.quadrantLabel(1)).toBe('重要紧急')
      expect(wrapper.vm.quadrantLabel(4)).toBe('不重要不紧急')
    })

    it('isOverdue returns true for past dates', () => {
      const wrapper = mount(ListView, { global: { stubs: elStubs } })
      expect(wrapper.vm.isOverdue('2020-01-01T00:00:00Z')).toBe(true)
      expect(wrapper.vm.isOverdue('2099-01-01T00:00:00Z')).toBe(false)
    })
  })

  describe('onCompleteTask', () => {
    it('marks completed task as todo (reopen)', async () => {
      taskStore.updateTask = vi.fn().mockResolvedValue(undefined)
      const wrapper = mount(ListView, { global: { stubs: elStubs } })
      await wrapper.vm.onCompleteTask('t2')
      expect(taskStore.updateTask).toHaveBeenCalledWith('t2', { status: 'todo' })
    })

    it('marks todo task as completed', async () => {
      taskStore.markCompleted = vi.fn().mockResolvedValue(undefined)
      const wrapper = mount(ListView, { global: { stubs: elStubs } })
      await wrapper.vm.onCompleteTask('t1')
      expect(taskStore.markCompleted).toHaveBeenCalledWith('t1')
    })
  })

  describe('onSaveTask', () => {
    it('updates existing task', async () => {
      taskStore.updateTask = vi.fn().mockResolvedValue(undefined)
      const wrapper = mount(ListView, { global: { stubs: elStubs } })
      wrapper.vm.editingTask = taskStore.tasks[0] as any
      await wrapper.vm.onSaveTask({ title: 'Updated' })
      expect(taskStore.updateTask).toHaveBeenCalledWith('t1', { title: 'Updated' })
      expect(wrapper.vm.showForm).toBe(false)
    })

    it('creates new task when editingTask is null', async () => {
      taskStore.createTask = vi.fn().mockResolvedValue(undefined)
      const wrapper = mount(ListView, { global: { stubs: elStubs } })
      wrapper.vm.editingTask = null
      await wrapper.vm.onSaveTask({ title: 'New' })
      expect(taskStore.createTask).toHaveBeenCalledWith({ title: 'New' })
    })
  })

  describe('handleCommand', () => {
    it('edit sets editingTask and shows form', () => {
      const wrapper = mount(ListView, { global: { stubs: elStubs } })
      wrapper.vm.handleCommand('edit', taskStore.tasks[0] as any)
      expect(wrapper.vm.editingTask).toEqual(taskStore.tasks[0])
      expect(wrapper.vm.showForm).toBe(true)
    })

    it('complete calls markCompleted', async () => {
      taskStore.markCompleted = vi.fn().mockResolvedValue(undefined)
      const wrapper = mount(ListView, { global: { stubs: elStubs } })
      await wrapper.vm.handleCommand('complete', taskStore.tasks[0] as any)
      expect(taskStore.markCompleted).toHaveBeenCalledWith('t1')
    })

    it('reopen calls updateTask with todo', async () => {
      taskStore.updateTask = vi.fn().mockResolvedValue(undefined)
      const wrapper = mount(ListView, { global: { stubs: elStubs } })
      await wrapper.vm.handleCommand('reopen', taskStore.tasks[1] as any)
      expect(taskStore.updateTask).toHaveBeenCalledWith('t2', { status: 'todo' })
    })

    it('delete calls deleteTask', async () => {
      taskStore.deleteTask = vi.fn().mockResolvedValue(undefined)
      const wrapper = mount(ListView, { global: { stubs: elStubs } })
      await wrapper.vm.handleCommand('delete', taskStore.tasks[0] as any)
      expect(taskStore.deleteTask).toHaveBeenCalledWith('t1')
    })
  })

  describe('startTimerForTask', () => {
    it('starts timer, shows success, navigates to timer', async () => {
      timerStore.createSession = vi.fn().mockResolvedValue(undefined)
      const wrapper = mount(ListView, { global: { stubs: elStubs } })
      await wrapper.vm.startTimerForTask(taskStore.tasks[0] as any)
      expect(timerStore.createSession).toHaveBeenCalledWith('t1', 'work')
      expect(elMsg.success).toHaveBeenCalledWith('开始专注：Task 1')
      expect(routerPush).toHaveBeenCalledWith('/timer')
    })

    it('shows error on failure', async () => {
      timerStore.createSession = vi.fn().mockRejectedValue(new Error('fail'))
      const wrapper = mount(ListView, { global: { stubs: elStubs } })
      await wrapper.vm.startTimerForTask(taskStore.tasks[0] as any)
      expect(elMsg.error).toHaveBeenCalledWith('启动计时器失败')
    })
  })

  describe('defineExpose', () => {
    it('exposes onAddTask', () => {
      const wrapper = mount(ListView, { global: { stubs: elStubs } })
      expect(wrapper.vm.showForm).toBe(false)
      ;(wrapper.vm as any).onAddTask()
      expect(wrapper.vm.showForm).toBe(true)
      expect(wrapper.vm.editingTask).toBeNull()
    })
  })
})
