<template>
  <div class="week-view">
    <div class="week-header">
      <div class="time-axis-header"></div>
      <div class="days-header">
        <div
          v-for="day in weekDays"
          :key="day.date"
          class="day-header"
          :class="{ 'is-today': isToday(day.date) }"
        >
          <span class="day-name">{{ day.dayName }}</span>
          <span class="day-number">{{ day.dayNumber }}</span>
        </div>
      </div>
    </div>
    <div class="week-body">
      <div class="time-axis">
        <div v-for="hour in hours" :key="hour" class="time-slot">
          <span class="time-label">{{ formatHour(hour) }}</span>
        </div>
      </div>
      <div class="days-container">
        <div
          v-for="day in weekDays"
          :key="day.date"
          class="day-column"
          :class="{ 'is-today': isToday(day.date) }"
        >
          <div class="day-content">
            <div
              v-for="hour in hours"
              :key="hour"
              class="hour-slot"
              @click="onSlotClick(day.date, hour)"
            ></div>
            <div
              v-for="item in getEventLayoutsForDay(day.date)"
              :key="item.event.id + '-' + day.date"
              class="event-block"
              :style="item.style"
              @click="onEventClick(item.event)"
              @mouseenter="showTooltip($event, item.event, day.date)"
              @mouseleave="hideTooltip"
            >
              <span class="event-time">{{ formatEventTime(item.event, day.date) }}</span>
              <span class="event-title">{{ item.event.title }}</span>
              <span v-if="getPomodoroText(item.event)" class="event-pomodoro">{{ getPomodoroText(item.event) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <Teleport to="body">
      <div v-if="tooltipVisible" class="event-tooltip week-tooltip" :style="tooltipStyle">
        <div class="tooltip-time">{{ tooltipData.time }}</div>
        <div class="tooltip-title">{{ tooltipData.title }}</div>
        <div v-if="tooltipData.status" class="tooltip-status">{{ tooltipData.status }}</div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onUnmounted } from 'vue'
import type { ScheduleEvent, TaskResponse } from '@/types'

const props = defineProps<{
  currentDate: Date
  events: ScheduleEvent[]
  tasksMap?: Record<string, TaskResponse>
}>()

const emit = defineEmits<{
  'event-click': [event: ScheduleEvent]
  'slot-click': [date: string, hour: number]
}>()

const hours = Array.from({ length: 24 }, (_, i) => i)

const weekDays = computed(() => {
  const days = []
  const startOfWeek = new Date(props.currentDate)
  const dayOfWeek = startOfWeek.getDay()
  startOfWeek.setDate(startOfWeek.getDate() - dayOfWeek + 1)

  for (let i = 0; i < 7; i++) {
    const date = new Date(startOfWeek)
    date.setDate(startOfWeek.getDate() + i)
    days.push({
      date: date.toISOString().split('T')[0],
      dayName: ['周日', '周一', '周二', '周三', '周四', '周五', '周六'][date.getDay()],
      dayNumber: date.getDate()
    })
  }
  return days
})

function formatHour(hour: number): string {
  return `${hour.toString().padStart(2, '0')}:00`
}

function formatEventTime(event: ScheduleEvent, dayDate: string): string {
  const startLocal = parseLocalTime(event.start)
  const endLocal = parseLocalTime(event.end)
  let timeStr = ''
  if (dayDate === startLocal.dateStr) {
    timeStr = `${startLocal.hours.toString().padStart(2, '0')}:${startLocal.minutes.toString().padStart(2, '0')}`
  }
  if (dayDate === endLocal.dateStr) {
    const endTime = `${endLocal.hours.toString().padStart(2, '0')}:${endLocal.minutes.toString().padStart(2, '0')}`
    timeStr = timeStr ? `${timeStr}-${endTime}` : endTime
  }
  return timeStr
}

function isToday(dateStr: string): boolean {
  return dateStr === new Date().toISOString().split('T')[0]
}

function parseLocalTime(isoString: string): { hours: number; minutes: number; dateStr: string } {
  const date = new Date(isoString)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return { hours: date.getHours(), minutes: date.getMinutes(), dateStr: `${year}-${month}-${day}` }
}

function getEventsForDay(dateStr: string): ScheduleEvent[] {
  return props.events.filter(event => {
    const startLocal = parseLocalTime(event.start)
    const endLocal = parseLocalTime(event.end)
    return dateStr >= startLocal.dateStr && dateStr <= endLocal.dateStr
  })
}

interface EventLayout {
  event: ScheduleEvent
  style: Record<string, string>
}

function getEventLayoutsForDay(dateStr: string): EventLayout[] {
  const dayEvents = getEventsForDay(dateStr)
  if (dayEvents.length === 0) return []

  const sortedEvents = [...dayEvents].sort((a, b) => {
    const startA = new Date(a.start).getTime()
    const startB = new Date(b.start).getTime()
    return startA - startB
  })

  const eventTimes = sortedEvents.map(event => {
    const startLocal = parseLocalTime(event.start)
    const endLocal = parseLocalTime(event.end)
    let startHour: number
    if (dateStr === startLocal.dateStr) {
      startHour = startLocal.hours + startLocal.minutes / 60
    } else {
      startHour = 0
    }
    let endHour: number
    if (dateStr === endLocal.dateStr) {
      endHour = endLocal.hours + endLocal.minutes / 60
    } else {
      endHour = 24
    }
    return { event, startHour, endHour }
  })

  const columns: Array<Array<typeof eventTimes[0]>> = []
  for (const item of eventTimes) {
    let placed = false
    for (let i = 0; i < columns.length; i++) {
      const lastInColumn = columns[i][columns[i].length - 1]
      if (item.startHour >= lastInColumn.endHour) {
        columns[i].push(item)
        placed = true
        break
      }
    }
    if (!placed) {
      columns.push([item])
    }
  }

  const totalColumns = columns.length
  const gap = 2
  const margin = 4
  const layouts: EventLayout[] = []

  for (let colIndex = 0; colIndex < columns.length; colIndex++) {
    for (const item of columns[colIndex]) {
      const top = item.startHour * 60
      const height = Math.max((item.endHour - item.startHour) * 60, 30)
      const color = item.event.color || getEventColorByType(item.event.type)
      const leftCalc = `calc(${margin}px + ${colIndex} * ((100% - ${margin * 2}px - ${(totalColumns - 1) * gap}px) / ${totalColumns}) + ${colIndex * gap}px)`
      const widthCalc = `calc((100% - ${margin * 2}px - ${(totalColumns - 1) * gap}px) / ${totalColumns})`

      layouts.push({
        event: item.event,
        style: { top: `${top}px`, height: `${height}px`, left: leftCalc, width: widthCalc, backgroundColor: color }
      })
    }
  }

  return layouts
}

function getEventColorByType(type: string): string {
  const colors: Record<string, string> = {
    task: '#B8452C', pomodoro: '#B8954D', break: '#6B8B6F', custom: '#9C9893'
  }
  return colors[type] || '#B8452C'
}

function onSlotClick(date: string, hour: number) { emit('slot-click', date, hour) }
function onEventClick(event: ScheduleEvent) { emit('event-click', event) }

const tooltipVisible = ref(false)
const tooltipStyle = ref<Record<string, string>>({})
const tooltipData = ref({ time: '', title: '', status: '' })
let tooltipTimer: ReturnType<typeof setTimeout> | null = null

function showTooltip(event: MouseEvent, scheduleEvent: ScheduleEvent, dayDate: string) {
  tooltipTimer = setTimeout(() => {
    const rect = (event.target as HTMLElement).getBoundingClientRect()
    const timeStr = formatEventTime(scheduleEvent, dayDate)
    tooltipData.value = {
      time: timeStr || formatTimeRange(scheduleEvent),
      title: scheduleEvent.title,
      status: scheduleEvent.status !== 'planned' ? getStatusLabel(scheduleEvent.status) : ''
    }
    tooltipStyle.value = {
      position: 'fixed', left: `${rect.left}px`, top: `${rect.top - 10}px`, transform: 'translateY(-100%)'
    }
    tooltipVisible.value = true
  }, 500)
}

function hideTooltip() {
  if (tooltipTimer) { clearTimeout(tooltipTimer); tooltipTimer = null }
  tooltipVisible.value = false
}

function formatTimeRange(event: ScheduleEvent): string {
  const startLocal = parseLocalTime(event.start)
  const endLocal = parseLocalTime(event.end)
  return `${startLocal.hours.toString().padStart(2, '0')}:${startLocal.minutes.toString().padStart(2, '0')} - ${endLocal.hours.toString().padStart(2, '0')}:${endLocal.minutes.toString().padStart(2, '0')}`
}

function getStatusLabel(status: string): string {
  const labels: Record<string, string> = { in_progress: '进行中', completed: '已完成', cancelled: '已取消' }
  return labels[status] || ''
}

function getPomodoroText(event: ScheduleEvent): string {
  if (!event.task_id || !props.tasksMap) return ''
  const task = props.tasksMap[event.task_id]
  if (!task || task.planned_pomodoros === 0) return ''
  return `${task.completed_pomodoros}/${task.planned_pomodoros} 番茄钟`
}

onUnmounted(() => { if (tooltipTimer) clearTimeout(tooltipTimer) })
</script>

<style scoped>
.week-view {
  display: flex;
  flex-direction: column;
  background: var(--bg-card);
  border-radius: var(--radius-xl);
  overflow: hidden;
  border: 1px solid var(--border-color);
}

.week-header {
  display: flex;
  border-bottom: 1px solid var(--border-color);
}

.time-axis-header {
  width: 60px;
  flex-shrink: 0;
}

.days-header {
  display: flex;
  flex: 1;
}

.week-body {
  display: flex;
  overflow-y: auto;
  max-height: 600px;
}

.time-axis {
  width: 60px;
  flex-shrink: 0;
  border-right: 1px solid var(--border-color);
}

.time-slot {
  height: 60px;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 4px;
}

.time-label {
  font-size: 11px;
  color: var(--text-muted);
  font-family: var(--font-mono);
}

.days-container {
  display: flex;
  flex: 1;
}

.day-column {
  flex: 1;
  min-width: 120px;
  border-right: 1px solid var(--border-color);
}

.day-column:last-child {
  border-right: none;
}

.day-header {
  padding: 10px 8px;
  text-align: center;
  flex: 1;
}

.day-column.is-today .day-header {
  background: rgba(184, 69, 44, 0.06);
}

.day-name {
  display: block;
  font-size: 11px;
  color: var(--text-secondary);
  margin-bottom: 2px;
}

.day-column.is-today .day-name {
  color: var(--accent-primary);
  font-weight: 600;
}

.day-number {
  display: block;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
}

.day-column.is-today .day-number {
  color: var(--accent-primary);
}

.day-content {
  position: relative;
  height: 1440px;
}

.hour-slot {
  height: 60px;
  border-bottom: 1px solid var(--border-color);
  cursor: pointer;
  transition: background var(--transition-fast);
}

.hour-slot:hover {
  background: rgba(0, 0, 0, 0.02);
}

.event-block {
  position: absolute;
  border-radius: 4px;
  padding: 3px 6px;
  cursor: pointer;
  overflow: hidden;
  transition: opacity var(--transition-fast);
  color: #fff;
  font-size: 11px;
}

.event-block:hover {
  opacity: 0.9;
}

.event-time {
  font-size: 10px;
  opacity: 0.85;
  display: block;
  margin-bottom: 1px;
}

.event-title {
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.event-pomodoro {
  display: block;
  font-size: 9px;
  opacity: 0.85;
  margin-top: 1px;
}
</style>

<style>
.week-tooltip.event-tooltip {
  background: var(--bg-elevated);
  color: var(--text-primary);
  padding: 10px 14px;
  border-radius: var(--radius-md);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
  border: 1px solid var(--border-color);
  z-index: 9999;
  pointer-events: none;
  min-width: 150px;
  max-width: 280px;
}

.week-tooltip .tooltip-time {
  font-size: 12px;
  color: var(--text-secondary);
  margin-bottom: 4px;
}

.week-tooltip .tooltip-title {
  font-size: 14px;
  font-weight: 600;
  line-height: 1.4;
  word-break: break-word;
}

.week-tooltip .tooltip-status {
  font-size: 11px;
  color: var(--text-secondary);
  margin-top: 6px;
  padding-top: 6px;
  border-top: 1px solid var(--border-color);
}
</style>
