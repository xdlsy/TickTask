<template>
  <div class="dashboard">
    <div class="page-header">
      <h1>仪表盘</h1>
      <p class="page-subtitle">欢迎回来，今天也要高效工作</p>
    </div>

    <div class="stats-cards">
      <div class="stat-card">
        <div class="stat-icon pomodoro">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <path d="M12 6v6l4 2"/>
          </svg>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ todayPomodoros }}</div>
          <div class="stat-label">今日番茄</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon focus">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83"/>
          </svg>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ formatDuration(focusTime) }}</div>
          <div class="stat-label">专注时长</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon completed">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
            <polyline points="22 4 12 14.01 9 11.01"/>
          </svg>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ completedTasks }}</div>
          <div class="stat-label">完成任务</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon pending">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="4" width="18" height="18" rx="2" ry="2"/>
            <line x1="16" y1="2" x2="16" y2="6"/>
            <line x1="8" y1="2" x2="8" y2="6"/>
            <line x1="3" y1="10" x2="21" y2="10"/>
          </svg>
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ pendingTasks }}</div>
          <div class="stat-label">待办任务</div>
        </div>
      </div>
    </div>

    <div class="dashboard-content">
      <div class="main-column">
        <div class="card recent-tasks-card">
          <div class="card-header">
            <h3>最近任务</h3>
            <el-button text type="primary" @click="$router.push('/tasks')">
              查看全部
              <el-icon class="el-icon--right"><ArrowRight /></el-icon>
            </el-button>
          </div>
          <div v-if="recentTasks.length === 0" class="empty-state">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="empty-icon">
              <rect x="3" y="3" width="18" height="18" rx="2"/>
              <path d="M9 12l2 2 4-4"/>
            </svg>
            <p>暂无任务</p>
            <el-button type="primary" @click="$emit('add-task')">创建第一个任务</el-button>
          </div>
          <div v-else class="task-list">
            <TaskCard
              v-for="task in recentTasks"
              :key="task.id"
              :task="task"
              @drag-start="() => {}"
              @edit="$emit('edit-task', task)"
              @complete="$emit('complete-task', $event)"
              @delete="$emit('delete-task', $event)"
            />
          </div>
        </div>

        <!-- AI 日程生成 -->
        <div class="card ai-section" v-if="aiStore.configured">
          <div class="card-header">
            <h3>
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="ai-icon">
                <path d="M12 2a2 2 0 0 1 2 2c0 .74-.4 1.39-1 1.73V7h1a7 7 0 0 1 7 7h1a1 1 0 0 1 1 1v3a1 1 0 0 1-1 1h-1v1a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-1H2a1 1 0 0 1-1-1v-3a1 1 0 0 1 1-1h1a7 7 0 0 1 7-7h1V5.73c-.6-.34-1-.99-1-1.73a2 2 0 0 1 2-2z"/>
                <circle cx="8" cy="14" r="1.5"/>
                <circle cx="16" cy="14" r="1.5"/>
              </svg>
              AI 日程助手
            </h3>
          </div>
          <div class="schedule-form">
            <div class="time-inputs">
              <el-time-select
                v-model="scheduleStartTime"
                placeholder="开始时间"
                start="06:00"
                step="00:30"
                end="22:00"
              />
              <span class="time-separator">至</span>
              <el-time-select
                v-model="scheduleEndTime"
                placeholder="结束时间"
                start="06:00"
                step="00:30"
                end="23:00"
              />
            </div>
            <el-button
              type="primary"
              :loading="aiStore.loading"
              @click="generateSchedule"
            >
              <el-icon class="el-icon--left"><MagicStick /></el-icon>
              生成今日日程
            </el-button>
          </div>

          <div v-if="generatedSchedule" class="schedule-result">
            <div
              v-for="item in generatedSchedule"
              :key="item.id"
              class="schedule-item"
            >
              <div class="schedule-time">
                {{ formatScheduleTime(item.start) }} - {{ formatScheduleTime(item.end) }}
              </div>
              <div class="schedule-task">{{ item.title }}</div>
            </div>
          </div>
        </div>

        <!-- AI 未配置提示 -->
        <div class="card ai-section ai-not-configured" v-else>
          <div class="card-header">
            <h3>
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="ai-icon">
                <path d="M12 2a2 2 0 0 1 2 2c0 .74-.4 1.39-1 1.73V7h1a7 7 0 0 1 7 7h1a1 1 0 0 1 1 1v3a1 1 0 0 1-1 1h-1v1a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-1H2a1 1 0 0 1-1-1v-3a1 1 0 0 1 1-1h1a7 7 0 0 1 7-7h1V5.73c-.6-.34-1-.99-1-1.73a2 2 0 0 1 2-2z"/>
                <circle cx="8" cy="14" r="1.5"/>
                <circle cx="16" cy="14" r="1.5"/>
              </svg>
              AI 日程助手
            </h3>
          </div>
          <div class="ai-placeholder">
            <p>配置 AI API Key 以解锁智能日程生成功能</p>
            <el-button type="primary" @click="$router.push('/settings')">
              前往设置
              <el-icon class="el-icon--right"><ArrowRight /></el-icon>
            </el-button>
          </div>
        </div>
      </div>

      <div class="side-column">
        <div class="card quick-actions-card">
          <div class="card-header">
            <h3>快速操作</h3>
          </div>
          <div class="quick-actions">
            <el-button type="primary" size="large" @click="$emit('add-task')">
              <el-icon><Plus /></el-icon>
              <span>创建任务</span>
            </el-button>
            <el-button size="large" @click="$emit('start-timer')">
              <el-icon><VideoPlay /></el-icon>
              <span>开始番茄</span>
            </el-button>
          </div>
        </div>

        <!-- AI 优先级建议 -->
        <div class="card priority-card" v-if="aiStore.configured">
          <div class="card-header">
            <h3>优先级建议</h3>
            <el-button
              size="small"
              :loading="aiStore.loading"
              @click="getPrioritySuggestions"
            >
              获取建议
            </el-button>
          </div>
          <div v-if="priorityTasks.length > 0" class="priority-list">
            <div
              v-for="(task, index) in priorityTasks"
              :key="task.id"
              class="priority-item"
            >
              <span class="priority-rank" :class="`rank-${index + 1}`">{{ index + 1 }}</span>
              <span class="priority-title">{{ task.title }}</span>
            </div>
          </div>
          <div v-else class="priority-empty">
            <p>点击"获取建议"查看任务优先级排序</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { ArrowRight, MagicStick, Plus, VideoPlay } from '@element-plus/icons-vue'
