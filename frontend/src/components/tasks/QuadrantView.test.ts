import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import QuadrantView from './QuadrantView.vue'

const store = vi.hoisted(() => ({
  tasksByQuadrant: {
    1: [{ id: 't1', title: 'Task 1', quadrant: 1 }],
    2: [],
    3: [{ id: 't3', title: 'Task 3', quadrant: 3 }, { id: 't4', title: 'Task 4', quadrant: 3 }],
    4: []
  } as Record<number, Array<{ id: string; title: string; quadrant: number }>>,
  moveTask: vi.fn(),
  updateTask: vi.fn(),
  createTask: vi.fn(),
  markCompleted: vi.fn(),
  deleteTask: vi.fn()
}))

vi.mock('@/stores/task', () => ({
  useTaskStore: () => store
}))

describe('QuadrantView', () => {
  const TaskCardStub = {
  name: 'TaskCard',
  props: ['task', 'mode'],
  template: '<div class="task-card-stub" :data-mode="mode">{{ task.title }}</div>'
}
const stubs = { TaskCard: TaskCardStub, TaskForm: true }

  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  describe('rendering', () => {
    it('renders all 4 quadrants', () => {
      const wrapper = mount(QuadrantView, { global: { stubs } })
      expect(wrapper.findAll('.quadrant')).toHaveLength(4)
    })

    it('renders quadrant names', () => {
      const wrapper = mount(QuadrantView, { global: { stubs } })
      const names = wrapper.findAll('.quadrant-name')
      expect(names[0].text()).toBe('重要且紧急')
      expect(names[2].text()).toBe('紧急不重要')
    })

    it('renders task counts per quadrant', () => {
      const wrapper = mount(QuadrantView, { global: { stubs } })
      const counts = wrapper.findAll('.quadrant-count')
      expect(counts[0].text()).toBe('1')
      expect(counts[1].text()).toBe('0')
      expect(counts[2].text()).toBe('2')
      expect(counts[3].text()).toBe('0')
    })

    it('passes mode="row" to TaskCard', () => {
      const wrapper = mount(QuadrantView, { global: { stubs } })
      const cards = wrapper.findAll('.task-card-stub')
      expect(cards.length).toBeGreaterThan(0)
      cards.forEach(card => {
        expect(card.attributes('data-mode')).toBe('row')
      })
    })
  })

  describe('events from TaskCard', () => {
    it('calls store.markCompleted on complete event', async () => {
      store.markCompleted = vi.fn().mockResolvedValue(undefined)
      const wrapper = mount(QuadrantView, { global: { stubs } })
      const taskCard = wrapper.findAllComponents({ name: 'TaskCard' })

      await taskCard[0].vm.$emit('complete', 't1')

      expect(store.markCompleted).toHaveBeenCalledWith('t1')
    })

    it('calls store.deleteTask on delete event', async () => {
      store.deleteTask = vi.fn().mockResolvedValue(undefined)
      const wrapper = mount(QuadrantView, { global: { stubs } })
      const taskCard = wrapper.findAllComponents({ name: 'TaskCard' })

      await taskCard[0].vm.$emit('delete', 't1')

      expect(store.deleteTask).toHaveBeenCalledWith('t1')
    })
  })

  describe('drag and drop', () => {
    it('moves task when dropping on different quadrant', async () => {
      store.moveTask = vi.fn().mockResolvedValue(undefined)
      const wrapper = mount(QuadrantView, { global: { stubs } })

      await wrapper.vm.onDragStart({ dataTransfer: null } as any, { id: 't1', quadrant: 1 } as any)
      await wrapper.vm.onDrop({} as any, 2)

      expect(store.moveTask).toHaveBeenCalledWith('t1', 2)
    })

    it('does not move when dropping on same quadrant', async () => {
      store.moveTask = vi.fn().mockResolvedValue(undefined)
      const wrapper = mount(QuadrantView, { global: { stubs } })

      await wrapper.vm.onDragStart({ dataTransfer: null } as any, { id: 't1', quadrant: 1 } as any)
      await wrapper.vm.onDrop({} as any, 1)

      expect(store.moveTask).not.toHaveBeenCalled()
    })

    it('does not move when no task is dragged', async () => {
      store.moveTask = vi.fn().mockResolvedValue(undefined)
      const wrapper = mount(QuadrantView, { global: { stubs } })

      await wrapper.vm.onDrop({} as any, 2)

      expect(store.moveTask).not.toHaveBeenCalled()
    })
  })

  describe('defineExpose', () => {
    it('exposes onAddTask which shows form with null editingTask', () => {
      const wrapper = mount(QuadrantView, { global: { stubs } })

      expect(wrapper.vm.showForm).toBe(false)
      ;(wrapper.vm as any).onAddTask()
      expect(wrapper.vm.showForm).toBe(true)
      expect(wrapper.vm.editingTask).toBeNull()
    })
  })
})
