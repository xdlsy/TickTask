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
        <el-button @click="resetSchedule" :disabled="scheduleStore.loading">
          <el-icon><Delete /></el-icon>
          <span>重置日程</span>
        </el-button>
        <el-button @click="generateSchedule" :loading="scheduleStore.loading">
          <el-icon><MagicStick /></el-icon>
          <span>生成日程</span>
        </el-button>
        <el-button
          @click="showReviseInput = true"
          :disabled="scheduleStore.loading || scheduleStore.events.length === 0"
          :title="scheduleStore.events.length === 0 ? '请先生成日程' : ''"
        >
          <el-icon><Edit /></el-icon>
          <span>修订日程</span>
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

    <div v-if="aiReasoning" class="ai-reasoning-bar">
      <div class="reasoning-header">
        <svg class="reasoning-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" width="18" height="18">
          <path d="M9.937 15.5A2 2 0 0 0 8.5 14.063l-1.135-5.675a.55.55 0 0 1 .652-.625L14 9l1.05-4.725a.55.55 0 0 1 1.045.152L14.063 15.5A2 2 0 0 1 12 17H4a1 1 0 0 1 0-2h5.937Z"/>
          <path d="M20 3v4"/>
          <path d="M22 5h-4"/>
          <path d="M4 17v4"/>
          <path d="M6 19H2"/>
        </svg>
        <span>排程总结</span>
      </div>
      <p class="reasoning-text">{{ aiReasoning }}</p>
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

    <TerminalOverlay
      :visible="scheduleStore.aiGenerating"
      :lines="scheduleStore.terminalOutput"
      :status="scheduleStore.terminalStatus"
      :status-message="scheduleStore.terminalStatusMessage"
      :status-detail="scheduleStore.terminalStatusDetail"
      :reasoning="scheduleStore.aiReasoning"
      :tool-name="scheduleStore.cliToolName"
      @close="scheduleStore.aiGenerating = false"
    />

    <!-- 修订指令输入对话框 -->
    <el-dialog v-model="showReviseInput" title="修订日程" width="520px" :close-on-click-modal="false">
      <div class="revise-input-body">
        <p class="revise-range-hint">修订范围：{{ currentPeriodLabel }}</p>
        <el-input
          v-model="revisePrompt"
          type="textarea"
          :rows="5"
          placeholder="描述你想如何调整日程，例如：把代码评审移到下午、优化上午的深度工作安排、为紧急任务腾出 2 小时……"
        />
      </div>
      <template #footer>
        <el-button @click="showReviseInput = false">取消</el-button>
        <el-button type="primary" @click="startRevise" :disabled="!revisePrompt.trim()">开始修订</el-button>
      </template>
    </el-dialog>

    <!-- 修订预览对话框 -->
    <el-dialog v-model="showRevisePreview" title="修订预览" width="560px" :close-on-click-modal="false">
      <div class="revise-preview-body">
        <div v-if="scheduleStore.revisionChanges.length > 0" class="revise-summary">
          ✨ {{ scheduleStore.revisionSummary }}
        </div>
        <div v-else class="revise-empty">
          当前日程已是最优安排，无需调整
        </div>
        <div v-if="scheduleStore.revisionChanges.length > 0" class="revise-changes">
          <div v-for="(change, index) in scheduleStore.revisionChanges" :key="index" class="change-item">
            <el-tag
              :type="change.type === 'moved' ? 'warning' : change.type === 'added' ? 'success' : 'info'"
              size="small"
            >
              {{ change.type === 'moved' ? '移动' : change.type === 'added' ? '新增' : '移除' }}
            </el-tag>
            <span class="change-title">{{ change.title }}</span>
            <span class="change-time">
              <template v-if="change.type === 'moved'">
                {{ formatRevisionTime(change.original_start) }} — {{ formatRevisionTime(change.original_end) }}
                → {{ formatRevisionTime(change.new_start) }} — {{ formatRevisionTime(change.new_end) }}
              </template>
              <template v-else-if="change.type === 'added'">
                {{ formatRevisionTime(change.new_start) }} — {{ formatRevisionTime(change.new_end) }}
              </template>
              <template v-else>
                {{ formatRevisionTime(change.original_start) }} — {{ formatRevisionTime(change.original_end) }}（将被移除）
              </template>
            </span>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="cancelRevision">取消</el-button>
        <el-button
          v-if="scheduleStore.revisionChanges.length > 0"
          type="primary"
          @click="confirmRevision"
        >
          确认应用
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, h } from 'vue'
import { Plus, MagicStick, Delete, Edit } from '@element-plus/icons-vue'
import TerminalOverlay from '@/components/schedule/TerminalOverlay.vue'
import { useScheduleStore } from '@/stores/schedule'
import { ElMessage, ElMessageBox } from 'element-plus'
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