import TaskCard from '@/components/tasks/TaskCard.vue'
import { useTaskStore } from '@/stores/task'
import { useTimerStore } from '@/stores/timer'
import { useAIStore } from '@/stores/ai'
import type { Task, ScheduleEvent } from '@/types'

const taskStore = useTaskStore()
const timerStore = useTimerStore()
const aiStore = useAIStore()

defineEmits<{
  'add-task': []
  'edit-task': [task: Task]
  'complete-task': [id: string]
  'delete-task': [id: string]
  'start-timer': []
}>()

const todayPomodoros = ref(0)
const focusTime = ref(0)
const completedTasks = ref(0)

const scheduleStartTime = ref('09:00')
const scheduleEndTime = ref('18:00')
const generatedSchedule = ref<ScheduleEvent[] | null>(null)
const priorityTasks = ref<Task[]>([])

const pendingTasks = computed(() =>
  taskStore.tasks.filter(t => t.status === 'todo').length
)

const recentTasks = computed(() =>
  taskStore.tasks.slice(0, 5)
)

function formatDuration(seconds: number): string {
  const mins = Math.floor(seconds / 60)
  const hours = Math.floor(mins / 60)
  const remainingMins = mins % 60

  if (hours > 0) {
    return remainingMins > 0 ? `${hours}h ${remainingMins}m` : `${hours}h`
  }
  return `${mins}m`
}

function formatScheduleTime(dateStr: string): string {
  const date = new Date(dateStr)
  return `${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}`
}

async function generateSchedule() {
  if (!scheduleStartTime.value || !scheduleEndTime.value) {
    ElMessage.warning('请选择时间范围')
    return
  }

  try {
    const result = await aiStore.generateSchedule(scheduleStartTime.value, scheduleEndTime.value)
    if (result) {
      generatedSchedule.value = result
      ElMessage.success('日程已生成')
    }
  } catch (error) {
    ElMessage.error('生成日程失败')
  }
}

async function getPrioritySuggestions() {
  try {
    const result = await aiStore.getPrioritySuggestions()
    if (result && result.priority_order) {
      const taskMap = new Map(taskStore.tasks.map(t => [t.id, t]))
      priorityTasks.value = result.priority_order
        .map(id => taskMap.get(id))
        .filter((t): t is Task => t !== undefined && t.status === 'todo')
        .slice(0, 5)
    }
  } catch (error) {
    ElMessage.error('获取优先级建议失败')
  }
}

onMounted(async () => {
  await taskStore.fetchTasks()
  await timerStore.fetchRecentSessions()

  const today = new Date().toDateString()
  const todaySessions = timerStore.recentSessions.filter(s =>
    new Date(s.start_time).toDateString() === today
  )
  todayPomodoros.value = todaySessions.filter(s => s.type === 'work' && s.status === 'completed').length
  focusTime.value = todaySessions.reduce((sum, s) => {
    if (s.type === 'work' && s.actual_duration) {
      return sum + s.actual_duration
    }
    return sum
  }, 0)
  completedTasks.value = taskStore.tasks.filter(t =>
    t.status === 'completed' &&
    new Date(t.completed_at!).toDateString() === today
  ).length
})
</script>

<style scoped>
.dashboard {
  padding: 32px;
  max-width: 1400px;
  margin: 0 auto;
}

