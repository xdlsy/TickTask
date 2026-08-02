<template>
  <div class="list-view">
    <div class="list-toolbar">
      <div class="filter-group">
        <el-select v-model="statusFilter" placeholder="状态筛选" clearable size="small" style="width: 120px">
          <el-option label="全部" value="" />
          <el-option label="待办" value="todo" />
          <el-option label="进行中" value="in_progress" />
          <el-option label="已完成" value="completed" />
        </el-select>

        <el-select v-model="quadrantFilter" placeholder="象限筛选" clearable size="small" style="width: 140px">
          <el-option label="全部象限" value="" />
          <el-option label="重要且紧急" value="1" />
          <el-option label="重要不紧急" value="2" />
          <el-option label="紧急不重要" value="3" />
          <el-option label="不重要不紧急" value="4" />
        </el-select>
      </div>

      <div class="sort-group">
        <el-select v-model="sortBy" placeholder="排序" size="small" style="width: 120px">
          <el-option label="创建时间" value="created" />
          <el-option label="截止日期" value="deadline" />
          <el-option label="优先级" value="quadrant" />
        </el-select>
      </div>
    </div>

    <div class="task-list">
      <div v-if="filteredTasks.length === 0" class="empty-state">
        <div class="empty-icon">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" width="48" height="48">
            <rect x="3" y="3" width="18" height="18" rx="2"/>
            <line x1="9" y1="9" x2="15" y2="9"/>
            <line x1="9" y1="13" x2="15" y2="13"/>
            <line x1="9" y1="17" x2="12" y2="17"/>
          </svg>
        </div>
        <p>暂无任务</p>
        <p class="empty-hint">点击右上角添加你的第一个任务</p>
      </div>

      <div
        v-for="task in filteredTasks"
        :key="task.id"
        class="task-item"
        :class="{ completed: task.status === 'completed' }"
      >
        <div class="task-checkbox" @click="onCompleteTask(task.id)">
          <svg v-if="task.status === 'completed'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" width="14" height="14" class="check-icon">
            <polyline points="20 6 9 17 4 12"/>
          </svg>
        </div>

        <div class="task-main" @click="onEditTask(task)">
          <div class="task-header">
            <span class="task-title">{{ task.title }}</span>
            <span class="task-quadrant" :class="`quadrant-${task.quadrant}`">
              {{ quadrantLabel(task.quadrant) }}
            </span>
          </div>

          <div class="task-meta">
            <span v-if="task.description" class="task-desc">{{ task.description }}</span>
            <div class="task-tags">
              <span v-if="task.deadline" class="tag deadline" :class="{ overdue: isOverdue(task.deadline) }">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" width="13" height="13" class="tag-icon"><rect x="3" y="4" width="18" height="18" rx="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>
                {{ formatDate(task.deadline) }}
              </span>
              <span v-if="task.estimated_time" class="tag time">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" width="13" height="13" class="tag-icon"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
                {{ task.estimated_time }} 分钟
              </span>
              <span v-if="task.planned_pomodoros > 0" class="tag pomodoro-tag">{{ task.completed_pomodoros }}/{{ task.planned_pomodoros }} 番茄钟</span>
              <span v-else class="tag pomodoro-tag pomodoro-na">—</span>
            </div>
          </div>
        </div>

        <div class="task-actions">
          <el-dropdown trigger="click" @command="(cmd: string) => handleCommand(cmd, task)">
            <span class="action-btn">
              <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16"><circle cx="12" cy="5" r="2"/><circle cx="12" cy="12" r="2"/><circle cx="12" cy="19" r="2"/></svg>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item v-if="task.status !== 'completed'" command="startTimer">开始番茄</el-dropdown-item>
                <el-dropdown-item command="edit">编辑</el-dropdown-item>
                <el-dropdown-item v-if="task.status !== 'completed'" command="complete">标记完成</el-dropdown-item>
                <el-dropdown-item v-if="task.status === 'completed'" command="reopen">重新打开</el-dropdown-item>
                <el-dropdown-item command="delete" divided>删除</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
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

    <TaskPomodoroDetail ref="pomodoroDetailRef" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import TaskForm from './TaskForm.vue'
import TaskPomodoroDetail from './TaskPomodoroDetail.vue'
import { useTaskStore } from '@/stores/task'
import { useTimerStore } from '@/stores/timer'
import type { Task, TaskResponse, Quadrant } from '@/types'

const router = useRouter()
const taskStore = useTaskStore()
const timerStore = useTimerStore()

const showForm = ref(false)
const editingTask = ref<Task | null>(null)
const pomodoroDetailRef = ref()

const statusFilter = ref('')
const quadrantFilter = ref('')
const sortBy = ref('created')

const tasks = computed(() => taskStore.tasks)

const filteredTasks = computed(() => {
  let result = [...tasks.value]

  if (statusFilter.value) {
    result = result.filter(t => t.status === statusFilter.value)
  }

  if (quadrantFilter.value) {
    result = result.filter(t => t.quadrant === parseInt(quadrantFilter.value))
  }

  if (sortBy.value === 'created') {
    result.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
  } else if (sortBy.value === 'deadline') {
    result.sort((a, b) => {
      if (!a.deadline) return 1
      if (!b.deadline) return -1
      return new Date(a.deadline).getTime() - new Date(b.deadline).getTime()
    })
  } else if (sortBy.value === 'quadrant') {
    result.sort((a, b) => a.quadrant - b.quadrant)
  }

  return result
})

