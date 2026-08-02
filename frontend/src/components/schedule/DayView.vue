<template>
  <div class="day-view">
    <div class="day-header">
      <div class="date-info">
        <span class="day-name">{{ dayName }}</span>
        <span class="day-number">{{ dayNumber }}</span>
        <span class="day-date">{{ fullDate }}</span>
      </div>
      <div class="day-stats" v-if="events.length > 0">
        <span class="stat-item">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16" height="16">
            <rect x="3" y="4" width="18" height="18" rx="2" />
            <line x1="16" y1="2" x2="16" y2="6" />
            <line x1="8" y1="2" x2="8" y2="6" />
            <line x1="3" y1="10" x2="21" y2="10" />
          </svg>
          {{ events.length }} 个日程
        </span>
      </div>
    </div>

    <div class="day-body">
      <div class="time-axis">
        <div v-for="hour in hours" :key="hour" class="time-slot">
          <span class="time-label">{{ formatHour(hour) }}</span>
        </div>
        <div v-if="isCurrentDay" class="current-time-line" :style="currentTimeStyle">
          <span class="current-time-dot"></span>
          <span class="current-time-label">{{ currentTimeLabel }}</span>
        </div>
      </div>

      <div class="day-content" ref="contentRef">
        <div
          v-for="hour in hours"
          :key="hour"
          class="hour-slot"
          @click="onSlotClick(hour)"
        ></div>

        <div
          v-for="item in eventLayouts"
          :key="item.event.id"
          class="event-block"
          :style="item.style"
          @click="onEventClick(item.event)"
          @mouseenter="showTooltip($event, item.event)"
          @mouseleave="hideTooltip"
        >
          <div class="event-content">
            <span class="event-time">{{ formatEventTime(item.event) }}</span>
            <span class="event-title">{{ item.event.title }}</span>
            <span v-if="getPomodoroText(item.event)" class="event-pomodoro">{{ getPomodoroText(item.event) }}</span>
            <span class="event-status" v-if="item.event.status !== 'planned'">
              {{ getStatusLabel(item.event.status) }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <Teleport to="body">
      <div v-if="tooltipVisible" class="event-tooltip" :style="tooltipStyle">
        <div class="tooltip-time">{{ tooltipData.time }}</div>
        <div class="tooltip-title">{{ tooltipData.title }}</div>
        <div v-if="tooltipData.status" class="tooltip-status">{{ tooltipData.status }}</div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
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

const contentRef = ref<HTMLElement | null>(null)
const currentTime = ref(new Date())
let timeUpdateInterval: ReturnType<typeof setInterval> | null = null

const hours = Array.from({ length: 24 }, (_, i) => i)

function parseLocalTime(isoString: string): { hours: number; minutes: number; dateStr: string } {
  const date = new Date(isoString)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return {
    hours: date.getHours(),
    minutes: date.getMinutes(),
    dateStr: `${year}-${month}-${day}`
  }
}

const currentDateString = computed(() => {
  const year = props.currentDate.getFullYear()
  const month = String(props.currentDate.getMonth() + 1).padStart(2, '0')
  const day = String(props.currentDate.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
})

const dayName = computed(() => {
  const days = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
  return days[props.currentDate.getDay()]
})

const dayNumber = computed(() => props.currentDate.getDate())

const fullDate = computed(() => {
  return `${props.currentDate.getFullYear()}年${props.currentDate.getMonth() + 1}月`
})

const isCurrentDay = computed(() => {
  const today = new Date()
  return (
    today.getFullYear() === props.currentDate.getFullYear() &&
    today.getMonth() === props.currentDate.getMonth() &&
    today.getDate() === props.currentDate.getDate()
  )
})

const currentTimeStyle = computed(() => {
  const hours = currentTime.value.getHours()
  const minutes = currentTime.value.getMinutes()
  const top = (hours + minutes / 60) * 60
  return { top: `${top}px` }
})

const currentTimeLabel = computed(() => {
  const hours = currentTime.value.getHours()
  const minutes = currentTime.value.getMinutes()
  return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}`
})

function formatHour(hour: number): string {
  return `${hour.toString().padStart(2, '0')}:00`
}

function formatEventTime(event: ScheduleEvent): string {
  const startLocal = parseLocalTime(event.start)
  const endLocal = parseLocalTime(event.end)
  const startTime = `${startLocal.hours.toString().padStart(2, '0')}:${startLocal.minutes.toString().padStart(2, '0')}`
  const endTime = `${endLocal.hours.toString().padStart(2, '0')}:${endLocal.minutes.toString().padStart(2, '0')}`
  return `${startTime} - ${endTime}`
}

function getStatusLabel(status: string): string {
  const labels: Record<string, string> = {
    in_progress: '进行中',
    completed: '已完成',
    cancelled: '已取消'
  }
  return labels[status] || ''
}

function getPomodoroText(event: ScheduleEvent): string {
  if (!event.task_id || !props.tasksMap) return ''
  const task = props.tasksMap[event.task_id]
  if (!task || task.planned_pomodoros === 0) return ''
  return `${task.completed_pomodoros}/${task.planned_pomodoros} 番茄钟`
}

function getEventColorByType(type: string): string {
  const colors: Record<string, string> = {
    task: '#B8452C',
    pomodoro: '#B8954D',
    break: '#6B8B6F',
    custom: '#9C9893'
  }
  return colors[type] || '#B8452C'
}

interface EventLayout {
  event: ScheduleEvent
  style: Record<string, string>
}

const eventLayouts = computed<EventLayout[]>(() => {
  if (props.events.length === 0) return []

  const sortedEvents = [...props.events].sort((a, b) => {
    const startA = new Date(a.start).getTime()
    const startB = new Date(b.start).getTime()
    return startA - startB
  })

  const eventTimes = sortedEvents.map(event => {
    const startLocal = parseLocalTime(event.start)
    const endLocal = parseLocalTime(event.end)
    return {
      event,
      startHour: startLocal.hours + startLocal.minutes / 60,
      endHour: endLocal.hours + endLocal.minutes / 60
    }
  })

  const groups: Array<Array<typeof eventTimes[0]>> = []
  let currentGroup: Array<typeof eventTimes[0]> = []

  for (const item of eventTimes) {
    if (currentGroup.length === 0) {
      currentGroup.push(item)
    } else {
      const hasOverlap = currentGroup.some(g => {
        return item.startHour < g.endHour && item.endHour > g.startHour
      })
      if (hasOverlap) {
        currentGroup.push(item)
      } else {
        groups.push(currentGroup)
        currentGroup = [item]
      }
    }
  }
  if (currentGroup.length > 0) {
    groups.push(currentGroup)
  }

  const layouts: EventLayout[] = []
  const margin = 8

  for (const group of groups) {
    const columns: Array<Array<typeof eventTimes[0]>> = []

    for (const item of group) {
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
    const gap = 4

    for (let colIndex = 0; colIndex < columns.length; colIndex++) {
      for (const item of columns[colIndex]) {
        const top = item.startHour * 60
        const height = Math.max((item.endHour - item.startHour) * 60, 30)
        const color = item.event.color || getEventColorByType(item.event.type)

        const leftCalc = `calc(${margin}px + ${colIndex} * ((100% - ${margin * 2}px - ${(totalColumns - 1) * gap}px) / ${totalColumns}) + ${colIndex * gap}px)`
        const widthCalc = `calc((100% - ${margin * 2}px - ${(totalColumns - 1) * gap}px) / ${totalColumns})`

        layouts.push({
          event: item.event,
          style: {
            top: `${top}px`,
            height: `${height}px`,
            left: leftCalc,
            width: widthCalc,
            backgroundColor: color
          }
        })
      }
    }
  }

  return layouts
})

function onSlotClick(hour: number) {
  emit('slot-click', currentDateString.value, hour)
}

function onEventClick(event: ScheduleEvent) {
  emit('event-click', event)
}

const tooltipVisible = ref(false)
const tooltipStyle = ref<Record<string, string>>({})
const tooltipData = ref({ time: '', title: '', status: '' })
let tooltipTimer: ReturnType<typeof setTimeout> | null = null

function showTooltip(event: MouseEvent, scheduleEvent: ScheduleEvent) {
  tooltipTimer = setTimeout(() => {
    const rect = (event.target as HTMLElement).getBoundingClientRect()
    tooltipData.value = {
      time: formatEventTime(scheduleEvent),
      title: scheduleEvent.title,
      status: scheduleEvent.status !== 'planned' ? getStatusLabel(scheduleEvent.status) : ''
    }
    tooltipStyle.value = {
      position: 'fixed',
      left: `${rect.left}px`,
      top: `${rect.top - 10}px`,
      transform: 'translateY(-100%)'
    }
    tooltipVisible.value = true
  }, 500)
}

function hideTooltip() {
  if (tooltipTimer) { clearTimeout(tooltipTimer); tooltipTimer = null }
  tooltipVisible.value = false
}

function updateTime() { currentTime.value = new Date() }

onMounted(() => {
  updateTime()
  timeUpdateInterval = setInterval(updateTime, 60000)
  if (isCurrentDay.value && contentRef.value) {
    const hours = currentTime.value.getHours()
    const scrollPosition = Math.max(0, hours * 60 - 100)
    contentRef.value.scrollTop = scrollPosition
  }
})

onUnmounted(() => {
  if (timeUpdateInterval) clearInterval(timeUpdateInterval)
  if (tooltipTimer) clearTimeout(tooltipTimer)
})
</script>

<style scoped>
.day-view {
  display: flex;
  flex-direction: column;
  background: var(--bg-card);
  border-radius: var(--radius-xl);
  overflow: hidden;
  border: 1px solid var(--border-color);
}

.day-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-bottom: 1px solid var(--border-color);
}

.date-info {
  display: flex;
  align-items: baseline;
  gap: 12px;
}

.day-name {
  font-size: 15px;
  color: var(--text-secondary);
  font-weight: 500;
}

.day-number {
  font-size: 32px;
  font-weight: 600;
  color: var(--text-primary);
  font-family: var(--font-display);
}

.day-date {
  font-size: 14px;
  color: var(--text-muted);
}

.day-stats {
  display: flex;
  gap: 16px;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--text-secondary);
}

.stat-item svg {
  color: var(--text-muted);
}

.day-body {
  display: flex;
  overflow-y: auto;
  max-height: 600px;
  position: relative;
}

.time-axis {
  width: 70px;
  flex-shrink: 0;
  border-right: 1px solid var(--border-color);
  position: relative;
}

.time-slot {
  height: 60px;
  display: flex;
  align-items: flex-start;
  justify-content: flex-end;
  padding-right: 12px;
  padding-top: 4px;
}

.time-label {
  font-size: 11px;
  color: var(--text-muted);
  font-weight: 500;
  font-family: var(--font-mono);
}

.current-time-line {
  position: absolute;
  left: 0;
  right: 0;
  height: 1px;
  background: var(--accent-primary);
  z-index: 10;
  display: flex;
  align-items: center;
}

.current-time-dot {
  width: 8px;
  height: 8px;
  background: var(--accent-primary);
  border-radius: 50%;
  position: absolute;
  right: -4px;
  top: -3.5px;
}

.current-time-label {
  position: absolute;
  right: 6px;
  top: -16px;
  font-size: 10px;
  color: var(--accent-primary);
  font-weight: 600;
  font-family: var(--font-mono);
}

.day-content {
  flex: 1;
  position: relative;
  min-height: 1440px;
}

.hour-slot {
  position: relative;
  height: 60px;
  border-bottom: 1px solid var(--border-color);
  cursor: pointer;
  transition: background var(--transition-fast);
  z-index: 1;
}

.hour-slot:hover {
  background: rgba(0, 0, 0, 0.02);
}

.event-block {
  position: absolute;
  border-radius: 6px;
  cursor: pointer;
  overflow: hidden;
  transition: opacity var(--transition-fast);
  display: flex;
  color: #fff;
  z-index: 2;
}

.event-block:hover {
  opacity: 0.9;
}

.event-content {
  flex: 1;
  padding: 6px 10px;
  min-width: 0;
}

.event-time {
  display: block;
  font-size: 11px;
  opacity: 0.85;
  margin-bottom: 2px;
}

.event-title {
  display: block;
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.event-status {
  display: inline-block;
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 3px;
  margin-top: 3px;
  background: rgba(255, 255, 255, 0.2);
}

.event-pomodoro {
  display: inline-block;
  font-size: 10px;
  padding: 1px 5px;
  border-radius: 3px;
  margin-top: 3px;
  margin-left: 4px;
  background: rgba(255, 255, 255, 0.2);
}
</style>

<style>
.event-tooltip {
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

.tooltip-time {
  font-size: 12px;
  color: var(--text-secondary);
  margin-bottom: 4px;
}

.tooltip-title {
  font-size: 14px;
  font-weight: 600;
  line-height: 1.4;
  word-break: break-word;
}

.tooltip-status {
  font-size: 11px;
  color: var(--text-secondary);
  margin-top: 6px;
  padding-top: 6px;
  border-top: 1px solid var(--border-color);
}
</style>
