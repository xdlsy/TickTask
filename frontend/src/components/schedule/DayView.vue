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
        <!-- 当前时间线 -->
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

        <!-- 日程事件 -->
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
            <span class="event-status" v-if="item.event.status !== 'planned'">
              {{ getStatusLabel(item.event.status) }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- Tooltip -->
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
import type { ScheduleEvent } from '@/types'

const props = defineProps<{
  currentDate: Date
  events: ScheduleEvent[]
}>()

const emit = defineEmits<{
  'event-click': [event: ScheduleEvent]
  'slot-click': [date: string, hour: number]
}>()

const contentRef = ref<HTMLElement | null>(null)
const currentTime = ref(new Date())
let timeUpdateInterval: ReturnType<typeof setInterval> | null = null

const hours = Array.from({ length: 24 }, (_, i) => i)

// 解析 ISO 时间字符串为本地时间组件
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

// 当前日期信息
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

function getEventColorByType(type: string): string {
  const colors: Record<string, string> = {
    task: '#3b82f6',
    pomodoro: '#f59e0b',
    break: '#22c55e',
    custom: '#6b7280'
  }
  return colors[type] || '#3b82f6'
}

// 计算事件的布局位置，处理重叠事件
interface EventLayout {
  event: ScheduleEvent
  style: Record<string, string>
}

const eventLayouts = computed<EventLayout[]>(() => {
  if (props.events.length === 0) return []

  // 将事件按开始时间排序
  const sortedEvents = [...props.events].sort((a, b) => {
    const startA = new Date(a.start).getTime()
    const startB = new Date(b.start).getTime()
    return startA - startB
  })

  // 计算每个事件的时间范围（以小时为单位）
  const eventTimes = sortedEvents.map(event => {
    const startLocal = parseLocalTime(event.start)
    const endLocal = parseLocalTime(event.end)
    return {
      event,
      startHour: startLocal.hours + startLocal.minutes / 60,
      endHour: endLocal.hours + endLocal.minutes / 60
    }
  })

  // 分组：找出重叠的事件组
  const groups: Array<Array<typeof eventTimes[0]>> = []
  let currentGroup: Array<typeof eventTimes[0]> = []

  for (const item of eventTimes) {
    if (currentGroup.length === 0) {
      currentGroup.push(item)
    } else {
      // 检查是否与当前组中的任何事件重叠
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

  // 为每个组计算布局
  const layouts: EventLayout[] = []
  const margin = 8 // 左右内边距

  for (const group of groups) {
    // 使用列分配算法
    const columns: Array<Array<typeof eventTimes[0]>> = []

    for (const item of group) {
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
    const gap = 4 // 列之间的间距（像素）

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
  }

  return layouts
})

function onSlotClick(hour: number) {
  emit('slot-click', currentDateString.value, hour)
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

function showTooltip(event: MouseEvent, scheduleEvent: ScheduleEvent) {
  // 延迟 500ms 显示 tooltip
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
  if (tooltipTimer) {
    clearTimeout(tooltipTimer)
    tooltipTimer = null
  }
  tooltipVisible.value = false
}

function updateTime() {
  currentTime.value = new Date()
}

onMounted(() => {
  updateTime()
  timeUpdateInterval = setInterval(updateTime, 60000) // 每分钟更新

  // 滚动到当前时间位置
  if (isCurrentDay.value && contentRef.value) {
    const hours = currentTime.value.getHours()
    const scrollPosition = Math.max(0, hours * 60 - 100)
    contentRef.value.scrollTop = scrollPosition
  }
})

onUnmounted(() => {
  if (timeUpdateInterval) {
    clearInterval(timeUpdateInterval)
  }
  if (tooltipTimer) {
    clearTimeout(tooltipTimer)
  }
})
</script>

<style scoped>
.day-view {
  display: flex;
  flex-direction: column;
  background: #fff;
  border-radius: 16px;
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.day-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-bottom: 1px solid #e5e7eb;
  background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
}

.date-info {
  display: flex;
  align-items: baseline;
  gap: 12px;
}

.day-name {
  font-size: 16px;
  color: #6b7280;
  font-weight: 500;
}

.day-number {
  font-size: 36px;
  font-weight: 700;
  color: #1f2937;
}

.day-date {
  font-size: 15px;
  color: #9ca3af;
}

.day-stats {
  display: flex;
  gap: 16px;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  color: #6b7280;
}

.stat-item svg {
  color: #9ca3af;
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
  border-right: 1px solid #e5e7eb;
  background: #f9fafb;
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
  font-size: 12px;
  color: #9ca3af;
  font-weight: 500;
}

.current-time-line {
  position: absolute;
  left: 0;
  right: 0;
  height: 2px;
  background: #ef4444;
  z-index: 10;
  display: flex;
  align-items: center;
}

.current-time-dot {
  width: 10px;
  height: 10px;
  background: #ef4444;
  border-radius: 50%;
  position: absolute;
  right: -5px;
  top: -4px;
}

.current-time-label {
  position: absolute;
  right: 8px;
  top: -18px;
  font-size: 11px;
  color: #ef4444;
  font-weight: 600;
  background: #fff;
  padding: 2px 6px;
  border-radius: 4px;
}

.day-content {
  flex: 1;
  position: relative;
  min-height: 1440px; /* 24 hours × 60px */
}

.hour-slot {
  position: relative;
  height: 60px;
  border-bottom: 1px solid #f3f4f6;
  cursor: pointer;
  transition: background 0.2s;
  z-index: 1;
}

.hour-slot:hover {
  background: #f0f9ff;
}

.event-block {
  position: absolute;
  border-radius: 8px;
  cursor: pointer;
  overflow: hidden;
  transition: all 0.2s;
  display: flex;
  color: #fff;
  z-index: 2;
}

.event-block:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
  transform: translateY(-1px);
}

.event-content {
  flex: 1;
  padding: 8px 12px;
  min-width: 0;
}

.event-time {
  display: block;
  font-size: 11px;
  opacity: 0.9;
  margin-bottom: 4px;
}

.event-title {
  display: block;
  font-size: 14px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.event-status {
  display: inline-block;
  font-size: 10px;
  padding: 2px 6px;
  border-radius: 4px;
  margin-top: 4px;
  background: rgba(255, 255, 255, 0.2);
}
</style>

<style>
.event-tooltip {
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

.tooltip-time {
  font-size: 12px;
  color: #94a3b8;
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
  color: #94a3b8;
  margin-top: 6px;
  padding-top: 6px;
  border-top: 1px solid #334155;
}
</style>