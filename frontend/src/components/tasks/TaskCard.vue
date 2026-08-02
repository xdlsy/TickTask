<template>
  <el-popover v-if="mode === 'row'" trigger="hover" :show-after="300" :width="280" placement="top">
    <template #reference>
      <div class="task-row" :class="{ 'task-completed': task.status === 'completed' }" draggable="true" @dragstart="$emit('drag-start', $event, task)" @click="showDetail">
        <span class="row-checkbox" @click.stop="onRowCheckbox">
          <svg v-if="task.status === 'completed'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" width="14" height="14"><polyline points="20 6 9 17 4 12"/></svg>
        </span>
        <span class="task-title">{{ task.title }}</span>
        <span v-if="task.estimated_time" class="row-time">{{ task.estimated_time }}m</span>
        <span v-if="task.deadline" class="row-deadline">{{ formatDate(task.deadline) }}</span>
        <span v-if="task.planned_pomodoros > 0 && task.status === 'completed'" class="row-pomodoro row-pomodoro-done">{{ task.completed_pomodoros }}/{{ task.planned_pomodoros }} ✓</span>
        <span v-else-if="task.planned_pomodoros > 0" class="row-pomodoro">{{ task.completed_pomodoros }}/{{ task.planned_pomodoros }} 番茄钟</span>
        <span v-else class="row-pomodoro row-pomodoro-na">—</span>
        <span v-if="task.status !== 'completed'" class="row-pomodoro-btn" @click.stop="$emit('start-pomodoro', task.id)" title="开始番茄钟">▶</span>
        <span class="row-more" @click.stop>
          <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16"><circle cx="12" cy="5" r="2"/><circle cx="12" cy="12" r="2"/><circle cx="12" cy="19" r="2"/></svg>
        </span>
      </div>
    </template>
    <div class="row-popover-content">
      <div v-if="task.description" class="popover-desc">{{ task.description }}</div>
      <div class="popover-tags">
        <span v-if="task.estimated_time">预估 {{ task.estimated_time }} 分钟</span>
        <span v-if="task.deadline">截止 {{ formatDate(task.deadline) }}</span>
        <span>{{ statusLabel }}</span>
      </div>
    </div>
  </el-popover>
  <div v-else class="task-card" :class="{ 'task-completed': task.status === 'completed' }" draggable="true" @dragstart="$emit('drag-start', $event, task)" @click="showDetail">
    <div class="task-header">
      <span class="task-title">{{ task.title }}</span>
      <el-dropdown @command="handleCommand" trigger="click" @click.stop>
        <span class="more-icon" @click.stop>
          <svg viewBox="0 0 24 24" fill="currentColor" width="16" height="16"><circle cx="12" cy="5" r="2"/><circle cx="12" cy="12" r="2"/><circle cx="12" cy="19" r="2"/></svg>
        </span>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item v-if="task.status !== 'completed'" command="startTimer">开始番茄</el-dropdown-item>
            <el-dropdown-item command="edit">编辑</el-dropdown-item>
            <el-dropdown-item command="ai-classify" :disabled="aiStore.loading">AI 智能分类</el-dropdown-item>
            <el-dropdown-item command="complete" v-if="task.status !== 'completed'">完成</el-dropdown-item>
            <el-dropdown-item command="delete" divided>删除</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
    <div v-if="task.description" class="task-description">{{ task.description }}</div>
    <div class="task-meta">
      <el-tag v-if="task.estimated_time" size="small">{{ task.estimated_time }}分钟</el-tag>
      <el-tag v-if="task.deadline" type="warning" size="small">{{ formatDate(task.deadline) }}</el-tag>
      <el-tag size="small" :type="task.status === 'completed' ? 'success' : 'info'">{{ statusLabel }}</el-tag>
    </div>
    <el-dialog v-model="showClassifyResult" title="AI 分类建议" width="400px" @click.stop>
      <div v-if="classifyResult" class="classify-result">
        <div class="result-item"><span class="label">推荐象限：</span><el-tag :type="getQuadrantTagType(classifyResult.quadrant)">{{ getQuadrantName(classifyResult.quadrant) }}</el-tag></div>
        <div class="result-item"><span class="label">重要性：</span><span>{{ classifyResult.important ? '重要' : '不重要' }}</span></div>
        <div class="result-item"><span class="label">紧急度：</span><span>{{ classifyResult.urgent ? '紧急' : '不紧急' }}</span></div>
        <div class="result-reason"><span class="label">理由：</span><p>{{ classifyResult.reason }}</p></div>
      </div>
      <template #footer>
        <el-button @click="showClassifyResult = false">取消</el-button>
        <el-button type="primary" @click="applyClassification">采纳建议</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import type { TaskResponse, TaskStatus, ClassificationResult } from '@/types'
