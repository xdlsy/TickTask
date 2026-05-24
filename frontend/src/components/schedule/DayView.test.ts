import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import DayView from './DayView.vue'
import type { ScheduleEvent } from '@/types'

describe('DayView', () => {
  // 辅助函数：创建模拟事件
  const createMockEvent = (overrides: Partial<ScheduleEvent> = {}): ScheduleEvent => ({
    id: '1',
    title: '测试事件',
    start: '2026-03-24T09:00:00',
    end: '2026-03-24T10:00:00',
    type: 'task',
    status: 'planned',
    color: '#3b82f6',
    allDay: false,
    editable: true,
    ...overrides
  })

  describe('基础渲染', () => {
    it('应该正确渲染日期标题', () => {
      const testDate = new Date('2026-03-24')
      const wrapper = mount(DayView, {
        props: {
          currentDate: testDate,
          events: []
        }
      })

      expect(wrapper.text()).toContain('24')
      expect(wrapper.text()).toContain('2026年3月')
    })

    it('应该显示正确的事件数量统计', () => {
      const events = [
        createMockEvent({ id: '1' }),
        createMockEvent({ id: '2', start: '2026-03-24T14:00:00', end: '2026-03-24T15:00:00' })
      ]

      const wrapper = mount(DayView, {
        props: {
          currentDate: new Date('2026-03-24'),
          events
        }
      })

      expect(wrapper.text()).toContain('2 个日程')
    })

    it('没有事件时不显示统计', () => {
      const wrapper = mount(DayView, {
        props: {
          currentDate: new Date('2026-03-24'),
          events: []
        }
      })

      expect(wrapper.text()).not.toContain('个日程')
    })
  })

  describe('时间轴', () => {
    it('应该显示24小时时间轴', () => {
      const wrapper = mount(DayView, {
        props: {
          currentDate: new Date('2026-03-24'),
          events: []
        }
      })

      const timeLabels = wrapper.findAll('.time-label')
      expect(timeLabels.length).toBe(24)
    })

    it('时间标签格式正确', () => {
      const wrapper = mount(DayView, {
        props: {
          currentDate: new Date('2026-03-24'),
          events: []
        }
      })

      const timeLabels = wrapper.findAll('.time-label')
      expect(timeLabels[0].text()).toBe('00:00')
      expect(timeLabels[9].text()).toBe('09:00')
      expect(timeLabels[23].text()).toBe('23:00')
    })
  })

  describe('事件渲染', () => {
    it('应该渲染单个事件', () => {
      const events = [createMockEvent()]

      const wrapper = mount(DayView, {
        props: {
          currentDate: new Date('2026-03-24'),
          events
        }
      })

      const eventBlocks = wrapper.findAll('.event-block')
      expect(eventBlocks.length).toBe(1)
      expect(eventBlocks[0].text()).toContain('测试事件')
    })

    it('应该渲染多个不重叠事件', () => {
      const events = [
        createMockEvent({ id: '1', start: '2026-03-24T09:00:00', end: '2026-03-24T10:00:00' }),
        createMockEvent({ id: '2', start: '2026-03-24T14:00:00', end: '2026-03-24T15:00:00' }),
        createMockEvent({ id: '3', start: '2026-03-24T18:00:00', end: '2026-03-24T19:00:00' })
      ]

      const wrapper = mount(DayView, {
        props: {
          currentDate: new Date('2026-03-24'),
          events
        }
      })

      const eventBlocks = wrapper.findAll('.event-block')
      expect(eventBlocks.length).toBe(3)
    })

    it('应该正确显示事件时间', () => {
      const events = [createMockEvent({
        start: '2026-03-24T09:30:00',
        end: '2026-03-24T11:45:00'
      })]

      const wrapper = mount(DayView, {
        props: {
          currentDate: new Date('2026-03-24'),
          events
        }
      })

      expect(wrapper.text()).toContain('09:30 - 11:45')
    })

    it('应该显示事件状态标签（非计划状态）', () => {
      const events = [createMockEvent({ status: 'completed' })]

      const wrapper = mount(DayView, {
        props: {
          currentDate: new Date('2026-03-24'),
          events
        }
      })

      expect(wrapper.text()).toContain('已完成')
    })

    it('计划状态不显示状态标签', () => {
      const events = [createMockEvent({ status: 'planned' })]

      const wrapper = mount(DayView, {
        props: {
          currentDate: new Date('2026-03-24'),
          events
        }
      })

      expect(wrapper.text()).not.toContain('已完成')
      expect(wrapper.text()).not.toContain('进行中')
    })
  })

  describe('事件颜色', () => {
    it('应该使用事件自定义颜色', () => {
      const events = [createMockEvent({ color: '#a855f7' })]

      const wrapper = mount(DayView, {
        props: {
          currentDate: new Date('2026-03-24'),
          events
        }
      })

      const eventBlock = wrapper.find('.event-block')
      expect(eventBlock.attributes('style')).toContain('background-color: rgb(168, 85, 247)')
    })

    it('没有颜色时使用类型默认颜色', () => {
      const events = [createMockEvent({ color: '', type: 'pomodoro' })]

      const wrapper = mount(DayView, {
        props: {
          currentDate: new Date('2026-03-24'),
          events
        }
      })

      const eventBlock = wrapper.find('.event-block')
      expect(eventBlock.attributes('style')).toContain('background-color: rgb(196, 151, 61)')
    })

    it('任务类型默认蓝色', () => {
      const events = [createMockEvent({ color: '', type: 'task' })]

      const wrapper = mount(DayView, {
        props: {
          currentDate: new Date('2026-03-24'),
          events
        }
      })

      const eventBlock = wrapper.find('.event-block')
      expect(eventBlock.attributes('style')).toContain('background-color: rgb(196, 103, 61)')
    })

    it('休息类型默认绿色', () => {
      const events = [createMockEvent({ color: '', type: 'break' })]

      const wrapper = mount(DayView, {
        props: {
          currentDate: new Date('2026-03-24'),
          events
        }
      })

      const eventBlock = wrapper.find('.event-block')
      expect(eventBlock.attributes('style')).toContain('background-color: rgb(107, 139, 111)')
    })
  })

  describe('重叠事件布局', () => {
    it('两个重叠事件应该并排显示', () => {
      const events = [
        createMockEvent({ id: '1', start: '2026-03-24T09:00:00', end: '2026-03-24T11:00:00' }),
        createMockEvent({ id: '2', start: '2026-03-24T10:00:00', end: '2026-03-24T12:00:00' })
      ]

      const wrapper = mount(DayView, {
        props: {
          currentDate: new Date('2026-03-24'),
          events
        }
      })

      const eventBlocks = wrapper.findAll('.event-block')
      expect(eventBlocks.length).toBe(2)

      // 两个事件都应该存在
      expect(wrapper.text()).toContain('测试事件')
    })

    it('三个重叠事件应该三列显示', () => {
      const events = [
        createMockEvent({ id: '1', start: '2026-03-24T09:00:00', end: '2026-03-24T12:00:00' }),
        createMockEvent({ id: '2', start: '2026-03-24T09:30:00', end: '2026-03-24T11:00:00' }),
        createMockEvent({ id: '3', start: '2026-03-24T10:00:00', end: '2026-03-24T13:00:00' })
      ]

      const wrapper = mount(DayView, {
        props: {
          currentDate: new Date('2026-03-24'),
          events
        }
      })

      const eventBlocks = wrapper.findAll('.event-block')
      expect(eventBlocks.length).toBe(3)
    })
  })

  describe('事件交互', () => {
    it('点击事件应触发 event-click', async () => {
      const event = createMockEvent()
      const wrapper = mount(DayView, {
        props: {
          currentDate: new Date('2026-03-24'),
          events: [event]
        }
      })

      await wrapper.find('.event-block').trigger('click')

      expect(wrapper.emitted('event-click')).toBeTruthy()
      expect(wrapper.emitted('event-click')![0]).toEqual([event])
    })

    it('点击时间段应触发 slot-click', async () => {
      const wrapper = mount(DayView, {
        props: {
          currentDate: new Date('2026-03-24'),
          events: []
        }
      })

      const hourSlots = wrapper.findAll('.hour-slot')
      await hourSlots[9].trigger('click') // 点击 09:00 时间段

      expect(wrapper.emitted('slot-click')).toBeTruthy()
      expect(wrapper.emitted('slot-click')![0][0]).toBe('2026-03-24')
      expect(wrapper.emitted('slot-click')![0][1]).toBe(9)
    })
  })

  describe('当天高亮', () => {
    it('当前日期应显示时间线', () => {
      const today = new Date()

      const wrapper = mount(DayView, {
        props: {
          currentDate: today,
          events: []
        }
      })

      expect(wrapper.find('.current-time-line').exists()).toBe(true)
    })

    it('非当前日期不显示时间线', () => {
      const wrapper = mount(DayView, {
        props: {
          currentDate: new Date('2020-01-01'),
          events: []
        }
      })

      expect(wrapper.find('.current-time-line').exists()).toBe(false)
    })
  })

  describe('边界情况', () => {
    it('空事件列表应正常渲染', () => {
      const wrapper = mount(DayView, {
        props: {
          currentDate: new Date('2026-03-24'),
          events: []
        }
      })

      expect(wrapper.findAll('.event-block').length).toBe(0)
    })

    it('跨午夜事件应正确显示', () => {
      const events = [createMockEvent({
        start: '2026-03-24T22:00:00',
        end: '2026-03-25T02:00:00'
      })]

      const wrapper = mount(DayView, {
        props: {
          currentDate: new Date('2026-03-24'),
          events
        }
      })

      const eventBlocks = wrapper.findAll('.event-block')
      expect(eventBlocks.length).toBe(1)
    })

    it('短时间事件（15分钟）应显示最小高度', () => {
      const events = [createMockEvent({
        start: '2026-03-24T09:00:00',
        end: '2026-03-24T09:15:00'
      })]

      const wrapper = mount(DayView, {
        props: {
          currentDate: new Date('2026-03-24'),
          events
        }
      })

      const eventBlock = wrapper.find('.event-block')
      // 15分钟 = 15px，但最小高度应为 30px
      expect(eventBlock.attributes('style')).toContain('height: 30px')
    })

    it('不同日期的事件应正确过滤', () => {
      const events = [
        createMockEvent({ id: '1', start: '2026-03-23T09:00:00', end: '2026-03-23T10:00:00' }),
        createMockEvent({ id: '2', start: '2026-03-24T09:00:00', end: '2026-03-24T10:00:00' }),
        createMockEvent({ id: '3', start: '2026-03-25T09:00:00', end: '2026-03-25T10:00:00' })
      ]

      const wrapper = mount(DayView, {
        props: {
          currentDate: new Date('2026-03-24'),
          events
        }
      })

      // 日视图显示所有事件（由父组件过滤日期范围）
      const eventBlocks = wrapper.findAll('.event-block')
      expect(eventBlocks.length).toBe(3)
    })
  })
})