// 修订日程状态
const showReviseInput = ref(false)
const showRevisePreview = ref(false)
const revisePrompt = ref('')
const aiReasoning = ref('')

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
    // getDay() 周日返回 0，转换为周一=0 ... 周日=6
    const mondayOffset = dayOfWeek === 0 ? -6 : 1 - dayOfWeek
    startOfWeek.setDate(startOfWeek.getDate() + mondayOffset)
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
    aiReasoning.value = scheduleStore.aiReasoning
    await loadSchedules()
    ElMessage.success('日程生成成功')
  } catch (error: any) {
    const msg = error?.response?.data?.error || '日程生成失败，请重试'
    ElMessage.error(msg)
  }
}

function formatRevisionTime(isoStr?: string) {
  if (!isoStr) return ''
  const d = new Date(isoStr)
  const weekdays = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
  const month = d.getMonth() + 1
  const day = d.getDate()
  const weekday = weekdays[d.getDay()]
  const hours = String(d.getHours()).padStart(2, '0')
  const minutes = String(d.getMinutes()).padStart(2, '0')
  return `${month}/${day} ${weekday} ${hours}:${minutes}`
}

async function startRevise() {
  if (!revisePrompt.value.trim()) return
  showReviseInput.value = false
  try {
    await scheduleStore.reviseSchedule(revisePrompt.value.trim())
    revisePrompt.value = ''
    showRevisePreview.value = true
  } catch (error: any) {
    const msg = error?.response?.data?.error || '日程修订失败，请重试'
    ElMessage.error(msg)
  }
}

async function confirmRevision() {
  showRevisePreview.value = false
  try {
    await scheduleStore.applyRevision()
    await loadSchedules()
    ElMessage.success('日程修订成功')
  } catch (error: any) {
    const msg = error?.response?.data?.error || '应用修订失败，请重试'
    ElMessage.error(msg)
  }
}

function cancelRevision() {
  showRevisePreview.value = false
  scheduleStore.revisionChanges = []
  scheduleStore.revisionSummary = ''
}

async function resetSchedule() {
  try {
    await ElMessageBox.confirm('确定要清空所有日程吗？此操作不可恢复。', '确认重置', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    const deleted = await scheduleStore.resetSchedules()
    ElMessage.success(`已清空 ${deleted} 条日程`)
  } catch (error: any) {
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error('重置失败，请重试')
    }
  }
}

function getDateRange() {
  const date = scheduleStore.currentDate
  if (scheduleStore.viewMode === 'week') {
    const startOfWeek = new Date(date)
    const dayOfWeek = startOfWeek.getDay()
    // getDay() 周日返回 0，转换为周一=0 ... 周日=6
    const mondayOffset = dayOfWeek === 0 ? -6 : 1 - dayOfWeek
    startOfWeek.setDate(startOfWeek.getDate() + mondayOffset)
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

.toolbar-actions {
  display: flex;
  align-items: center;
  gap: 8px;
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

/* 修订日程对话框 */
.revise-input-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.revise-range-hint {
  font-size: 13px;
  color: var(--text-muted);
  margin: 0;
}

.revise-preview-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.revise-summary {
  font-size: 15px;
  font-weight: 500;
  color: var(--text-primary);
  padding: 10px 14px;
  background: rgba(107, 139, 111, 0.08);
  border-radius: var(--radius-md);
}

.revise-empty {
  text-align: center;
  color: var(--text-muted);
  padding: 24px 0;
  font-size: 14px;
}

.revise-changes {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.change-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  background: var(--bg-secondary);
  border-radius: var(--radius-md);
  font-size: 13px;
}

.change-title {
  font-weight: 600;
  color: var(--text-primary);
  min-width: 80px;
}

.change-time {
  color: var(--text-secondary);
  flex: 1;
}

/* ── AI 排程总结 ── */
.ai-reasoning-bar {
  margin-top: 28px;
  padding: 22px 26px;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  border-left: 3px solid var(--accent-sage);
  animation: reasoningSlideIn 0.4s ease-out;
}

@keyframes reasoningSlideIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.reasoning-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.reasoning-icon {
  color: var(--accent-sage);
  flex-shrink: 0;
}

.reasoning-header span {
  font-family: var(--font-display);
  font-size: 16px;
  font-weight: 600;
  color: var(--accent-sage);
  letter-spacing: -0.2px;
}

.reasoning-text {
  font-family: var(--font-body);
  font-size: 14px;
  line-height: 1.75;
  color: var(--text-secondary);
  margin: 0;
  white-space: pre-wrap;
}
</style>