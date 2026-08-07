<template>
  <el-dialog v-model="visible" :title="task?.title || '任务详情'" width="460px" @close="onClose">
    <div v-if="task" class="pomodoro-detail">
      <!-- 基本信息 -->
      <div class="detail-section">
        <div class="detail-meta">
          <span class="detail-quadrant" :style="{ color: quadrantColor }">{{ quadrantName }}</span>
          <span v-if="task.deadline" class="detail-deadline">截止 {{ formatDate(task.deadline) }}</span>
          <span class="detail-status" :class="'status-' + task.status">{{ statusLabel }}</span>
        </div>
        <p v-if="task.description" class="detail-desc">{{ task.description }}</p>
      </div>

      <!-- 番茄钟进度 -->
      <div class="detail-section">
        <h4>番茄钟进度</h4>
        <div class="progress-area">
          <div class="progress-bar-wrapper">
            <div class="progress-bar" :style="{ width: progressPercent + '%' }"></div>
          </div>
          <span class="progress-text">{{ task.completed_pomodoros }}/{{ task.planned_pomodoros }} {{ progressPercentText }}</span>
        </div>
        <p v-if="task.planned_pomodoros === 0" class="progress-hint">自由番茄钟模式（无预估时间）</p>
      </div>

      <!-- 今日历史 -->
      <div class="detail-section" v-if="todaySessions.length > 0">
        <h4>今日番茄记录</h4>
        <div class="session-history">
          <div v-for="session in todaySessions" :key="session.id" class="session-item">
            <span class="session-time">{{ formatSessionTime(session) }}</span>
            <span class="session-duration">{{ Math.round((session.actual_duration || session.planned_duration) / 60) }} 分钟</span>
          </div>
        </div>
      </div>

      <!-- 开始按钮 -->
      <div class="detail-section" v-if="task.status !== 'completed'">
        <button class="start-pomodoro-btn" @click="startNextPomodoro">
          开始第 {{ task.completed_pomodoros + 1 }} 个番茄钟
        </button>
      </div>

      <!-- 底部统计 -->
      <div class="detail-footer">
        <span>已专注 {{ focusedMinutes }} 分钟</span>
        <span v-if="remainingMinutes > 0"> · 剩余约 {{ remainingMinutes }} 分钟</span>
      </div>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { TaskResponse, PomodoroSession } from '@/types'
import { QUADRANT_INFO } from '@/types'
import { api } from '@/api/client'
import { useTimerStore } from '@/stores/timer'
import { ElMessage } from 'element-plus'

const timerStore = useTimerStore()

const visible = ref(false)
const task = ref<TaskResponse | null>(null)
const todaySessions = ref<PomodoroSession[]>([])

const quadrantName = computed(() => {
  if (!task.value) return ''
  return QUADRANT_INFO[task.value.quadrant as 1 | 2 | 3 | 4]?.name || ''
})

const quadrantColor = computed(() => {
  if (!task.value) return ''
  return QUADRANT_INFO[task.value.quadrant as 1 | 2 | 3 | 4]?.color || ''
})

const statusLabel = computed(() => {
  const labels: Record<string, string> = { todo: '待办', in_progress: '进行中', completed: '已完成', cancelled: '已取消' }
  return task.value ? labels[task.value.status] || '' : ''
})

const progressPercent = computed(() => {
  if (!task.value || task.value.planned_pomodoros === 0) return 0
  return Math.min(100, Math.round((task.value.completed_pomodoros / task.value.planned_pomodoros) * 100))
})

const progressPercentText = computed(() => {
  if (!task.value || task.value.planned_pomodoros === 0) return ''
  return `(${progressPercent.value}%)`
})

const focusedMinutes = computed(() => {
  const totalSeconds = todaySessions.value.reduce((sum, s) => {
    return sum + (s.actual_duration || s.planned_duration || 0)
  }, 0)
  return Math.round(totalSeconds / 60)
})

const remainingMinutes = computed(() => {
  if (!task.value || task.value.planned_pomodoros === 0) return 0
  const remaining = task.value.planned_pomodoros - task.value.completed_pomodoros
  if (remaining <= 0) return 0
  // Estimate ~25 minutes per remaining pomodoro (use settings if available)
  return remaining * 25
})

function formatDate(d: string): string {
  const dt = new Date(d)
  return `${dt.getMonth() + 1}/${dt.getDate()}`
}