.page-header {
  margin-bottom: 32px;
}

.page-header h1 {
  font-size: 28px;
  font-weight: 700;
  color: #1e293b;
  margin: 0 0 4px 0;
}

.page-subtitle {
  font-size: 15px;
  color: #64748b;
  margin: 0;
}

.stats-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
  margin-bottom: 32px;
}

.stat-card {
  background: #fff;
  border-radius: 16px;
  padding: 24px;
  display: flex;
  align-items: center;
  gap: 16px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.stat-icon svg {
  width: 24px;
  height: 24px;
}

.stat-icon.pomodoro {
  background: linear-gradient(135deg, #ef4444 0%, #dc2626 100%);
  color: #fff;
}

.stat-icon.focus {
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
  color: #fff;
}

.stat-icon.completed {
  background: linear-gradient(135deg, #22c55e 0%, #16a34a 100%);
  color: #fff;
}

.stat-icon.pending {
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
  color: #fff;
}

.stat-value {
  font-size: 32px;
  font-weight: 700;
  color: #1e293b;
  line-height: 1;
}

.stat-label {
  font-size: 14px;
  color: #64748b;
  margin-top: 4px;
}

.dashboard-content {
  display: grid;
  grid-template-columns: 1fr 320px;
  gap: 24px;
}

.main-column {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.side-column {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.card {
  background: #fff;
  border-radius: 16px;
  padding: 24px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.card-header h3 {
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
  margin: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.ai-icon {
  width: 20px;
  height: 20px;
  color: #8b5cf6;
}

.empty-state {
  text-align: center;
  padding: 48px 24px;
}

.empty-icon {
  width: 64px;
  height: 64px;
  color: #cbd5e1;
  margin-bottom: 16px;
}

.empty-state p {
  color: #64748b;
  margin-bottom: 16px;
}

.task-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.quick-actions-card .quick-actions {
  display: flex;
  flex-direction: column;
  gap: 12px;
  align-items: stretch;
}

.quick-actions .el-button {
  width: 100%;
  height: 48px;
  font-size: 15px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.quick-actions .el-button .el-icon {
  font-size: 16px;
}

.ai-section .schedule-form {
  display: flex;
  gap: 12px;
  align-items: flex-end;
  margin-bottom: 20px;
  flex-wrap: wrap;
}

.time-inputs {
  display: flex;
  align-items: center;
  gap: 8px;
}

.time-separator {
  color: #64748b;
  font-size: 14px;
}

.schedule-result {
  margin-top: 16px;
}

.schedule-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  background: #f8fafc;
  border-radius: 10px;
  margin-bottom: 8px;
  transition: all 0.2s ease;
}

.schedule-item:hover {
  background: #f1f5f9;
}

.schedule-time {
  font-size: 13px;
  color: #3b82f6;
  font-weight: 600;
  min-width: 100px;
}

.schedule-task {
  flex: 1;
  font-weight: 500;
  color: #334155;
}

.schedule-pomodoros {
  font-size: 12px;
  color: #64748b;
  background: #e0f2fe;
  padding: 2px 8px;
  border-radius: 4px;
}

.ai-placeholder {
  text-align: center;
  padding: 32px 24px;
  background: #f8fafc;
  border-radius: 12px;
  border: 1px dashed #cbd5e1;
}

.ai-placeholder p {
  color: #64748b;
  margin-bottom: 16px;
}

.priority-card .priority-list {
  margin-top: 8px;
}

.priority-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 0;
  border-bottom: 1px solid #f1f5f9;
}

.priority-item:last-child {
  border-bottom: none;
}

.priority-rank {
  width: 28px;
  height: 28px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  font-weight: 700;
  flex-shrink: 0;
}

.priority-rank.rank-1 {
  background: linear-gradient(135deg, #fbbf24 0%, #f59e0b 100%);
  color: #fff;
}

.priority-rank.rank-2 {
  background: linear-gradient(135deg, #94a3b8 0%, #64748b 100%);
  color: #fff;
}

.priority-rank.rank-3 {
  background: linear-gradient(135deg, #cd7f32 0%, #b8860b 100%);
  color: #fff;
}

.priority-rank.rank-4,
.priority-rank.rank-5 {
  background: #e2e8f0;
  color: #475569;
}

.priority-title {
  flex: 1;
  font-size: 14px;
  color: #334155;
}

.priority-empty {
  text-align: center;
  padding: 24px;
  color: #94a3b8;
  font-size: 14px;
}

/* 响应式 */
@media (max-width: 1200px) {
  .stats-cards {
    grid-template-columns: repeat(2, 1fr);
  }

  .dashboard-content {
    grid-template-columns: 1fr;
  }

  .side-column {
    flex-direction: row;
  }

  .side-column .card {
    flex: 1;
  }
}

@media (max-width: 768px) {
  .dashboard {
    padding: 20px;
  }

  .stats-cards {
    grid-template-columns: 1fr;
  }

  .side-column {
    flex-direction: column;
  }
}
</style>