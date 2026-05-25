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
            <el-button type="primary" @click="router.push('/tasks?add=true')">创建第一个任务</el-button>
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
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowRight, Plus, VideoPlay } from '@element-plus/icons-vue'
import TaskCard from '@/components/tasks/TaskCard.vue'
import { useTaskStore } from '@/stores/task'
import { useTimerStore } from '@/stores/timer'
import { useAIStore } from '@/stores/ai'
import type { Task } from '@/types'

const router = useRouter()
const taskStore = useTaskStore()
const timerStore = useTimerStore()
const aiStore = useAIStore()

defineEmits<{
  'edit-task': [task: Task]
  'complete-task': [id: string]
  'delete-task': [id: string]
  'start-timer': []
}>()

const todayPomodoros = ref(0)
const focusTime = ref(0)
const completedTasks = ref(0)

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

async function startTimer() {
  try {
    await timerStore.createSession(null, 'work')
    ElMessage.success('开始番茄计时')
    router.push('/timer')
  } catch {
    ElMessage.error('启动计时器失败')
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
  max-width: 1100px;
  margin: 0 auto;
}

.page-header {
  margin-bottom: 48px;
}

.page-header h1 {
  font-family: var(--font-display);
  font-size: 32px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 8px 0;
  letter-spacing: -0.5px;
}

.page-subtitle {
  font-size: 14px;
  color: var(--text-muted);
  margin: 0;
  font-weight: 400;
}

/* 统计卡片 */
.stats-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 48px;
}

.stat-card {
  background: var(--bg-card);
  border-radius: var(--radius-xl);
  padding: 24px 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  border: 1px solid var(--border-color);
  transition: all var(--transition-normal);
}

.stat-card:hover {
  border-color: var(--border-accent);
  box-shadow: 0 2px 16px rgba(0, 0, 0, 0.04);
  transform: translateY(-1px);
}

.stat-icon {
  width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.stat-icon svg {
  width: 20px;
  height: 20px;
}

.stat-icon.pomodoro {
  background: rgba(184, 69, 44, 0.08);
  color: var(--accent-primary);
}

.stat-icon.focus {
  background: rgba(184, 149, 77, 0.08);
  color: var(--accent-gold);
}

.stat-icon.completed {
  background: rgba(107, 139, 111, 0.08);
  color: var(--accent-sage);
}

.stat-icon.pending {
  background: rgba(0, 0, 0, 0.04);
  color: var(--text-secondary);
}

.stat-content {
  display: flex;
  flex-direction: column;
  justify-content: center;
  height: 44px;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: var(--text-primary);
  line-height: 1;
  font-family: var(--font-mono);
  letter-spacing: -0.5px;
}

.stat-label {
  font-size: 12px;
  color: var(--text-muted);
  margin-top: 4px;
  font-weight: 400;
}

.dashboard-content {
  display: grid;
  grid-template-columns: 1fr 300px;
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
  gap: 16px;
}

.card {
  background: var(--bg-card);
  border-radius: var(--radius-xl);
  padding: 24px;
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
  margin-bottom: 20px;
}

.card-header h3 {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
  letter-spacing: -0.2px;
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
  opacity: 0.5;
}

.empty-state p {
  color: var(--text-muted);
  margin-bottom: 16px;
  font-size: 14px;
}

.task-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
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
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  margin-left: 0;
}

.quick-actions .el-button:hover {
  background: rgba(0, 0, 0, 0.03);
  border-color: var(--border-accent);
}

.quick-actions .el-button--primary {
  background: var(--accent-primary);
  border: none;
  color: #fff;
}

.quick-actions .el-button--primary:hover {
  background: var(--accent-secondary);
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
  gap: 12px;
  padding: 10px 0;
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
  width: 24px;
  height: 24px;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
  flex-shrink: 0;
  font-family: var(--font-mono);
}

.priority-rank.rank-1 {
  background: rgba(184, 69, 44, 0.10);
  color: var(--accent-primary);
}

.priority-rank.rank-2 {
  background: rgba(0, 0, 0, 0.05);
  color: var(--text-secondary);
}

.priority-rank.rank-3 {
  background: rgba(0, 0, 0, 0.05);
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
  color: var(--text-primary);
  font-weight: 500;
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
}
</style>