import { QUADRANT_INFO } from '@/types'
import { useAIStore } from '@/stores/ai'
import { useTaskStore } from '@/stores/task'
import { useTimerStore } from '@/stores/timer'
import { useRouter } from 'vue-router'

interface Props { task: TaskResponse; mode?: 'card' | 'row' }
const props = defineProps<Props>()
const emit = defineEmits<{ 'drag-start': [event: DragEvent, task: TaskResponse]; 'edit': [task: TaskResponse]; 'complete': [id: string]; 'delete': [id: string]; 'start-pomodoro': [taskId: string]; 'show-detail': [task: TaskResponse] }>()
const aiStore = useAIStore()
const taskStore = useTaskStore()
const timerStore = useTimerStore()
const router = useRouter()
const showClassifyResult = ref(false)
const classifyResult = ref<ClassificationResult | null>(null)
const statusLabels: Record<TaskStatus, string> = { todo: '待办', in_progress: '进行中', completed: '已完成', cancelled: '已取消' }
const statusLabel = computed(() => statusLabels[props.task.status])

function formatDate(d: string): string { const dt = new Date(d); return `${dt.getMonth() + 1}/${dt.getDate()}` }
function getQuadrantName(q: number): string { return QUADRANT_INFO[q as 1|2|3|4]?.name || `象限 ${q}` }
function getQuadrantTagType(q: number): string { return ({1:'danger',2:'warning',3:'primary',4:'info'} as Record<number,string>)[q] || 'info' }

async function handleCommand(cmd: string) {
  switch (cmd) {
    case 'startTimer': await startTimerForTask(); break
    case 'edit': emit('edit', props.task); break
    case 'complete': emit('complete', props.task.id); break
    case 'delete': emit('delete', props.task.id); break
    case 'ai-classify': await doClassify(); break
  }
}

async function startTimerForTask() {
  try { await timerStore.createSession(props.task.id, 'work'); ElMessage.success(`开始专注：${props.task.title}`); router.push('/timer') } catch { ElMessage.error('启动计时器失败') }
}

async function doClassify() {
  if (!aiStore.configured) { ElMessage.warning('请先在设置中配置 AI API Key'); router.push('/settings'); return }
  try { const r = await aiStore.classifyTask(props.task.id); if (r) { classifyResult.value = r; showClassifyResult.value = true } } catch { ElMessage.error('AI 分类失败') }
}

async function applyClassification() {
  if (classifyResult.value && classifyResult.value.quadrant !== props.task.quadrant) { await taskStore.moveTask(props.task.id, classifyResult.value.quadrant as 1|2|3|4); ElMessage.success('已应用 AI 分类建议') }
  showClassifyResult.value = false
}

function showDetail() { emit('show-detail', props.task) }
function onRowCheckbox() {
  if (props.task.status === 'completed') {
    taskStore.updateTask(props.task.id, { status: 'todo' })
  } else {
    emit('complete', props.task.id)
  }
}
</script>

<style scoped>
.task-card {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: 14px 16px;
  cursor: pointer;
  transition: border-color var(--transition-fast);
}

.task-card:hover {
  border-color: var(--border-accent);
}

.task-completed {
  opacity: 0.45;
}

.task-completed .task-title {
  text-decoration: line-through;
  color: var(--text-muted);
}

.task-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 8px;
}

.task-title {
  font-weight: 600;
  flex: 1;
  color: var(--text-primary);
  font-size: 14px;
  line-height: 1.4;
}

.more-icon {
  color: var(--text-muted);
  cursor: pointer;
  transition: all var(--transition-fast);
  padding: 4px;
  border-radius: 4px;
  display: flex;
}

.more-icon:hover {
  color: var(--text-primary);
  background: rgba(0, 0, 0, 0.04);
}

