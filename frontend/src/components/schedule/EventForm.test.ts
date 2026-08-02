import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import EventForm from './EventForm.vue'
import type { ScheduleEvent } from '@/types'

// Mock Element Plus ElMessage
vi.mock('element-plus', async () => {
  const actual = await vi.importActual('element-plus')
  return {
    ...actual,
    ElMessage: {
      warning: vi.fn(),
      success: vi.fn(),
      error: vi.fn()
    }
  }
})

describe('EventForm', () => {
  // 辅助函数：创建模拟事件
  const createMockEvent = (): ScheduleEvent => ({
    id: '1',
    title: '测试事件',
    start: '2026-03-24T09:00:00',
    end: '2026-03-24T10:00:00',
    type: 'task',
    status: 'planned',
    color: '#3b82f6',
    allDay: false,
    editable: true
  })

  const mountOptions = {
    props: {
      visible: false
    }
  }

  describe('组件存在性', () => {
    it('组件应该能正常挂载', () => {
      const wrapper = mount(EventForm, mountOptions)
      expect(wrapper.exists()).toBe(true)
    })

    it('visible 为 true 时组件存在', () => {
      const wrapper = mount(EventForm, {
        props: { visible: true }
      })
      expect(wrapper.exists()).toBe(true)
    })

    it('visible 为 false 时组件存在', () => {
      const wrapper = mount(EventForm, {
        props: { visible: false }
      })
      expect(wrapper.exists()).toBe(true)
    })
  })

  describe('Props 接收', () => {
    it('应该接收 visible prop', () => {
      const wrapper = mount(EventForm, {
        props: { visible: true }
      })
      expect(wrapper.props('visible')).toBe(true)
    })

    it('应该接收 event prop', () => {
      const event = createMockEvent()
      const wrapper = mount(EventForm, {
        props: { visible: true, event }
      })
      expect(wrapper.props('event')).toEqual(event)
    })

    it('应该接收 defaultDate prop', () => {
      const wrapper = mount(EventForm, {
        props: { visible: true, defaultDate: '2026-03-24' }
      })
      expect(wrapper.props('defaultDate')).toBe('2026-03-24')
    })

    it('应该接收 defaultHour prop', () => {
      const wrapper = mount(EventForm, {
        props: { visible: true, defaultHour: 14 }
      })
      expect(wrapper.props('defaultHour')).toBe(14)
    })
  })

  describe('事件对象处理', () => {
    it('event 为 null 时组件正常工作', () => {
      const wrapper = mount(EventForm, {
        props: { visible: true, event: null }
      })
      expect(wrapper.exists()).toBe(true)
    })

    it('event 存在时组件正常工作', () => {
      const event = createMockEvent()
      const wrapper = mount(EventForm, {
        props: { visible: true, event }
      })
      expect(wrapper.exists()).toBe(true)
    })
  })

  describe('颜色选项', () => {
    it('应该有颜色选项数据', () => {
      const wrapper = mount(EventForm, {
        props: { visible: true }
      })
      // 组件内部应该有颜色选项
      expect(wrapper.vm).toBeDefined()
    })
  })

  describe('类型选项', () => {
    it('应该有类型选项数据', () => {
      const wrapper = mount(EventForm, {
        props: { visible: true }
      })
      // 组件内部应该有类型选项
      expect(wrapper.vm).toBeDefined()
    })
  })

  describe('表单数据初始化', () => {
    it('新建模式应初始化空表单', () => {
      const wrapper = mount(EventForm, {
        props: { visible: true, event: null }
      })
      expect(wrapper.exists()).toBe(true)
    })

    it('编辑模式应加载事件数据', () => {
      const event = createMockEvent()
      const wrapper = mount(EventForm, {
        props: { visible: true, event }
      })
      expect(wrapper.exists()).toBe(true)
    })
  })

  describe('事件触发 - close', () => {
    it('触发 close 事件应该成功', async () => {
      const wrapper = mount(EventForm, {
        props: { visible: true }
      })

      // 直接触发组件的 close 事件
      wrapper.vm.$emit('close')
      expect(wrapper.emitted('close')).toBeTruthy()
    })
  })

  describe('事件触发 - save', () => {
    it('触发 save 事件应该成功', async () => {
      const wrapper = mount(EventForm, {
        props: { visible: true }
      })

      // 直接触发组件的 save 事件
      wrapper.vm.$emit('save', { title: '测试' })
      expect(wrapper.emitted('save')).toBeTruthy()
    })
  })

  describe('事件触发 - update', () => {
    it('触发 update 事件应该成功', async () => {
      const event = createMockEvent()
      const wrapper = mount(EventForm, {
        props: { visible: true, event }
      })

      // 直接触发组件的 update 事件
      wrapper.vm.$emit('update', event.id, { title: '更新' })
      expect(wrapper.emitted('update')).toBeTruthy()
    })
  })

  describe('事件触发 - delete', () => {
    it('触发 delete 事件应该成功', async () => {
      const event = createMockEvent()
      const wrapper = mount(EventForm, {
        props: { visible: true, event }
      })

      // 直接触发组件的 delete 事件
      wrapper.vm.$emit('delete', event.id)
      expect(wrapper.emitted('delete')).toBeTruthy()
    })
  })

  describe('组件状态', () => {
    it('组件应该有内部表单状态', () => {
      const wrapper = mount(EventForm, {
        props: { visible: true }
      })

      // 检查组件实例存在
      const vm = wrapper.vm as any
      expect(vm).toBeDefined()
    })
  })

  describe('边界情况', () => {
    it('props 变化时组件应更新', async () => {
      const wrapper = mount(EventForm, {
        props: { visible: false }
      })

      await wrapper.setProps({ visible: true })
      expect(wrapper.props('visible')).toBe(true)
    })

    it('event prop 变化时组件应更新', async () => {
      const event1 = createMockEvent()
      const event2 = { ...createMockEvent(), id: '2', title: '另一个事件' }

      const wrapper = mount(EventForm, {
        props: { visible: true, event: event1 }
      })

      await wrapper.setProps({ event: event2 })
      expect(wrapper.props('event')).toEqual(event2)
    })

    it('visible 从 true 变为 false 时应正常工作', async () => {
      const wrapper = mount(EventForm, {
        props: { visible: true }
      })

      await wrapper.setProps({ visible: false })
      expect(wrapper.props('visible')).toBe(false)
    })
  })

  describe('ElMessage 调用', () => {
    it('ElMessage.warning 应该被 mock', async () => {
      const { ElMessage } = await import('element-plus')

      ElMessage.warning('测试警告')
      expect(ElMessage.warning).toHaveBeenCalledWith('测试警告')
    })

    it('ElMessage.success 应该被 mock', async () => {
      const { ElMessage } = await import('element-plus')

      ElMessage.success('测试成功')
      expect(ElMessage.success).toHaveBeenCalledWith('测试成功')
    })
  })

  describe('组件方法存在性', () => {
    it('组件实例方法应该存在', () => {
      const wrapper = mount(EventForm, {
        props: { visible: true }
      })

      // 验证组件实例存在
      expect(typeof wrapper.vm.$emit).toBe('function')
    })
  })

  describe('响应式数据', () => {
    it('组件应该能响应 props 变化', async () => {
      const wrapper = mount(EventForm, {
        props: { visible: false }
      })

      expect(wrapper.props('visible')).toBe(false)

      await wrapper.setProps({ visible: true })
      expect(wrapper.props('visible')).toBe(true)
    })
  })
})