function formatSessionTime(session: PomodoroSession): string {
  const start = new Date(session.start_time)
  const end = session.end_time ? new Date(session.end_time) : null
  const startStr = `${String(start.getHours()).padStart(2, '0')}:${String(start.getMinutes()).padStart(2, '0')}`
  if (end) {
    const endStr = `${String(end.getHours()).padStart(2, '0')}:${String(end.getMinutes()).padStart(2, '0')}`
    return `${startStr} - ${endStr}`
  }
  return startStr
}

async function open(t: TaskResponse) {
  task.value = t
  visible.value = true
  await fetchTodaySessions(t.id)
}

async function fetchTodaySessions(taskId: string) {
  try {
    const res = await api.getRecentSessions(20)
    const allSessions = res.data
    const today = new Date().toISOString().split('T')[0]
    todaySessions.value = allSessions.filter(s =>
      s.task_id === taskId &&
      s.type === 'work' &&
      s.status === 'completed' &&
      s.start_time.startsWith(today)
    )
  } catch (error) {
    console.error('Failed to fetch sessions:', error)
  }
}

async function startNextPomodoro() {
  if (!task.value) return
  try {
    await timerStore.createSession(task.value.id, 'work')
    visible.value = false
    ElMessage.success(`开始专注：${task.value.title}`)
  } catch {
    ElMessage.error('启动番茄钟失败')
  }
}

function onClose() {
  task.value = null
  todaySessions.value = []
}

defineExpose({ open })
</script>

<style scoped>
.pomodoro-detail {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.detail-section h4 {
  font-family: var(--font-display);
  font-variation-settings: 'opsz' 40;
  font-size: 15px;
  font-weight: 440;
  color: var(--text-primary);
  margin: 0 0 10px 0;
  letter-spacing: -0.02em;
}

.detail-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 8px;
  flex-wrap: wrap;
}

.detail-quadrant {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 9px;
  border-radius: 999px;
  background: rgba(239, 231, 215, 0.05);
  font-family: var(--font-mono);
  letter-spacing: 0.04em;
}

.detail-deadline {
  font-size: 12px;
  color: var(--text-secondary);
  font-family: var(--font-mono);
}

.detail-status {
  font-size: 11px;
  padding: 2px 9px;
  border-radius: 999px;
  background: rgba(239, 231, 215, 0.05);
  color: var(--text-secondary);
  font-family: var(--font-mono);
}

.detail-status.status-completed {
  color: var(--accent-sage);
  background: var(--sage-fill);
}

.detail-desc {
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.6;
  margin: 0;
}

.progress-area {
  display: flex;
  align-items: center;
  gap: 12px;
}

.progress-bar-wrapper {
  flex: 1;
  height: 7px;
  background: rgba(239, 231, 215, 0.08);
  border-radius: 999px;
  overflow: hidden;
}

.progress-bar {
  height: 100%;
  background: var(--gradient-primary);
  border-radius: 999px;
  transition: width 0.3s ease;
}

.progress-text {
  font-size: 12.5px;
  font-weight: 500;
  color: var(--text-primary);
  white-space: nowrap;
  font-family: var(--font-mono);
}

.progress-hint {
  font-size: 12px;
  color: var(--text-muted);
  margin: 6px 0 0;
}

.session-history {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: 160px;
  overflow-y: auto;
}

.session-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 7px 11px;
  background: rgba(239, 231, 215, 0.03);
  border-radius: var(--radius-sm);
  font-size: 13px;
}

.session-time {
  color: var(--text-primary);
  font-family: var(--font-mono);
}

.session-duration {
  color: var(--text-secondary);
  font-size: 12px;
  font-family: var(--font-mono);
}

.start-pomodoro-btn {
  width: 100%;
  padding: 12px;
  background: var(--gradient-primary);
  color: var(--bg-primary);
  border: none;
  border-radius: var(--radius-md);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all var(--transition-fast);
  font-family: var(--font-body);
}

.start-pomodoro-btn:hover {
  box-shadow: 0 8px 24px rgba(230, 162, 60, 0.28);
}

.detail-footer {
  padding-top: 12px;
  border-top: 1px solid var(--border-color);
  font-size: 13px;
  color: var(--text-secondary);
  font-family: var(--font-mono);
}
</style>