.task-description {
  color: var(--text-secondary);
  font-size: 13px;
  margin-top: 10px;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.task-meta {
  display: flex;
  gap: 6px;
  margin-top: 12px;
  flex-wrap: wrap;
}

.task-meta .el-tag {
  --el-tag-bg-color: rgba(0, 0, 0, 0.03);
  --el-tag-border-color: rgba(0, 0, 0, 0.06);
  --el-tag-text-color: var(--text-secondary);
}

.task-meta .el-tag--warning {
  --el-tag-bg-color: rgba(0, 0, 0, 0.03);
  --el-tag-border-color: rgba(0, 0, 0, 0.06);
  --el-tag-text-color: var(--text-secondary);
}

.task-meta .el-tag--success {
  --el-tag-bg-color: rgba(107, 139, 111, 0.06);
  --el-tag-border-color: rgba(107, 139, 111, 0.12);
  --el-tag-text-color: var(--accent-sage);
}

.classify-result { padding: 8px 0 }

.result-item {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.result-item .label {
  color: var(--text-secondary);
  min-width: 70px;
  font-size: 13px;
}

.result-reason {
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid var(--border-color);
}

.result-reason .label {
  color: var(--text-secondary);
  margin-bottom: 8px;
  display: block;
  font-size: 13px;
}

.result-reason p {
  margin: 0;
  line-height: 1.6;
  color: var(--text-primary);
}

/* Row mode */
.task-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 8px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background var(--transition-fast);
}

.task-row:hover {
  background: rgba(0, 0, 0, 0.03);
}

.task-row.task-completed {
  opacity: 0.45;
}

.task-row.task-completed .task-title {
  text-decoration: line-through;
  color: var(--text-muted);
}

.task-row .task-title {
  font-weight: 500;
  font-size: 13px;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-primary);
}

.row-checkbox {
  width: 14px;
  height: 14px;
  border: 1.5px solid var(--border-accent);
  border-radius: 3px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.row-checkbox:hover {
  border-color: var(--accent-primary);
}

.row-checkbox svg {
  color: var(--accent-sage);
}

.task-row.task-completed .row-checkbox {
  border-color: var(--accent-sage);
  background: var(--accent-sage);
}

.task-row.task-completed .row-checkbox svg {
  color: #fff;
}

.row-time {
  font-size: 11px;
  color: var(--text-muted);
  background: rgba(0, 0, 0, 0.04);
  padding: 1px 6px;
  border-radius: 4px;
  flex-shrink: 0;
}

.row-deadline {
  font-size: 11px;
  color: var(--text-muted);
  background: rgba(0, 0, 0, 0.04);
  padding: 1px 6px;
  border-radius: 4px;
  flex-shrink: 0;
}

.row-more {
  color: var(--text-muted);
  cursor: pointer;
  padding: 2px;
  border-radius: 4px;
  display: flex;
  flex-shrink: 0;
  transition: all var(--transition-fast);
}

.row-more:hover {
  color: var(--text-primary);
  background: rgba(0, 0, 0, 0.04);
}

.row-popover-content {
  font-size: 13px;
  line-height: 1.6;
}

.popover-desc {
  color: var(--text-secondary);
  margin-bottom: 8px;
}

.popover-tags {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  font-size: 12px;
  color: var(--text-muted);
}

.popover-tags span {
  background: rgba(0, 0, 0, 0.03);
  padding: 2px 8px;
  border-radius: 4px;
}

/* Pomodoro progress in row mode */
.row-pomodoro {
  font-size: 11px;
  color: var(--accent-primary);
  background: rgba(184, 69, 44, 0.06);
  padding: 1px 6px;
  border-radius: 4px;
  flex-shrink: 0;
  white-space: nowrap;
}

.row-pomodoro-btn {
  width: 28px;
  height: 28px;
  background: var(--accent-primary);
  color: white;
  border: none;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  cursor: pointer;
  flex-shrink: 0;
  transition: opacity var(--transition-fast);
}

.row-pomodoro-btn:hover {
  opacity: 0.85;
}

.row-pomodoro-done {
  color: var(--accent-sage);
  background: rgba(107, 139, 111, 0.08);
}

.row-pomodoro-na {
  color: var(--text-muted);
  background: rgba(0, 0, 0, 0.03);
}
</style>
