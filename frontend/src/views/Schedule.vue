<template>
  <div class="schedule-page">
    <div class="page-header">
      <div class="header-left">
        <h1>日程</h1>
        <p class="page-subtitle">规划你的时间</p>
      </div>
      <div class="header-actions">
        <div class="view-switch">
          <button
            v-for="mode in viewModes"
            :key="mode.value"
            :class="['view-btn', { active: scheduleStore.viewMode === mode.value }]"
            @click="scheduleStore.setViewMode(mode.value)"
          >
            <component :is="mode.icon" :size="18" />
            <span>{{ mode.label }}</span>
          </button>
        </div>
        <el-button type="primary" size="large" @click="openCreateDialog" class="create-btn">
          <el-icon><Plus /></el-icon>
          <span>新建日程</span>
        </el-button>
      </div>
    </div>

    <div class="schedule-toolbar">
      <div class="navigation">
        <button class="nav-btn" @click="scheduleStore.goToPrevious()">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="15 18 9 12 15 6"/>
          </svg>
        </button>
        <span class="current-period">{{ currentPeriodLabel }}</span>
        <button class="nav-btn" @click="scheduleStore.goToNext()">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="9 18 15 12 9 6"/>
          </svg>
        </button>
        <button class="today-btn" @click="scheduleStore.goToToday()">今天</button>
      </div>
      <div class="toolbar-actions">
        <el-button @click="generateSchedule" :loading="scheduleStore.loading">
          <el-icon><MagicStick /></el-icon>
          <span>AI 生成</span>
        </el-button>
      </div>
    </div>

    <div class="schedule-content">
      <DayView
        v-if="scheduleStore.viewMode === 'day'"
        :current-date="scheduleStore.currentDate"
        :events="scheduleStore.events"
        @event-click="onEventClick"
        @slot-click="onSlotClick"
      />
      <WeekView
        v-if="scheduleStore.viewMode === 'week'"
        :current-date="scheduleStore.currentDate"
        :events="scheduleStore.events"
        @event-click="onEventClick"
        @slot-click="onSlotClick"
      />
      <MonthView
        v-if="scheduleStore.viewMode === 'month'"
        :current-date="scheduleStore.currentDate"
        :events="scheduleStore.events"
        @event-click="onEventClick"
        @day-click="onDayClick"
      />
    </div>

    <EventForm
      :visible="showForm"
      :event="editingEvent"
      :default-date="defaultDate"
      :default-hour="defaultHour"
      @close="closeForm"
      @save="createSchedule"
      @update="updateSchedule"
      @delete="deleteSchedule"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, h } from 'vue'
import { Plus, MagicStick } from '@element-plus/icons-vue'
import { useScheduleStore } from '@/stores/schedule'
import { ElMessage } from 'element-plus'
import WeekView from '@/components/schedule/WeekView.vue'
import DayView from '@/components/schedule/DayView.vue'
import MonthView from '@/components/schedule/MonthView.vue'
import EventForm from '@/components/schedule/EventForm.vue'
import type { ScheduleEvent, CreateScheduleDTO, UpdateScheduleDTO } from '@/types'

const scheduleStore = useScheduleStore()

const showForm = ref(false)
const editingEvent = ref<ScheduleEvent | null>(null)
const defaultDate = ref<string>('')
const defaultHour = ref<number>(9)

// 视图模式配置
const viewModes = [
  {
    label: '日',
    value: 'day' as const,
    icon: {
      render() {
        return h('svg', { viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': '2' }, [
          h('rect', { x: '3', y: '4', width: '18', height: '18', rx: '2' }),
          h('line', { x1: '16', y1: '2', x2: '16', y2: '6' }),
          h('line', { x1: '8', y1: '2', x2: '8', y2: '6' }),
          h('line', { x1: '3', y1: '10', x2: '21', y2: '10' })
        ])
      }
    }
  },
  {
    label: '周',
    value: 'week' as const,
    icon: {
      render() {
        return h('svg', { viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': '2' }, [
          h('rect', { x: '3', y: '3', width: '7', height: '7', rx: '1' }),
          h('rect', { x: '14', y: '3', width: '7', height: '7', rx: '1' }),
          h('rect', { x: '3', y: '14', width: '7', height: '7', rx: '1' }),
          h('rect', { x: '14', y: '14', width: '7', height: '7', rx: '1' })
        ])
      }
    }
  },
  {
    label: '月',
    value: 'month' as const,
    icon: {
      render() {
        return h('svg', { viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': '2' }, [
          h('rect', { x: '3', y: '4', width: '18', height: '18', rx: '2' }),
          h('line', { x1: '3', y1: '10', x2: '21', y2: '10' }),
          h('line', { x1: '9', y1: '4', x2: '9', y2: '22' }),
          h('line', { x1: '15', y1: '4', x2: '15', y2: '22' })
        ])
      }
    }
  }
]

const currentPeriodLabel = computed(() => {
  const date = scheduleStore.currentDate
  if (scheduleStore.viewMode === 'week') {
    const startOfWeek = new Date(date)
    const dayOfWeek = startOfWeek.getDay()
    startOfWeek.setDate(startOfWeek.getDate() - dayOfWeek + 1)
    const endOfWeek = new Date(startOfWeek)
    endOfWeek.setDate(startOfWeek.getDate() + 6)

    const formatDate = (d: Date) => `${d.getMonth() + 1}月${d.getDate()}日`
    return `${formatDate(startOfWeek)} - ${formatDate(endOfWeek)}`
  } else if (scheduleStore.viewMode === 'month') {
    return `${date.getFullYear()}年${date.getMonth() + 1}月`
  } else {
    return `${date.getFullYear()}年${date.getMonth() + 1}月${date.getDate()}日`
  }
})

