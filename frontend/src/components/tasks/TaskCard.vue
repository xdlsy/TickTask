<template>
  <div
    class="task-card"
    :class="{ 'task-completed': task.status === 'completed' }"
    draggable="true"
    @dragstart="$emit('drag-start', $event, task)"
    @click="showDetail"
  >
    <div class="task-header">
      <span class="task-title">{{ task.title }}</span>
      <el-dropdown @command="handleCommand" trigger="click" @click.stop>
        <el-icon class="more-icon" @click.stop><MoreFilled /></el-icon>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item v-if="task.status !== 'completed'" command="startTimer">
              🍅 开始番茄
            </el-dropdown-item>
            <el-dropdown-item command="edit">编辑</el-dropdown-item>
            <el-dropdown-item command="ai-classify" :disabled="aiStore.loading">
              <el-icon v-if="aiStore.loading" class="is-loading"><Loading /></el-icon>
              AI 智能分类
            </el-dropdown-item>
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
      <el-tag size="small" :type="task.status === 'completed' ? 'success' : 'info'">
        {{ statusLabel }}
      </el-tag>
    </div>

    <!-- AI 分类结果弹窗 -->
    <el-dialog
      v-model="showClassifyResult"
      title="AI 分类建议"
      width="400px"
      @click.stop
    >
      <div v-if="classifyResult" class="classify-result">
        <div class="result-item">
          <span class="label">推荐象限：</span>
          <el-tag :type="getQuadrantTagType(classifyResult.quadrant)">
            {{ getQuadrantName(classifyResult.quadrant) }}
          </el-tag>
        </div>
        <div class="result-item">
          <span class="label">重要性：</span>
          <span>{{ classifyResult.important ? '重要' : '不重要' }}</span>
        </div>
        <div class="result-item">
          <span class="label">紧急度：</span>
          <span>{{ classifyResult.urgent ? '紧急' : '不紧急' }}</span>
        </div>
        <div class="result-reason">
          <span class="label">理由：</span>
          <p>{{ classifyResult.reason }}</p>
        </div>
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
import { MoreFilled, Loading } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import type { Task, TaskStatus, ClassificationResult } from '@/types'
import { QUADRANT_INFO } from '@/types'
import { useAIStore } from '@/stores/ai'
import { useTaskStore } from '@/stores/task'
import { useTimerStore } from '@/stores/timer'
import { useRouter } from 'vue-router'

interface Props {
  task: Task
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'drag-start': [event: DragEvent, task: Task]
  'edit': [task: Task]
  'complete': [id: string]
  'delete': [id: string]
}>()

const aiStore = useAIStore()
const taskStore = useTaskStore()
const timerStore = useTimerStore()
const router = useRouter()

const showClassifyResult = ref(false)
const classifyResult = ref<ClassificationResult | null>(null)

const statusLabels: Record<TaskStatus, string> = {
  todo: '待办',
  in_progress: '进行中',
  completed: '已完成',
  cancelled: '已取消'
}

const statusLabel = computed(() => statusLabels[props.task.status])

function formatDate(dateStr: string): string {
  const date = new Date(dateStr)
  return `${date.getMonth() + 1}/${date.getDate()}`
}

function getQuadrantName(quadrant: number): string {
  return QUADRANT_INFO[quadrant as 1 | 2 | 3 | 4]?.name || `象限 ${quadrant}`
}

function getQuadrantTagType(quadrant: number): 'danger' | 'warning' | 'primary' | 'info' {
  const types: Record<number, 'danger' | 'warning' | 'primary' | 'info'> = {
    1: 'danger',
    2: 'warning',
    3: 'primary',
    4: 'info'
  }
  return types[quadrant] || 'info'
}

async function handleCommand(command: string) {
  switch (command) {
    case 'startTimer':
      await startTimerForTask()
      break
    case 'edit':
      emit('edit', props.task)
      break
    case 'complete':
      emit('complete', props.task.id)
      break
    case 'delete':
      emit('delete', props.task.id)
      break
    case 'ai-classify':
      await doClassify()
      break
  }
}

async function startTimerForTask() {
  try {
    await timerStore.createSession(props.task.id, 'work')
    ElMessage.success(`开始专注：${props.task.title}`)
    router.push('/timer')
  } catch (error) {
    ElMessage.error('启动计时器失败')
  }
}

async function doClassify() {
  if (!aiStore.configured) {
    ElMessage.warning('请先在设置中配置 AI API Key')
    router.push('/settings')
    return
  }

  try {
    const result = await aiStore.classifyTask(props.task.id)
    if (result) {
      classifyResult.value = result
      showClassifyResult.value = true
    }
  } catch (error) {
    ElMessage.error('AI 分类失败，请检查网络或 API 配置')
  }
}

async function applyClassification() {
  if (classifyResult.value && classifyResult.value.quadrant !== props.task.quadrant) {
    await taskStore.moveTask(props.task.id, classifyResult.value.quadrant as 1 | 2 | 3 | 4)
    ElMessage.success('已应用 AI 分类建议')
  }
  showClassifyResult.value = false
}

function showDetail() {
  emit('edit', props.task)
}
</script>

<style scoped>
.task-card {
  background: #fff;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  padding: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.task-card:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.task-completed {
  opacity: 0.6;
}

.task-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 8px;
}

.task-title {
  font-weight: 500;
  flex: 1;
}

.more-icon {
  color: #9ca3af;
  cursor: pointer;
}

.more-icon:hover {
  color: #3b82f6;
}

.task-description {
  color: #6b7280;
  font-size: 13px;
  margin-top: 8px;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.task-meta {
  display: flex;
  gap: 6px;
  margin-top: 8px;
  flex-wrap: wrap;
}

.classify-result {
  padding: 8px 0;
}

.result-item {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.result-item .label {
  color: #6b7280;
  min-width: 70px;
}

.result-reason {
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid #e5e7eb;
}

.result-reason .label {
  color: #6b7280;
  margin-bottom: 8px;
  display: block;
}

.result-reason p {
  margin: 0;
  line-height: 1.6;
  color: #1f2937;
}

.is-loading {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>