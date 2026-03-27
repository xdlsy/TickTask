<template>
  <div class="month-view">
    <div class="weekdays-header">
      <div v-for="day in weekDayNames" :key="day" class="weekday-cell">{{ day }}</div>
    </div>

    <div class="month-grid">
      <div
        v-for="(day, index) in calendarDays"
        :key="index"
        class="day-cell"
        :class="{
          'other-month': !day.isCurrentMonth,
          'is-today': day.isToday,
          'has-events': day.events.length > 0
        }"
        @click="onDayClick(day)"
      >
        <div class="day-header">
          <span class="day-number">{{ day.dayNumber }}</span>
          <span v-if="day.events.length > 0" class="event-count">{{ day.events.length }}</span>
        </div>

        <div class="day-events" v-if="day.events.length > 0">
          <div
            v-for="event in day.events.slice(0, 3)"
            :key="event.id"
            class="event-item"
            :style="{ backgroundColor: getEventColor(event) }"
            @click.stop="onEventClick(event)"
          >
            <span class="event-title">{{ event.title }}</span>
          </div>
          <div v-if="day.events.length > 3" class="more-events">
            +{{ day.events.length - 3 }} 更多
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ScheduleEvent } from '@/types'

const props = defineProps<{
  currentDate: Date
  events: ScheduleEvent[]
}>()

const emit = defineEmits<{
  'event-click': [event: ScheduleEvent]
  'day-click': [date: string]
}>()

const weekDayNames = ['周一', '周二', '周三', '周四', '周五', '周六', '周日']

// 解析 ISO 时间字符串为本地日期字符串
function parseLocalDate(isoString: string): string {
  const date = new Date(isoString)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

// 获取某日期的事件
function getEventsForDate(dateStr: string): ScheduleEvent[] {
  return props.events.filter(event => {
    const eventStartDate = parseLocalDate(event.start)
    const eventEndDate = parseLocalDate(event.end)
    return dateStr >= eventStartDate && dateStr <= eventEndDate
  })
}

// 生成日历网格
const calendarDays = computed(() => {
  const year = props.currentDate.getFullYear()
  const month = props.currentDate.getMonth()

  // 当月第一天
  const firstDay = new Date(year, month, 1)
  // 当月最后一天
  const lastDay = new Date(year, month + 1, 0)

  // 计算日历开始日期（从周一开始）
  const startOffset = (firstDay.getDay() + 6) % 7 // 调整为周一开始
  const startDate = new Date(firstDay)
  startDate.setDate(startDate.getDate() - startOffset)

  // 计算日历结束日期（到周日结束）
  const endOffset = (7 - lastDay.getDay()) % 7 || 7
  const endDate = new Date(lastDay)
  endDate.setDate(endDate.getDate() + endOffset)

  // 生成日期数组
  const days: Array<{
    date: Date
    dateStr: string
    dayNumber: number
    isCurrentMonth: boolean
    isToday: boolean
    events: ScheduleEvent[]
  }> = []

  const current = new Date(startDate)
  const today = new Date()
  const todayStr = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}-${String(today.getDate()).padStart(2, '0')}`

  while (current <= endDate) {
    const dateStr = `${current.getFullYear()}-${String(current.getMonth() + 1).padStart(2, '0')}-${String(current.getDate()).padStart(2, '0')}`

    days.push({
      date: new Date(current),
      dateStr,
      dayNumber: current.getDate(),
      isCurrentMonth: current.getMonth() === month,
      isToday: dateStr === todayStr,
      events: getEventsForDate(dateStr)
    })

    current.setDate(current.getDate() + 1)

    // 限制最多 6 周
    if (days.length >= 42) break
  }

  return days
})

function onDayClick(day: { dateStr: string; events: ScheduleEvent[] }) {
  emit('day-click', day.dateStr)
}

function onEventClick(event: ScheduleEvent) {
  emit('event-click', event)
}

function getEventColor(event: ScheduleEvent): string {
  if (event.color) return event.color
  const colors: Record<string, string> = {
    task: '#3b82f6',
    pomodoro: '#f59e0b',
    break: '#22c55e',
    custom: '#6b7280'
  }
  return colors[event.type] || '#3b82f6'
}
</script>

<style scoped>
.month-view {
  background: #fff;
  border-radius: 16px;
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.weekdays-header {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  background: #f8fafc;
  border-bottom: 1px solid #e5e7eb;
}

.weekday-cell {
  padding: 12px 8px;
  text-align: center;
  font-size: 13px;
  font-weight: 600;
  color: #6b7280;
}

.month-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
}

.day-cell {
  min-height: 100px;
  border-right: 1px solid #f3f4f6;
  border-bottom: 1px solid #f3f4f6;
  padding: 8px;
  cursor: pointer;
  transition: background 0.2s;
}

.day-cell:nth-child(7n) {
  border-right: none;
}

.day-cell:hover {
  background: #f8fafc;
}

.day-cell.other-month {
  background: #f9fafb;
}

.day-cell.other-month .day-number {
  color: #d1d5db;
}

.day-cell.is-today {
  background: linear-gradient(135deg, #eff6ff 0%, #dbeafe 100%);
}

.day-cell.is-today .day-number {
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
  color: #fff;
}

.day-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
}

.day-number {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  font-weight: 600;
  color: #374151;
  border-radius: 50%;
}

.event-count {
  font-size: 11px;
  color: #fff;
  background: #6b7280;
  padding: 2px 6px;
  border-radius: 10px;
  font-weight: 500;
}

.day-events {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.event-item {
  display: flex;
  align-items: center;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 11px;
  color: #fff;
  overflow: hidden;
  transition: transform 0.2s, box-shadow 0.2s;
  cursor: pointer;
}

.event-item:hover {
  transform: scale(1.02);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.15);
}

.event-title {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-weight: 500;
}

.more-events {
  font-size: 11px;
  color: #6b7280;
  padding: 2px 6px;
  text-align: center;
}
</style>