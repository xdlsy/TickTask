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
            <!-- 日程事件 -->
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
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Tooltip -->
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
import type { ScheduleEvent } from '@/types'

const props = defineProps<{
  currentDate: Date
  events: ScheduleEvent[]
}>()

const emit = defineEmits<{
  'event-click': [event: ScheduleEvent]
  'slot-click': [date: string, hour: number]
}>()

const hours = Array.from({ length: 24 }, (_, i) => i)

// 周视图从周一开始，而非 JavaScript 默认的周日
// 这符合中国及多数国家的日历习惯
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

// 解析 ISO 时间字符串为本地时间组件
function parseLocalTime(isoString: string): { hours: number; minutes: number; dateStr: string } {
  const date = new Date(isoString)
  // 获取本地时间的各个组件
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hours = date.getHours()
  const minutes = date.getMinutes()

  return {
    hours,
    minutes,
    dateStr: `${year}-${month}-${day}`
  }
}

function getEventsForDay(dateStr: string): ScheduleEvent[] {
  return props.events.filter(event => {
    const startLocal = parseLocalTime(event.start)
    const endLocal = parseLocalTime(event.end)
    // 事件覆盖该日期（包括跨日事件），使用本地日期比较
    return dateStr >= startLocal.dateStr && dateStr <= endLocal.dateStr
  })
}

// 计算事件的布局位置，处理重叠事件
interface EventLayout {
  event: ScheduleEvent
  style: Record<string, string>
}

function getEventLayoutsForDay(dateStr: string): EventLayout[] {
  const dayEvents = getEventsForDay(dateStr)
  if (dayEvents.length === 0) return []

  // 将事件按开始时间排序
  const sortedEvents = [...dayEvents].sort((a, b) => {
    const startA = new Date(a.start).getTime()
    const startB = new Date(b.start).getTime()
    return startA - startB
  })

  // 计算每个事件的时间范围（以小时为单位）
  const eventTimes = sortedEvents.map(event => {
    const startLocal = parseLocalTime(event.start)
    const endLocal = parseLocalTime(event.end)

    // 如果是开始日期，使用实际的开始时间；否则从 00:00 开始
    let startHour: number
    if (dateStr === startLocal.dateStr) {
      startHour = startLocal.hours + startLocal.minutes / 60
    } else {
      startHour = 0
    }

    // 如果是结束日期，使用实际的结束时间；否则到 24:00
    let endHour: number
    if (dateStr === endLocal.dateStr) {
      endHour = endLocal.hours + endLocal.minutes / 60
    } else {
      endHour = 24
    }

    return { event, startHour, endHour }
  })

  // 使用列分配算法处理重叠
  const columns: Array<Array<typeof eventTimes[0]>> = []

  for (const item of eventTimes) {
    // 找到第一个可以放置的列
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
  const gap = 2 // 列之间的间距（像素）
  const margin = 4 // 左右内边距

  const layouts: EventLayout[] = []

  for (let colIndex = 0; colIndex < columns.length; colIndex++) {
    for (const item of columns[colIndex]) {
      const top = item.startHour * 60
      const height = Math.max((item.endHour - item.startHour) * 60, 30)
      const color = item.event.color || getEventColorByType(item.event.type)

      // 使用 calc 计算位置和宽度
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

  return layouts
}

function getEventColorByType(type: string): string {
  const colors: Record<string, string> = {
    task: '#3b82f6',
    pomodoro: '#f59e0b',
    break: '#22c55e',
    custom: '#6b7280'
  }
  return colors[type] || '#3b82f6'
}

function onSlotClick(date: string, hour: number) {
  emit('slot-click', date, hour)
}

function onEventClick(event: ScheduleEvent) {
  emit('event-click', event)
}

// Tooltip 相关
const tooltipVisible = ref(false)
const tooltipStyle = ref<Record<string, string>>({})
const tooltipData = ref({
  time: '',
  title: '',
  status: ''
})
let tooltipTimer: ReturnType<typeof setTimeout> | null = null

function showTooltip(event: MouseEvent, scheduleEvent: ScheduleEvent, dayDate: string) {
  // 延迟 500ms 显示 tooltip
  tooltipTimer = setTimeout(() => {
    const rect = (event.target as HTMLElement).getBoundingClientRect()
    const timeStr = formatEventTime(scheduleEvent, dayDate)
    tooltipData.value = {
      time: timeStr || formatTimeRange(scheduleEvent),
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
  if (tooltipTimer) {
    clearTimeout(tooltipTimer)
    tooltipTimer = null
  }
  tooltipVisible.value = false
}

function formatTimeRange(event: ScheduleEvent): string {
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

onUnmounted(() => {
  if (tooltipTimer) {
    clearTimeout(tooltipTimer)
  }
})
</script>

<style scoped>
.week-view {
  display: flex;
  flex-direction: column;
  background: #fff;
  border-radius: 16px;
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.week-header {
  display: flex;
  border-bottom: 1px solid #e5e7eb;
}

.time-axis-header {
  width: 60px;
  flex-shrink: 0;
  background: #f9fafb;
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
  border-right: 1px solid #e5e7eb;
  background: #f9fafb;
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
  color: #9ca3af;
}

.days-container {
  display: flex;
  flex: 1;
}

.day-column {
  flex: 1;
  min-width: 120px;
  border-right: 1px solid #f3f4f6;
}

.day-column:last-child {
  border-right: none;
}

.day-header {
  padding: 12px 8px;
  text-align: center;
  background: #f9fafb;
  flex: 1;
}

.day-column.is-today .day-header {
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
  color: #fff;
}

.day-name {
  display: block;
  font-size: 12px;
  color: #6b7280;
  margin-bottom: 4px;
}

.is-today .day-name {
  color: rgba(255, 255, 255, 0.8);
}

.day-number {
  display: block;
  font-size: 20px;
  font-weight: 600;
  color: #1f2937;
}

.is-today .day-number {
  color: #fff;
}

.day-content {
  position: relative;
  /* 高度 = 24小时 × 60px/小时，与 getEventStyle 中的定位计算保持一致 */
  height: 1440px;
}

.hour-slot {
  height: 60px;
  border-bottom: 1px solid #f3f4f6;
  cursor: pointer;
  transition: background 0.2s;
}

.hour-slot:hover {
  background: #f0f9ff;
}

.event-block {
  position: absolute;
  border-radius: 6px;
  padding: 4px 8px;
  cursor: pointer;
  overflow: hidden;
  transition: all 0.2s;
  color: #fff;
  font-size: 12px;
}

.event-block:hover {
  transform: scale(1.02);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  z-index: 10;
}

.event-time {
  font-size: 10px;
  opacity: 0.9;
  display: block;
  margin-bottom: 2px;
}

.event-title {
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>

<style>
.week-tooltip.event-tooltip {
  background: #1e293b;
  color: #fff;
  padding: 10px 14px;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.25);
  z-index: 9999;
  pointer-events: none;
  min-width: 150px;
  max-width: 280px;
  animation: tooltipFadeIn 0.15s ease-out;
}

@keyframes tooltipFadeIn {
  from {
    opacity: 0;
    transform: translateY(-100%) translateY(5px);
  }
  to {
    opacity: 1;
    transform: translateY(-100%) translateY(0);
  }
}

.week-tooltip .tooltip-time {
  font-size: 12px;
  color: #94a3b8;
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
  color: #94a3b8;
  margin-top: 6px;
  padding-top: 6px;
  border-top: 1px solid #334155;
}
</style>