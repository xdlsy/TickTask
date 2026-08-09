<template>
  <div class="dashboard">
    <div class="page-header">
      <span class="eyebrow">Overview</span>
      <h1>仪表盘</h1>
      <p class="page-subtitle">欢迎回来，今天也要高效工作 — 保持节奏，把重要的事先做完。</p>
    </div>

    <div class="stats-cards">
      <div class="stat-card">
        <div class="stat-top">
          <div class="stat-icon pomodoro">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/>
              <path d="M12 6v6l4 2"/>
            </svg>
          </div>
        </div>
        <div class="stat-value">{{ todayPomodoros }}</div>
        <div class="stat-label">今日番茄</div>
      </div>
      <div class="stat-card">
        <div class="stat-top">
          <div class="stat-icon focus">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="4"/>
              <path d="M12 2v2M12 20v2M2 12h2M20 12h2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/>
            </svg>
          </div>
        </div>
        <div class="stat-value" v-html="formatDurationHtml(focusTime)"></div>
        <div class="stat-label">专注时长</div>
      </div>
      <div class="stat-card">
        <div class="stat-top">
          <div class="stat-icon completed">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
              <polyline points="22 4 12 14.01 9 11.01"/>
            </svg>
          </div>
        </div>
        <div class="stat-value">{{ completedTasks }}</div>
        <div class="stat-label">完成任务</div>
      </div>
      <div class="stat-card">
        <div class="stat-top">
          <div class="stat-icon pending">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="4" width="18" height="18" rx="2" ry="2"/>
              <line x1="16" y1="2" x2="16" y2="6"/>
              <line x1="8" y1="2" x2="8" y2="6"/>
              <line x1="3" y1="10" x2="21" y2="10"/>
            </svg>
          </div>
        </div>
        <div class="stat-value">{{ pendingTasks }}</div>
        <div class="stat-label">待办任务</div>
      </div>
    </div>

    <div class="dashboard-content">
      <div class="main-column">
        <div class="card recent-tasks-card">
          <div class="card-header">
            <h3>最近任务</h3>
            <button class="header-link" @click="$router.push('/tasks')">
              查看全部
              <el-icon class="el-icon--right"><ArrowRight /></el-icon>
            </button>
          </div>
          <div v-if="recentTasks.length === 0" class="empty-state">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="empty-icon">
              <rect x="3" y="3" width="18" height="18" rx="2"/>
              <path d="M9 12l2 2 4-4"/>
            </svg>
            <p>暂无任务</p>
            <el-button type="primary" @click="router.push('/tasks?add=true')">创建第一个任务</el-button>
          </div>
          <div v-else class="task-list">
            <TaskCard
              v-for="task in recentTasks"
              :key="task.id"
              :task="task"
              @drag-start="() => {}"
              @edit="onEditTask"
              @complete="onCompleteTask"
              @delete="onDeleteTask"
            />
          </div>
        </div>
      </div>

      <div class="side-column">
        <div class="card quick-actions-card">
          <div class="card-header">
            <h3>快速操作</h3>
          </div>
          <div class="quick-actions">
            <el-button type="primary" size="large" @click="router.push('/tasks?add=true')">
              <el-icon><Plus /></el-icon>
              <span>创建任务</span>
            </el-button>
            <el-button size="large" @click="startTimer">
              <el-icon><VideoPlay /></el-icon>
              <span>开始番茄</span>
            </el-button>
          </div>
        </div>

        <div class="card priority-card" v-if="agentStore.status.configured">
          <div class="card-header">
            <h3>AI 助手</h3>
            <button
              class="header-link"
              @click="openAgent"
            >
              打开助手
            </button>
          </div>
          <div class="priority-empty">
            <p>使用顶栏的 AI 助手获取任务优先级建议、智能分类等功能</p>
          </div>
        </div>
      </div>
    </div>

    <TaskForm
      v-if="showForm"
      :visible="showForm"
      :task="editingTask"
      @close="showForm = false"
      @save="onSaveTask"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowRight, Plus, VideoPlay } from '@element-plus/icons-vue'
import TaskCard from '@/components/tasks/TaskCard.vue'
import TaskForm from '@/components/tasks/TaskForm.vue'
import { useTaskStore } from '@/stores/task'
import { useTimerStore } from '@/stores/timer'
import { useAgentStore } from '@/stores/agent'
import type { TaskResponse } from '@/types'

