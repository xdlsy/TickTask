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
              <span v-if="getPomodoroText(event)" class="event-pomodoro-badge">{{ getPomodoroText(event) }}</span>
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
import type { ScheduleEvent, TaskResponse } from '@/types'

const props = defineProps<{
  currentDate: Date
  events: ScheduleEvent[]
  tasksMap?: Record<string, TaskResponse>
}>()

const emit = defineEmits<{
  'event-click': [event: ScheduleEvent]
  'day-click': [date: string]
}>()

const weekDayNames = ['周一', '周二', '周三', '周四', '周五', '周六', '周日']

function parseLocalDate(isoString: string): string {
  const date = new Date(isoString)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function getEventsForDate(dateStr: string): ScheduleEvent[] {
  return props.events.filter(event => {
    const eventStartDate = parseLocalDate(event.start)
    const eventEndDate = parseLocalDate(event.end)
    return dateStr >= eventStartDate && dateStr <= eventEndDate
  })
}

const calendarDays = computed(() => {
  const year = props.currentDate.getFullYear()
  const month = props.currentDate.getMonth()

  const firstDay = new Date(year, month, 1)
  const lastDay = new Date(year, month + 1, 0)

  const startOffset = (firstDay.getDay() + 6) % 7
  const startDate = new Date(firstDay)
  startDate.setDate(startDate.getDate() - startOffset)

  const endOffset = (7 - lastDay.getDay()) % 7 || 7
  const endDate = new Date(lastDay)
  endDate.setDate(endDate.getDate() + endOffset)

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
    if (days.length >= 42) break
  }

  return days
})

function onDayClick(day: { dateStr: string }) { emit('day-click', day.dateStr) }
function onEventClick(event: ScheduleEvent) { emit('event-click', event) }

function getEventColor(event: ScheduleEvent): string {
  if (event.color) return event.color
  const colors: Record<string, string> = {
    task: '#B8452C', pomodoro: '#B8954D', break: '#6B8B6F', custom: '#9C9893'
  }
  return colors[event.type] || '#B8452C'
}

function getPomodoroText(event: ScheduleEvent): string {
  if (!event.task_id || !props.tasksMap) return ''
  const task = props.tasksMap[event.task_id]
  if (!task || task.planned_pomodoros === 0) return ''
  return `${task.completed_pomodoros}/${task.planned_pomodoros}`
}
</script>

<style scoped>
.month-view {
  background: var(--bg-card);
  border-radius: var(--radius-xl);
  overflow: hidden;
  border: 1px solid var(--border-color);
}

.weekdays-header {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  border-bottom: 1px solid var(--border-color);
}

.weekday-cell {
  padding: 12px 8px;
  text-align: center;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-muted);
}

.month-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
}

.day-cell {
  min-height: 100px;
  border-right: 1px solid var(--border-color);
  border-bottom: 1px solid var(--border-color);
  padding: 10px;
  cursor: pointer;
  transition: background var(--transition-fast);
}

.day-cell:nth-child(7n) {
  border-right: none;
}

.day-cell:hover {
  background: rgba(0, 0, 0, 0.02);
}

.day-cell.other-month .day-number {
  color: var(--text-muted);
  opacity: 0.5;
}

.day-cell.is-today {
  background: rgba(184, 69, 44, 0.04);
}

.day-cell.is-today .day-number {
  background: var(--accent-primary);
  color: #fff;
}

.day-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
}

.day-number {
  width: 26px;
  height: 26px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
  border-radius: 50%;
}

.event-count {
  font-size: 10px;
  color: var(--text-muted);
  font-weight: 500;
  font-family: var(--font-mono);
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
  border-radius: 3px;
  font-size: 11px;
  color: #fff;
  overflow: hidden;
  cursor: pointer;
  transition: opacity var(--transition-fast);
}

.event-item:hover {
  opacity: 0.85;
}

.event-title {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-weight: 500;
  flex: 1;
  min-width: 0;
}

.event-pomodoro-badge {
  font-size: 9px;
  opacity: 0.8;
  flex-shrink: 0;
  margin-left: 2px;
}

.more-events {
  font-size: 11px;
  color: var(--text-muted);
  padding: 2px 6px;
}
</style>
