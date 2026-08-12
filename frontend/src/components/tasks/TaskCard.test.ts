import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import TaskCard from './TaskCard.vue'
import type { Task } from '@/types'

const mockTask: Task = {
  id: 'task-1',
  title: '测试任务',
  description: '任务描述',
  quadrant: 1,
  is_important: true,
  is_urgent: true,
  status: 'todo',
  estimated_time: 30,
  deadline: '2026-05-25T00:00:00Z',
  tags: [],
  order: 0,
  created_at: '2026-05-21T09:00:00Z',
  updated_at: '2026-05-21T09:00:00Z',
  completed_at: null
}

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() })
}))

vi.mock('element-plus', () => ({
  ElMessage: { success: vi.fn(), error: vi.fn(), warning: vi.fn() }
}))

const elStubs = {
  'el-dropdown': { template: '<div><slot /></div>' },
  'el-dropdown-menu': true,
  'el-dropdown-item': true,
  'el-tag': true,
  'el-dialog': true,
  'el-button': true,
  'el-popover': { template: '<div><slot name="reference"/><slot/></div>' }
}

describe('TaskCard', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  describe('rendering', () => {
    it('should render task title', () => {
      const wrapper = mount(TaskCard, {
        props: { task: mockTask },
        global: { stubs: elStubs }
      })

      expect(wrapper.find('.task-title').text()).toBe('测试任务')
    })

    it('should render task description', () => {
      const wrapper = mount(TaskCard, {
        props: { task: mockTask },
        global: { stubs: elStubs }
      })

      expect(wrapper.find('.task-description').text()).toBe('任务描述')
    })

    it('should not render description when empty', () => {
      const wrapper = mount(TaskCard, {
        props: { task: { ...mockTask, description: '' } },
        global: { stubs: elStubs }
      })

      expect(wrapper.find('.task-description').exists()).toBe(false)
    })

    it('should apply completed class for completed task', () => {
      const wrapper = mount(TaskCard, {
        props: { task: { ...mockTask, status: 'completed' } },
        global: { stubs: elStubs }
      })

      expect(wrapper.find('.task-card').classes()).toContain('task-completed')
    })

    it('should not apply completed class for active task', () => {
      const wrapper = mount(TaskCard, {
        props: { task: { ...mockTask, status: 'in_progress' } },
        global: { stubs: elStubs }
      })

      expect(wrapper.find('.task-card').classes()).not.toContain('task-completed')
    })
  })

  describe('emits', () => {
    it('should emit show-detail when card is clicked', async () => {
      const wrapper = mount(TaskCard, {
        props: { task: mockTask },
        global: { stubs: elStubs }
      })

      await wrapper.find('.task-card').trigger('click')

      expect(wrapper.emitted('show-detail')).toBeTruthy()
      expect(wrapper.emitted('show-detail')![0]).toEqual([mockTask])
    })

    it('should emit drag-start on dragstart', async () => {
      const wrapper = mount(TaskCard, {
        props: { task: mockTask },
        global: { stubs: elStubs }
      })

      await wrapper.find('.task-card').trigger('dragstart')

      expect(wrapper.emitted('drag-start')).toBeTruthy()
    })
  })

  describe('formatDate', () => {
    it('should format date correctly', () => {
      const wrapper = mount(TaskCard, {
        props: { task: mockTask },
        global: { stubs: elStubs }
      })

      expect(wrapper.vm.formatDate('2026-12-25T00:00:00Z')).toBe('12/25')
    })
  })

  describe('row mode', () => {
    it('renders task-row class instead of task-card', () => {
      const wrapper = mount(TaskCard, {
        props: { task: mockTask, mode: 'row' },
        global: { stubs: elStubs }
      })

      expect(wrapper.find('.task-row').exists()).toBe(true)
      expect(wrapper.find('.task-card').exists()).toBe(false)
    })

    it('renders task title in row mode', () => {
      const wrapper = mount(TaskCard, {
        props: { task: mockTask, mode: 'row' },
        global: { stubs: elStubs }
      })

      expect(wrapper.find('.task-row .task-title').text()).toBe('测试任务')
    })

    it('renders checkbox in row mode', () => {
      const wrapper = mount(TaskCard, {
        props: { task: mockTask, mode: 'row' },
        global: { stubs: elStubs }
      })

      expect(wrapper.find('.row-checkbox').exists()).toBe(true)
    })

    it('renders estimated time pill when task has estimated_time', () => {
      const wrapper = mount(TaskCard, {
        props: { task: mockTask, mode: 'row' },
        global: { stubs: elStubs }
      })

      expect(wrapper.find('.row-time').exists()).toBe(true)
      expect(wrapper.find('.row-time').text()).toContain('30')
    })

    it('does not render estimated time pill when estimated_time is 0', () => {
      const wrapper = mount(TaskCard, {
        props: { task: { ...mockTask, estimated_time: 0 }, mode: 'row' },
        global: { stubs: elStubs }
      })

      expect(wrapper.find('.row-time').exists()).toBe(false)
    })

    it('renders deadline pill when task has deadline', () => {
      const wrapper = mount(TaskCard, {
        props: { task: mockTask, mode: 'row' },
        global: { stubs: elStubs }
      })

      expect(wrapper.find('.row-deadline').exists()).toBe(true)
    })

    it('does not render deadline pill when deadline is null', () => {
      const wrapper = mount(TaskCard, {
        props: { task: { ...mockTask, deadline: null }, mode: 'row' },
        global: { stubs: elStubs }
      })

      expect(wrapper.find('.row-deadline').exists()).toBe(false)
    })

    it('renders more menu icon in row mode', () => {
      const wrapper = mount(TaskCard, {
        props: { task: mockTask, mode: 'row' },
        global: { stubs: elStubs }
      })

      expect(wrapper.find('.row-more').exists()).toBe(true)
    })

    it('renders popover with description in row mode', () => {
      const wrapper = mount(TaskCard, {
        props: { task: mockTask, mode: 'row' },
        global: { stubs: elStubs }
      })

      expect(wrapper.text()).toContain('任务描述')
    })

    it('emits complete when row checkbox is clicked for non-completed task', async () => {
      const wrapper = mount(TaskCard, {
        props: { task: mockTask, mode: 'row' },
        global: { stubs: elStubs }
      })

      await wrapper.find('.row-checkbox').trigger('click')

      expect(wrapper.emitted('complete')).toBeTruthy()
      expect(wrapper.emitted('complete')![0]).toEqual(['task-1'])
    })
  })
})