const router = useRouter()
const taskStore = useTaskStore()
const timerStore = useTimerStore()
const agentStore = useAgentStore()

const showForm = ref(false)
const editingTask = ref<TaskResponse | null>(null)

function onEditTask(task: TaskResponse) {
  editingTask.value = task
  showForm.value = true
}

async function onCompleteTask(id: string) {
  await taskStore.markCompleted(id)
  ElMessage.success('任务已完成')
}

async function onDeleteTask(id: string) {
  await taskStore.deleteTask(id)
  ElMessage.success('任务已删除')
}

async function onSaveTask(data: any) {
  if (editingTask.value) {
    await taskStore.updateTask(editingTask.value.id, data)
  }
  showForm.value = false
  editingTask.value = null
}

const todayPomodoros = ref(0)
const focusTime = ref(0)
const completedTasks = ref(0)

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

/* 渲染为带弱化单位的 HTML,供大号数字展示用(基于 formatDuration 注入单位 span) */
function formatDurationHtml(seconds: number): string {
  return formatDuration(seconds)
    .replace(/(\d+)h/g, '$1<span class="unit">h</span>')
    .replace(/(\d+)m/g, '$1<span class="unit">m</span>')
}

async function startTimer() {
  try {
    await timerStore.createSession(null, 'work')
    ElMessage.success('开始番茄计时')
    router.push('/timer')
  } catch {
    ElMessage.error('启动计时器失败')
  }
}

function openAgent() {
  agentStore.openDrawer()
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
  max-width: 1120px;
  margin: 0 auto;
}

.page-header {
  margin-bottom: 48px;
}

.page-header h1 {
  font-family: var(--font-display);
  font-variation-settings: 'opsz' 144;
  font-size: 46px;
  font-weight: 380;
  color: var(--text-primary);
  margin: 0 0 12px 0;
  letter-spacing: -0.035em;
  line-height: 1.04;
}

.page-header h1 em {
  font-style: italic;
  font-weight: 360;
  color: var(--text-secondary);
}

.page-subtitle {
  font-size: 14.5px;
  color: var(--text-muted);
  margin: 0;
  font-weight: 400;
  max-width: 520px;
}

/* 编辑式 eyebrow:细发丝线 + 等宽小标签,衬在标题之上 */
.eyebrow {
  display: inline-flex;
  align-items: center;
  gap: 11px;
  font-family: var(--font-mono);
  font-size: 10.5px;
  font-weight: 500;
  letter-spacing: 0.28em;
  text-transform: uppercase;
  color: var(--accent-primary);
  margin-bottom: 18px;
}

.eyebrow::before {
  content: '';
  width: 26px;
  height: 1px;
  background: var(--accent-primary);
  opacity: 0.6;
}

/* 统计卡片 */
.stats-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 14px;
  margin-bottom: 48px;
}

.stat-card {
  position: relative;
  background: var(--gradient-card);
  border-radius: var(--radius-lg);
  padding: 22px 20px 20px;
  border: 1px solid var(--border-color);
  overflow: hidden;
  transition: transform var(--transition-slow), border-color var(--transition-slow), box-shadow var(--transition-slow);
}

.stat-card::after {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  height: 2px;
  width: 100%;
  background: linear-gradient(90deg, var(--accent-primary), transparent 60%);
  opacity: 0;
  transition: opacity var(--transition-slow);
}

.stat-card:hover {
  transform: translateY(-3px);
  border-color: var(--border-accent);
  box-shadow: var(--shadow-card-hover);
}

.stat-card:hover::after {
  opacity: 1;
}

.stat-top {
  margin-bottom: 24px;
}

