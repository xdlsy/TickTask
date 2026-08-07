import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import MonthView from './MonthView.vue'
import type { ScheduleEvent } from '@/types'

describe('MonthView', () => {
  // 辅助函数：创建模拟事件
  const createMockEvent = (overrides: Partial<ScheduleEvent> = {}): ScheduleEvent => ({
    id: '1',
    title: '测试事件',
    start: '2026-03-15T09:00:00',
    end: '2026-03-15T10:00:00',
    type: 'task',
    status: 'planned',
    color: '#3b82f6',
    allDay: false,
    editable: true,
    ...overrides
  })

  const testDate = new Date('2026-03-15')

  describe('基础渲染', () => {
    it('应该显示星期标题行', () => {
      const wrapper = mount(MonthView, {
        props: {
          currentDate: testDate,
          events: []
        }
      })

      const weekdayCells = wrapper.findAll('.weekday-cell')
      expect(weekdayCells.length).toBe(7)

      const weekdays = weekdayCells.map(c => c.text())
      expect(weekdays).toContain('周一')
      expect(weekdays).toContain('周日')
    })

    it('星期标题应从周一开始', () => {
      const wrapper = mount(MonthView, {
        props: {
          currentDate: testDate,
          events: []
        }
      })

      const weekdayCells = wrapper.findAll('.weekday-cell')
      expect(weekdayCells[0].text()).toBe('周一')
      expect(weekdayCells[6].text()).toBe('周日')
    })

    it('应该显示日历网格', () => {
      const wrapper = mount(MonthView, {
        props: {
          currentDate: testDate,
          events: []
        }
      })

      const dayCells = wrapper.findAll('.day-cell')
      // 网格应该是 35 或 42 个格子（5或6周）
      expect(dayCells.length).toBeGreaterThanOrEqual(35)
      expect(dayCells.length).toBeLessThanOrEqual(42)
    })
  })

  describe('当月日期显示', () => {
    it('当月日期应有正确样式', () => {
      const wrapper = mount(MonthView, {
        props: {
          currentDate: testDate,
          events: []
        }
      })

      const currentMonthCells = wrapper.findAll('.day-cell:not(.other-month)')
      expect(currentMonthCells.length).toBeGreaterThan(0)
    })

    it('非当月日期应有 other-month 样式', () => {
      const wrapper = mount(MonthView, {
        props: {
          currentDate: testDate,
          events: []
        }
      })

      const otherMonthCells = wrapper.findAll('.day-cell.other-month')
      expect(otherMonthCells.length).toBeGreaterThan(0)
    })

    it('应该显示正确的月份日期数量', () => {
      const wrapper = mount(MonthView, {
        props: {
          currentDate: testDate, // 2026年3月
          events: []
        }
      })

      // 2026年3月有31天
      const currentMonthCells = wrapper.findAll('.day-cell:not(.other-month)')
      expect(currentMonthCells.length).toBe(31)
    })
  })

  describe('当天高亮', () => {
    it('当天单元格应有 is-today 样式', () => {
      const today = new Date()

      const wrapper = mount(MonthView, {
        props: {
          currentDate: today,
          events: []
        }
      })

      const todayCell = wrapper.find('.day-cell.is-today')
      expect(todayCell.exists()).toBe(true)
    })

    it('非当月不应有 today 样式', () => {
      const wrapper = mount(MonthView, {
        props: {
          currentDate: new Date('2020-01-15'),
          events: []
        }
      })

      const todayCell = wrapper.find('.day-cell.is-today')
      expect(todayCell.exists()).toBe(false)
    })
  })

  describe('事件渲染', () => {
    it('应该渲染事件', () => {
      const events = [createMockEvent()]

      const wrapper = mount(MonthView, {
        props: {
          currentDate: testDate,
          events
        }
      })

      const eventItems = wrapper.findAll('.event-item')
      expect(eventItems.length).toBe(1)
    })

    it('应该显示事件标题', () => {
      const events = [createMockEvent({ title: '项目评审会议' })]

      const wrapper = mount(MonthView, {
        props: {
          currentDate: testDate,
          events
        }
      })

      expect(wrapper.text()).toContain('项目评审会议')
    })

    it('应该显示多个事件', () => {
      const events = [
        createMockEvent({ id: '1', start: '2026-03-15T09:00:00', end: '2026-03-15T10:00:00' }),
        createMockEvent({ id: '2', start: '2026-03-15T14:00:00', end: '2026-03-15T15:00:00' }),
        createMockEvent({ id: '3', start: '2026-03-16T10:00:00', end: '2026-03-16T11:00:00' })
      ]

      const wrapper = mount(MonthView, {
        props: {
          currentDate: testDate,
          events
        }
      })

      const eventItems = wrapper.findAll('.event-item')
      expect(eventItems.length).toBe(3)
    })

    it('同一天超过3个事件应显示"更多"', () => {
      const events = [
        createMockEvent({ id: '1', start: '2026-03-15T09:00:00', end: '2026-03-15T10:00:00' }),
        createMockEvent({ id: '2', start: '2026-03-15T10:00:00', end: '2026-03-15T11:00:00' }),
        createMockEvent({ id: '3', start: '2026-03-15T11:00:00', end: '2026-03-15T12:00:00' }),
        createMockEvent({ id: '4', start: '2026-03-15T14:00:00', end: '2026-03-15T15:00:00' }),
        createMockEvent({ id: '5', start: '2026-03-15T16:00:00', end: '2026-03-15T17:00:00' })
      ]

      const wrapper = mount(MonthView, {
        props: {
          currentDate: testDate,
          events
        }
      })

      expect(wrapper.text()).toContain('更多')
    })

    it('同一天最多显示3个事件', () => {
      const events = [
        createMockEvent({ id: '1', start: '2026-03-15T09:00:00', end: '2026-03-15T10:00:00' }),
        createMockEvent({ id: '2', start: '2026-03-15T10:00:00', end: '2026-03-15T11:00:00' }),
        createMockEvent({ id: '3', start: '2026-03-15T11:00:00', end: '2026-03-15T12:00:00' }),
        createMockEvent({ id: '4', start: '2026-03-15T14:00:00', end: '2026-03-15T15:00:00' })
      ]

      const wrapper = mount(MonthView, {
        props: {
          currentDate: testDate,
          events
        }
      })

      const eventItems = wrapper.findAll('.event-item')
      expect(eventItems.length).toBe(3)
    })
  })

  describe('事件颜色', () => {
    it('应该使用事件自定义颜色', () => {
      const events = [createMockEvent({ color: '#8b5cf6' })]

      const wrapper = mount(MonthView, {
        props: {
          currentDate: testDate,
          events
        }
      })

      const eventItem = wrapper.find('.event-item')
      expect(eventItem.attributes('style')).toContain('background-color: #8b5cf6')
    })

    it('番茄钟类型默认橙色', () => {
      const events = [createMockEvent({ color: '', type: 'pomodoro' })]

      const wrapper = mount(MonthView, {
        props: {
          currentDate: testDate,
          events
        }
      })

      const eventItem = wrapper.find('.event-item')
      expect(eventItem.attributes('style')).toContain('background-color: #B8954D')
    })

    it('休息类型默认绿色', () => {
      const events = [createMockEvent({ color: '', type: 'break' })]

      const wrapper = mount(MonthView, {
        props: {
          currentDate: testDate,
          events
        }
      })

      const eventItem = wrapper.find('.event-item')
      expect(eventItem.attributes('style')).toContain('background-color: #6B8B6F')
    })
  })

  describe('事件交互', () => {
    it('点击事件应触发 event-click', async () => {
      const event = createMockEvent()
      const wrapper = mount(MonthView, {
        props: {
          currentDate: testDate,
          events: [event]
        }
      })

      await wrapper.find('.event-item').trigger('click')

      expect(wrapper.emitted('event-click')).toBeTruthy()
      expect(wrapper.emitted('event-click')![0]).toEqual([event])
    })

    it('点击日期应触发 day-click', async () => {
      const wrapper = mount(MonthView, {
        props: {
          currentDate: testDate,
          events: []
        }
      })

      const dayCells = wrapper.findAll('.day-cell')
      await dayCells[0].trigger('click')

      expect(wrapper.emitted('day-click')).toBeTruthy()
    })
  })

  describe('事件计数', () => {
    it('有事件的日期应显示事件数量', () => {
      const events = [createMockEvent()]

      const wrapper = mount(MonthView, {
        props: {
          currentDate: testDate,
          events
        }
      })

      // 找到有事件的日期单元格
      const hasEventsCell = wrapper.find('.day-cell.has-events')
      expect(hasEventsCell.exists()).toBe(true)
    })

    it('显示正确的事件数量', () => {
      const events = [
        createMockEvent({ id: '1' }),
        createMockEvent({ id: '2', start: '2026-03-15T14:00:00', end: '2026-03-15T15:00:00' })
      ]

      const wrapper = mount(MonthView, {
        props: {
          currentDate: testDate,
          events
        }
      })

      // 应该显示数字 2
      const eventCount = wrapper.find('.event-count')
      expect(eventCount.text()).toBe('2')
    })
  })

  describe('边界情况', () => {
    it('空事件列表应正常渲染', () => {
      const wrapper = mount(MonthView, {
        props: {
          currentDate: testDate,
          events: []
        }
      })

      const eventItems = wrapper.findAll('.event-item')
      expect(eventItems.length).toBe(0)
    })

    it('跨日事件应在多天显示', () => {
      const events = [createMockEvent({
        start: '2026-03-15T09:00:00',
        end: '2026-03-17T18:00:00'
      })]

      const wrapper = mount(MonthView, {
        props: {
          currentDate: testDate,
          events
        }
      })

      const eventItems = wrapper.findAll('.event-item')
      // 应该在15、16、17号都显示
      expect(eventItems.length).toBe(3)
    })

    it('跨月事件应正确处理', () => {
      const events = [createMockEvent({
        start: '2026-03-30T09:00:00',
        end: '2026-04-02T18:00:00'
      })]

      const wrapper = mount(MonthView, {
        props: {
          currentDate: testDate, // 2026年3月
          events
        }
      })

      // 在3月视图应该只显示30和31号的事件（跨日事件在每天的格子中都会显示）
      const eventItems = wrapper.findAll('.event-item')
      expect(eventItems.length).toBeGreaterThanOrEqual(2)
    })
  })
})