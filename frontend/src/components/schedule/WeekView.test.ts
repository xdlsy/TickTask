import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import WeekView from './WeekView.vue'
import type { ScheduleEvent } from '@/types'

describe('WeekView', () => {
  // 辅助函数：创建模拟事件
  const createMockEvent = (overrides: Partial<ScheduleEvent> = {}): ScheduleEvent => ({
    id: '1',
    title: '测试事件',
    start: '2026-03-23T09:00:00',
    end: '2026-03-23T10:00:00',
    type: 'task',
    status: 'planned',
    color: '#3b82f6',
    allDay: false,
    editable: true,
    ...overrides
  })

  // 2026-03-23 是周一
  const testDate = new Date('2026-03-25') // 周三

  describe('基础渲染', () => {
    it('应该显示7天列', () => {
      const wrapper = mount(WeekView, {
        props: {
          currentDate: testDate,
          events: []
        }
      })

      const dayColumns = wrapper.findAll('.day-column')
      expect(dayColumns.length).toBe(7)
    })

    it('应该显示星期标题', () => {
      const wrapper = mount(WeekView, {
        props: {
          currentDate: testDate,
          events: []
        }
      })

      const dayHeaders = wrapper.findAll('.day-header')
      const headerTexts = dayHeaders.map(h => h.text())

      expect(headerTexts.some(t => t.includes('周一'))).toBe(true)
      expect(headerTexts.some(t => t.includes('周二'))).toBe(true)
      expect(headerTexts.some(t => t.includes('周三'))).toBe(true)
      expect(headerTexts.some(t => t.includes('周四'))).toBe(true)
      expect(headerTexts.some(t => t.includes('周五'))).toBe(true)
      expect(headerTexts.some(t => t.includes('周六'))).toBe(true)
      expect(headerTexts.some(t => t.includes('周日'))).toBe(true)
    })

    it('应该从周一开始显示', () => {
      const wrapper = mount(WeekView, {
        props: {
          currentDate: testDate,
          events: []
        }
      })

      const dayHeaders = wrapper.findAll('.day-name')
      expect(dayHeaders[0].text()).toBe('周一')
      expect(dayHeaders[6].text()).toBe('周日')
    })

    it('应该显示正确的日期数字', () => {
      const wrapper = mount(WeekView, {
        props: {
          currentDate: new Date('2026-03-25'), // 周三
          events: []
        }
      })

      // 周视图应该显示 23-29 号（2026年3月）
      const dayNumbers = wrapper.findAll('.day-number')
      const numbers = dayNumbers.map(n => parseInt(n.text()))

      expect(numbers).toContain(23) // 周一
      expect(numbers).toContain(29) // 周日
    })
  })

  describe('时间轴', () => {
    it('应该显示24小时时间轴', () => {
      const wrapper = mount(WeekView, {
        props: {
          currentDate: testDate,
          events: []
        }
      })

      const timeSlots = wrapper.findAll('.time-slot')
      expect(timeSlots.length).toBe(24)
    })

    it('时间标签格式正确', () => {
      const wrapper = mount(WeekView, {
        props: {
          currentDate: testDate,
          events: []
        }
      })

      const timeLabels = wrapper.findAll('.time-label')
      expect(timeLabels[0].text()).toBe('00:00')
      expect(timeLabels[12].text()).toBe('12:00')
      expect(timeLabels[23].text()).toBe('23:00')
    })
  })

  describe('事件渲染', () => {
    it('应该渲染事件', () => {
      const events = [createMockEvent()]

      const wrapper = mount(WeekView, {
        props: {
          currentDate: testDate,
          events
        }
      })

      const eventBlocks = wrapper.findAll('.event-block')
      expect(eventBlocks.length).toBe(1)
    })

    it('应该将事件显示在正确的日期列', () => {
      const events = [
        createMockEvent({ id: '1', start: '2026-03-23T09:00:00', end: '2026-03-23T10:00:00' }) // 周一
      ]

      const wrapper = mount(WeekView, {
        props: {
          currentDate: new Date('2026-03-25'), // 周三，但周视图包含周一
          events
        }
      })

      const eventBlocks = wrapper.findAll('.event-block')
      expect(eventBlocks.length).toBe(1)
    })

    it('应该显示多个事件', () => {
      const events = [
        createMockEvent({ id: '1', start: '2026-03-23T09:00:00', end: '2026-03-23T10:00:00' }),
        createMockEvent({ id: '2', start: '2026-03-24T14:00:00', end: '2026-03-24T15:00:00' }),
        createMockEvent({ id: '3', start: '2026-03-25T10:00:00', end: '2026-03-25T11:00:00' })
      ]

      const wrapper = mount(WeekView, {
        props: {
          currentDate: new Date('2026-03-25'),
          events
        }
      })

      const eventBlocks = wrapper.findAll('.event-block')
      expect(eventBlocks.length).toBe(3)
    })

    it('应该正确显示事件标题', () => {
      const events = [createMockEvent({ title: '重要会议' })]

      const wrapper = mount(WeekView, {
        props: {
          currentDate: testDate,
          events
        }
      })

      expect(wrapper.text()).toContain('重要会议')
    })
  })

  describe('事件颜色', () => {
    it('应该使用事件自定义颜色', () => {
      const events = [createMockEvent({ color: '#ef4444' })]

      const wrapper = mount(WeekView, {
        props: {
          currentDate: testDate,
          events
        }
      })

      const eventBlock = wrapper.find('.event-block')
      expect(eventBlock.attributes('style')).toContain('background-color: #ef4444')
    })

    it('番茄钟类型默认橙色', () => {
      const events = [createMockEvent({ color: '', type: 'pomodoro' })]

      const wrapper = mount(WeekView, {
        props: {
          currentDate: testDate,
          events
        }
      })

      const eventBlock = wrapper.find('.event-block')
      expect(eventBlock.attributes('style')).toContain('background-color: #B8954D')
    })

    it('休息类型默认绿色', () => {
      const events = [createMockEvent({ color: '', type: 'break' })]

      const wrapper = mount(WeekView, {
        props: {
          currentDate: testDate,
          events
        }
      })

      const eventBlock = wrapper.find('.event-block')
      expect(eventBlock.attributes('style')).toContain('background-color: #6B8B6F')
    })
  })

  describe('重叠事件布局', () => {
    it('同一天的重叠事件应该并排显示', () => {
      const events = [
        createMockEvent({ id: '1', start: '2026-03-23T09:00:00', end: '2026-03-23T11:00:00' }),
        createMockEvent({ id: '2', start: '2026-03-23T10:00:00', end: '2026-03-23T12:00:00' })
      ]

      const wrapper = mount(WeekView, {
        props: {
          currentDate: new Date('2026-03-25'),
          events
        }
      })

      const eventBlocks = wrapper.findAll('.event-block')
      expect(eventBlocks.length).toBe(2)
    })
  })

  describe('事件交互', () => {
    it('点击事件应触发 event-click', async () => {
      const event = createMockEvent()
      const wrapper = mount(WeekView, {
        props: {
          currentDate: testDate,
          events: [event]
        }
      })

      await wrapper.find('.event-block').trigger('click')

      expect(wrapper.emitted('event-click')).toBeTruthy()
      expect(wrapper.emitted('event-click')![0]).toEqual([event])
    })

    it('点击时间段应触发 slot-click', async () => {
      const wrapper = mount(WeekView, {
        props: {
          currentDate: testDate,
          events: []
        }
      })

      const hourSlots = wrapper.findAll('.hour-slot')
      await hourSlots[0].trigger('click')

      expect(wrapper.emitted('slot-click')).toBeTruthy()
    })
  })

  describe('当天高亮', () => {
    it('当前日期列应有高亮样式', () => {
      const today = new Date()
      // 获取今天是周几
      const dayOfWeek = today.getDay()
      // 调整为周一开始（0=周一, 6=周日）
      const adjustedDay = dayOfWeek === 0 ? 6 : dayOfWeek - 1

      const wrapper = mount(WeekView, {
        props: {
          currentDate: today,
          events: []
        }
      })

      const dayColumns = wrapper.findAll('.day-column')
      expect(dayColumns[adjustedDay].classes()).toContain('is-today')
    })

    it('非当前周不应有高亮', () => {
      const wrapper = mount(WeekView, {
        props: {
          currentDate: new Date('2020-01-01'),
          events: []
        }
      })

      const todayColumns = wrapper.findAll('.day-column.is-today')
      expect(todayColumns.length).toBe(0)
    })
  })

  describe('边界情况', () => {
    it('空事件列表应正常渲染', () => {
      const wrapper = mount(WeekView, {
        props: {
          currentDate: testDate,
          events: []
        }
      })

      const eventBlocks = wrapper.findAll('.event-block')
      expect(eventBlocks.length).toBe(0)
    })

    it('跨日事件应正确显示', () => {
      const events = [createMockEvent({
        start: '2026-03-23T22:00:00',
        end: '2026-03-24T02:00:00'
      })]

      const wrapper = mount(WeekView, {
        props: {
          currentDate: new Date('2026-03-25'),
          events
        }
      })

      // 跨日事件应该在两天都显示
      const eventBlocks = wrapper.findAll('.event-block')
      expect(eventBlocks.length).toBe(2)
    })

    it('事件在周范围外不应显示', () => {
      const events = [
        createMockEvent({ id: '1', start: '2026-03-20T09:00:00', end: '2026-03-20T10:00:00' }), // 上周
        createMockEvent({ id: '2', start: '2026-03-30T09:00:00', end: '2026-03-30T10:00:00' })  // 下周
      ]

      const wrapper = mount(WeekView, {
        props: {
          currentDate: new Date('2026-03-25'), // 周三，周范围 23-29
          events
        }
      })

      const eventBlocks = wrapper.findAll('.event-block')
      expect(eventBlocks.length).toBe(0)
    })
  })
})