.stat-icon {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.stat-icon svg {
  width: 16px;
  height: 16px;
}

.stat-icon.pomodoro {
  background: var(--accent-fill);
  color: var(--accent-primary);
}

.stat-icon.focus {
  background: var(--gold-fill);
  color: var(--accent-gold);
}

.stat-icon.completed {
  background: var(--sage-fill);
  color: var(--accent-sage);
}

.stat-icon.pending {
  background: rgba(239, 231, 215, 0.05);
  color: var(--text-secondary);
}

.stat-value {
  font-family: var(--font-display);
  font-variation-settings: 'opsz' 144;
  font-size: 48px;
  font-weight: 360;
  color: var(--text-primary);
  line-height: 0.95;
  letter-spacing: -0.04em;
  font-feature-settings: 'tnum';
}

.stat-value :deep(.unit) {
  font-size: 18px;
  color: var(--text-muted);
  margin-left: 3px;
  font-weight: 400;
}

.stat-label {
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--text-muted);
  margin-top: 12px;
  font-weight: 500;
  letter-spacing: 0.18em;
  text-transform: uppercase;
}

.dashboard-content {
  display: grid;
  grid-template-columns: 1fr 320px;
  gap: 18px;
}

.main-column {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.side-column {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.card {
  background: var(--gradient-card);
  border-radius: var(--radius-xl);
  padding: 26px;
  border: 1px solid var(--border-color);
  transition: border-color var(--transition-normal);
}

.card:hover {
  border-color: var(--border-accent);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 22px;
}

.card-header h3 {
  font-family: var(--font-display);
  font-variation-settings: 'opsz' 60;
  font-size: 19px;
  font-weight: 420;
  color: var(--text-primary);
  margin: 0;
  letter-spacing: -0.02em;
}

.header-link {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-family: var(--font-mono);
  font-size: 10.5px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--accent-primary);
  background: none;
  border: none;
  cursor: pointer;
  transition: gap var(--transition-fast), color var(--transition-fast);
  padding: 0;
}

.header-link:hover:not(:disabled) {
  gap: 9px;
  color: var(--accent-secondary);
}

.header-link:disabled {
  opacity: 0.6;
  cursor: progress;
}

.empty-state {
  text-align: center;
  padding: 48px 32px;
  border-radius: var(--radius-md);
}

.empty-icon {
  width: 48px;
  height: 48px;
  color: var(--text-muted);
  margin: 0 auto 16px;
  opacity: 0.4;
}

.empty-state p {
  color: var(--text-muted);
  margin-bottom: 16px;
  font-size: 14px;
}

.task-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.quick-actions-card .quick-actions {
  display: flex;
  flex-direction: column;
  gap: 10px;
  align-items: stretch;
}

.quick-actions .el-button {
  width: 100%;
  height: 48px;
  font-size: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border-radius: var(--radius-md);
  font-weight: 500;
  transition: all var(--transition-normal);
  background: var(--bg-secondary);
  border: 1px solid var(--border-accent);
  color: var(--text-secondary);
  margin-left: 0;
}

.quick-actions .el-button:hover {
  background: var(--bg-card-hover);
  border-color: var(--border-strong);
  color: var(--text-primary);
}

.quick-actions .el-button .el-icon {
  font-size: 16px;
}

.priority-card .priority-list {
  margin-top: 4px;
}

.priority-item {
  display: flex;
  align-items: center;
  gap: 13px;
  padding: 12px 0;
  border-bottom: 1px solid var(--border-color);
  transition: padding var(--transition-fast);
}

.priority-item:last-child {
  border-bottom: none;
}

.priority-item:hover {
  padding-left: 4px;
}

.priority-rank {
  width: 26px;
  height: 26px;
  border-radius: 7px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 600;
  flex-shrink: 0;
  font-family: var(--font-mono);
}

.priority-rank.rank-1 {
  background: var(--accent-primary);
  color: var(--bg-primary);
  box-shadow: 0 0 14px rgba(230, 162, 60, 0.4);
}

.priority-rank.rank-2 {
  background: var(--accent-fill);
  color: var(--accent-primary);
}

.priority-rank.rank-3 {
  background: var(--bg-elevated);
  color: var(--text-secondary);
}

.priority-rank.rank-4,
.priority-rank.rank-5 {
  background: transparent;
  color: var(--text-muted);
}

.priority-title {
  flex: 1;
  font-size: 13px;
  color: var(--text-secondary);
  font-weight: 450;
}

.priority-empty {
  text-align: center;
  padding: 20px;
  color: var(--text-muted);
  font-size: 13px;
}

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
  .stats-cards {
    grid-template-columns: 1fr;
  }

  .side-column {
    flex-direction: column;
  }

  .page-header h1 {
    font-size: 36px;
  }
}
</style>