function quadrantLabel(quadrant: Quadrant): string {
  const labels: Record<Quadrant, string> = {
    1: '重要紧急',
    2: '重要不紧急',
    3: '紧急不重要',
    4: '不重要不紧急'
  }
  return labels[quadrant]
}

function formatDate(dateStr: string): string {
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}

function isOverdue(deadline: string): boolean {
  return new Date(deadline) < new Date()
}

function onEditTask(task: Task) {
  pomodoroDetailRef.value?.open(task as TaskResponse)
}

async function onCompleteTask(id: string) {
  const task = tasks.value.find(t => t.id === id)
  if (task?.status === 'completed') {
    await taskStore.updateTask(id, { status: 'todo' })
  } else {
    await taskStore.markCompleted(id)
  }
}

async function onSaveTask(data: any) {
  if (editingTask.value) {
    await taskStore.updateTask(editingTask.value.id, data)
  } else {
    await taskStore.createTask(data)
  }
  showForm.value = false
  editingTask.value = null
}

async function handleCommand(command: string, task: Task) {
  switch (command) {
    case 'startTimer':
      await startTimerForTask(task)
      break
    case 'edit':
      editingTask.value = task
      showForm.value = true
      break
    case 'complete':
      await taskStore.markCompleted(task.id)
      break
    case 'reopen':
      await taskStore.updateTask(task.id, { status: 'todo' })
      break
    case 'delete':
      await taskStore.deleteTask(task.id)
      break
  }
}

async function startTimerForTask(task: Task) {
  try {
    await timerStore.createSession(task.id, 'work')
    ElMessage.success(`开始专注：${task.title}`)
    router.push('/timer')
  } catch (error) {
    ElMessage.error('启动计时器失败')
  }
}

function onAddTask() {
  editingTask.value = null
  showForm.value = true
}

defineExpose({
  onAddTask
})
</script>

<style scoped>
.list-view {
  padding: 20px 24px;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.list-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 20px;
  border-bottom: 1px solid var(--border-color);
}

.filter-group,
.sort-group {
  display: flex;
  gap: 12px;
}

.task-list {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.empty-state {
  text-align: center;
  padding: 64px 24px;
  border-radius: var(--radius-lg);
  border: 1px dashed var(--border-color);
}

.empty-icon {
  margin-bottom: 16px;
  color: var(--text-muted);
  opacity: 0.4;
}

.empty-state p {
  color: var(--text-secondary);
  margin: 0 0 4px 0;
  font-size: 14px;
}

.empty-hint {
  font-size: 13px;
  color: var(--text-muted) !important;
}

.task-item {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  padding: 14px 18px;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: border-color var(--transition-fast);
}

.task-item:hover {
  border-color: var(--border-accent);
}

.task-item.completed {
  opacity: 0.5;
}

.task-item.completed .task-title {
  text-decoration: line-through;
  color: var(--text-muted);
}

.task-checkbox {
  width: 20px;
  height: 20px;
  border: 2px solid var(--border-accent);
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  flex-shrink: 0;
  margin-top: 1px;
  transition: all var(--transition-fast);
}

.task-checkbox:hover {
  border-color: var(--accent-primary);
}

.task-item.completed .task-checkbox {
  background: var(--accent-sage);
  border-color: var(--accent-sage);
}

.check-icon {
  color: #fff;
}

.task-main {
  flex: 1;
  min-width: 0;
}

.task-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 4px;
}

.task-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-quadrant {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  font-weight: 500;
  flex-shrink: 0;
}

.task-quadrant.quadrant-1 {
  background: rgba(184, 69, 44, 0.06);
  color: var(--accent-primary);
}

.task-quadrant.quadrant-2 {
  background: rgba(0, 0, 0, 0.04);
  color: var(--text-secondary);
}

.task-quadrant.quadrant-3 {
  background: rgba(0, 0, 0, 0.04);
  color: var(--text-secondary);
}

.task-quadrant.quadrant-4 {
  background: rgba(0, 0, 0, 0.03);
  color: var(--text-muted);
}

.task-meta {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.task-desc {
  font-size: 13px;
  color: var(--text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-tags {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.tag {
  font-size: 12px;
  color: var(--text-muted);
  display: flex;
  align-items: center;
  gap: 4px;
}

.tag-icon {
  flex-shrink: 0;
}

.tag.deadline.overdue {
  color: var(--accent-primary);
}

.pomodoro-tag {
  color: var(--accent-primary) !important;
}

.pomodoro-tag.pomodoro-na {
  color: var(--text-muted) !important;
}

.task-actions {
  flex-shrink: 0;
}

.action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 6px;
  cursor: pointer;
  color: var(--text-muted);
  transition: all var(--transition-fast);
}

.action-btn:hover {
  background: rgba(0, 0, 0, 0.04);
  color: var(--text-primary);
}

@media (max-width: 640px) {
  .list-view {
    padding: 12px 16px;
  }

  .list-toolbar {
    flex-direction: column;
    gap: 12px;
    align-items: flex-start;
  }

  .filter-group {
    width: 100%;
  }

  .filter-group .el-select {
    flex: 1;
  }
}
</style>
