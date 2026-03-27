<template>
  <div class="list-view">
    <!-- 筛选和排序栏 -->
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

    <!-- 任务列表 -->
    <div class="task-list">
      <div v-if="filteredTasks.length === 0" class="empty-state">
        <div class="empty-icon">📋</div>
        <p>暂无任务</p>
        <p class="empty-hint">点击右上角添加你的第一个任务</p>
      </div>

      <div
        v-for="task in filteredTasks"
        :key="task.id"
        class="task-item"
        :class="{ completed: task.status === 'completed' }"
      >
        <!-- 完成勾选 -->
        <div class="task-checkbox" @click="onCompleteTask(task.id)">
          <span v-if="task.status === 'completed'" class="check-icon">✓</span>
        </div>

        <!-- 任务内容 -->
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
                📅 {{ formatDate(task.deadline) }}
              </span>
              <span v-if="task.estimated_time" class="tag time">
                ⏱ {{ task.estimated_time }} 分钟
              </span>
            </div>
          </div>
        </div>

        <!-- 操作按钮 -->
        <div class="task-actions">
          <el-dropdown trigger="click" @command="(cmd: string) => handleCommand(cmd, task)">
            <span class="action-btn">⋯</span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item v-if="task.status !== 'completed'" command="startTimer">
                  🍅 开始番茄
                </el-dropdown-item>
                <el-dropdown-item command="edit">编辑</el-dropdown-item>
                <el-dropdown-item v-if="task.status !== 'completed'" command="complete">标记完成</el-dropdown-item>
                <el-dropdown-item v-if="task.status === 'completed'" command="reopen">重新打开</el-dropdown-item>
                <el-dropdown-item command="delete" divided style="color: #ef4444">删除</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
    </div>

    <!-- 任务表单 -->
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
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import TaskForm from './TaskForm.vue'
import { useTaskStore } from '@/stores/task'
import { useTimerStore } from '@/stores/timer'
import type { Task, Quadrant } from '@/types'

const router = useRouter()
const taskStore = useTaskStore()
const timerStore = useTimerStore()

const showForm = ref(false)
const editingTask = ref<Task | null>(null)

// 筛选和排序
const statusFilter = ref('')
const quadrantFilter = ref('')
const sortBy = ref('created')

// 获取任务列表
const tasks = computed(() => taskStore.tasks)

// 筛选后的任务
const filteredTasks = computed(() => {
  let result = [...tasks.value]

  // 状态筛选
  if (statusFilter.value) {
    result = result.filter(t => t.status === statusFilter.value)
  }

  // 象限筛选
  if (quadrantFilter.value) {
    result = result.filter(t => t.quadrant === parseInt(quadrantFilter.value))
  }

  // 排序
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
  editingTask.value = task
  showForm.value = true
}

async function onCompleteTask(id: string) {
  const task = tasks.value.find(t => t.id === id)
  if (task?.status === 'completed') {
    // 已完成，重新打开
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

// 暴露添加任务方法
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
  padding: 16px 24px;
  height: 100%;
  display: flex;
  flex-direction: column;
}

/* 工具栏 */
.list-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid #e5e7eb;
}

.filter-group,
.sort-group {
  display: flex;
  gap: 12px;
}

/* 任务列表 */
.task-list {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.empty-state {
  text-align: center;
  padding: 60px 24px;
}

.empty-icon {
  font-size: 48px;
  margin-bottom: 16px;
}

.empty-state p {
  color: #6b7280;
  margin: 0 0 4px 0;
}

.empty-hint {
  font-size: 13px;
  color: #9ca3af !important;
}

/* 任务项 */
.task-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 14px 16px;
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.task-item:hover {
  border-color: #d1d5db;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.task-item.completed {
  background: #f9fafb;
  opacity: 0.7;
}

.task-item.completed .task-title {
  text-decoration: line-through;
  color: #9ca3af;
}

/* 勾选框 */
.task-checkbox {
  width: 20px;
  height: 20px;
  border: 2px solid #d1d5db;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  flex-shrink: 0;
  margin-top: 2px;
  transition: all 0.2s ease;
}

.task-checkbox:hover {
  border-color: #3b82f6;
}

.task-item.completed .task-checkbox {
  background: #22c55e;
  border-color: #22c55e;
}

.check-icon {
  color: #fff;
  font-size: 12px;
  font-weight: bold;
}

/* 任务主体 */
.task-main {
  flex: 1;
  min-width: 0;
}

.task-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 6px;
}

.task-title {
  font-size: 15px;
  font-weight: 600;
  color: #1f2937;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-quadrant {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 4px;
  font-weight: 500;
  flex-shrink: 0;
}

.task-quadrant.quadrant-1 {
  background: #fef2f2;
  color: #dc2626;
}

.task-quadrant.quadrant-2 {
  background: #fffbeb;
  color: #d97706;
}

.task-quadrant.quadrant-3 {
  background: #eff6ff;
  color: #2563eb;
}

.task-quadrant.quadrant-4 {
  background: #f3f4f6;
  color: #6b7280;
}

.task-meta {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.task-desc {
  font-size: 13px;
  color: #6b7280;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-tags {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.tag {
  font-size: 12px;
  color: #6b7280;
}

.tag.deadline {
  color: #6b7280;
}

.tag.deadline.overdue {
  color: #ef4444;
}

/* 操作按钮 */
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
  color: #9ca3af;
  font-weight: bold;
  letter-spacing: 2px;
}

.action-btn:hover {
  background: #f3f4f6;
  color: #6b7280;
}

/* 响应式 */
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