function openCreateDialog() {
  editingEvent.value = null
  defaultDate.value = scheduleStore.currentDate.toISOString().split('T')[0]
  defaultHour.value = 9
  showForm.value = true
}

function onSlotClick(date: string, hour: number) {
  editingEvent.value = null
  defaultDate.value = date
  defaultHour.value = hour
  showForm.value = true
}

function onEventClick(event: ScheduleEvent) {
  editingEvent.value = event
  showForm.value = true
}

function onDayClick(dateStr: string) {
  // 切换到日视图并设置日期
  scheduleStore.setViewMode('day')
  scheduleStore.setCurrentDate(new Date(dateStr))
}

function closeForm() {
  showForm.value = false
  editingEvent.value = null
}

async function createSchedule(data: CreateScheduleDTO) {
  try {
    await scheduleStore.createSchedule(data)
    ElMessage.success('日程创建成功')
    closeForm()
  } catch (error) {
    console.error('Failed to create schedule:', error)
    ElMessage.error('创建失败，请检查网络连接')
  }
}

async function updateSchedule(id: string, data: UpdateScheduleDTO) {
  try {
    await scheduleStore.updateSchedule(id, data)
    ElMessage.success('日程更新成功')
    closeForm()
  } catch (error) {
    ElMessage.error('更新失败')
  }
}

async function deleteSchedule(id: string) {
  try {
    await scheduleStore.deleteSchedule(id)
    ElMessage.success('日程删除成功')
    closeForm()
  } catch (error) {
    ElMessage.error('删除失败')
  }
}

async function generateSchedule() {
  try {
    await scheduleStore.generateSchedule('09:00', '18:00')
    ElMessage.success('日程生成成功')
  } catch (error) {
    ElMessage.error('生成失败')
  }
}

function getDateRange() {
  const date = scheduleStore.currentDate
  if (scheduleStore.viewMode === 'week') {
    const startOfWeek = new Date(date)
    const dayOfWeek = startOfWeek.getDay()
    startOfWeek.setDate(startOfWeek.getDate() - dayOfWeek + 1)
    const endOfWeek = new Date(startOfWeek)
    endOfWeek.setDate(startOfWeek.getDate() + 6)

    return {
      start: startOfWeek.toISOString().split('T')[0],
      end: endOfWeek.toISOString().split('T')[0]
    }
  } else {
    return {
      start: date.toISOString().split('T')[0],
      end: date.toISOString().split('T')[0]
    }
  }
}

onMounted(async () => {
  await loadSchedules()
})

// 监听日期和视图模式变化，自动刷新数据
watch(
  () => [scheduleStore.currentDate, scheduleStore.viewMode],
  () => {
    loadSchedules()
  }
)

async function loadSchedules() {
  const range = getDateRange()
  await scheduleStore.fetchSchedules(range.start, range.end)
}
</script>

<style scoped>
.schedule-page {
  padding: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
  max-width: 1200px;
  margin: 0 auto;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 32px;
  padding-bottom: 24px;
  border-bottom: 1px solid var(--border-color);
  position: relative;
}

.header-left h1 {
  font-family: var(--font-display);
  font-size: 30px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 6px 0;
  letter-spacing: -0.5px;
}

.page-subtitle {
  font-size: 14px;
  color: var(--text-muted);
  margin: 0;
  font-weight: 400;
}

.header-actions {
  display: flex;
  gap: 16px;
  align-items: center;
}

.view-switch {
  display: flex;
  background: var(--bg-elevated);
  border-radius: 14px;
  padding: 6px;
  border: 1px solid var(--border-color);
}

.view-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 20px;
  border: none;
  background: transparent;
  border-radius: 10px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-secondary);
  transition: all var(--transition-normal);
}

.view-btn:hover {
  color: var(--text-primary);
}

.view-btn.active {
  background: var(--accent-primary);
  color: #fff;
}

.create-btn {
  height: 44px;
  padding: 0 24px;
  font-size: 14px;
  font-weight: 500;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  gap: 8px;
}

.schedule-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.navigation {
  display: flex;
  align-items: center;
  gap: 16px;
}

.nav-btn {
  width: 40px;
  height: 40px;
  border: 1px solid var(--border-color);
  background: var(--bg-card);
  border-radius: 12px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all var(--transition-normal);
}

.nav-btn:hover {
  background: rgba(0, 0, 0, 0.03);
  border-color: var(--border-accent);
}

.nav-btn svg {
  width: 20px;
  height: 20px;
  color: var(--text-secondary);
}

.current-period {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  min-width: 200px;
  text-align: center;
}

.today-btn {
  padding: 10px 20px;
  border: 1px solid var(--border-color);
  background: var(--bg-card);
  border-radius: 10px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-secondary);
  transition: all var(--transition-normal);
}

.today-btn:hover {
  background: rgba(0, 0, 0, 0.03);
  border-color: var(--border-accent);
  color: var(--text-primary);
}

.toolbar-actions .el-button {
  display: flex;
  align-items: center;
  gap: 8px;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  border-radius: var(--radius-md);
}

.toolbar-actions .el-button:hover {
  background: rgba(0, 0, 0, 0.03);
  border-color: var(--border-accent);
}

.schedule-content {
  flex: 1;
  overflow: auto;
}
</style>