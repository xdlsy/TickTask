<template>
  <el-dialog
    :title="task ? '编辑任务' : '创建任务'"
    :model-value="visible"
    @update:model-value="$emit('close')"
    width="500px"
  >
    <el-form :model="formData" label-width="80px">
      <el-form-item label="标题" required>
        <div class="title-row">
          <el-input v-model="formData.title" placeholder="输入任务标题" style="flex: 1" />
          <el-button
            v-if="!task && aiStore.configured"
            type="primary"
            link
            :loading="aiClassifying"
            @click="getAIRecommendation"
          >
            AI 推荐
          </el-button>
        </div>
      </el-form-item>

      <el-form-item label="描述">
        <el-input
          v-model="formData.description"
          type="textarea"
          :rows="3"
          placeholder="输入任务描述（AI 推荐会更准确）"
        />
      </el-form-item>

      <el-form-item label="象限" required>
        <el-radio-group v-model="formData.quadrant">
          <el-radio-button :label="1">重要且紧急</el-radio-button>
          <el-radio-button :label="2">重要不紧急</el-radio-button>
        </el-radio-group>
        <br />
        <el-radio-group v-model="formData.quadrant" style="margin-top: 8px">
          <el-radio-button :label="3">紧急不重要</el-radio-button>
          <el-radio-button :label="4">不重要不紧急</el-radio-button>
        </el-radio-group>
      </el-form-item>

      <div v-if="aiRecommendation" class="ai-recommendation">
        <div class="recommendation-header">
          <span>AI 推荐象限</span>
          <el-tag :type="getQuadrantTagType(aiRecommendation.quadrant)">
            {{ getQuadrantName(aiRecommendation.quadrant) }}
          </el-tag>
        </div>
        <p class="recommendation-reason">{{ aiRecommendation.reason }}</p>
        <el-button type="primary" size="small" @click="applyRecommendation">
          采纳建议
        </el-button>
      </div>

      <el-form-item label="预估时间">
        <el-input-number v-model="formData.estimated_time" :min="0" :step="5" />
        <span style="margin-left: 8px; color: var(--text-secondary)">分钟</span>
      </el-form-item>

      <el-form-item label="偏好时段">
        <div style="display: flex; align-items: center; gap: 8px">
          <el-time-picker
            v-model="formData.preferred_start_time"
            format="HH:mm"
            value-format="HH:mm"
            placeholder="开始"
            style="width: 130px"
          />
          <span style="color: var(--text-muted)">—</span>
          <el-time-picker
            v-model="formData.preferred_end_time"
            format="HH:mm"
            value-format="HH:mm"
            placeholder="结束"
            style="width: 130px"
          />
        </div>
      </el-form-item>

      <el-form-item label="开始日期">
        <el-date-picker
          v-model="formData.start_date"
          type="date"
          placeholder="选择开始日期"
          style="width: 100%"
          value-format="YYYY-MM-DD"
        />
      </el-form-item>

      <el-form-item label="截止日期">
        <el-date-picker
          v-model="formData.due_date"
          type="date"
          placeholder="选择截止日期"
          style="width: 100%"
          value-format="YYYY-MM-DD"
        />
      </el-form-item>

      <el-form-item label="截止时间">
        <el-date-picker
          v-model="formData.deadline"
          type="datetime"
          placeholder="选择截止时间"
          style="width: 100%"
        />
      </el-form-item>

      <el-form-item label="重复任务">
        <el-switch v-model="formData.is_recurring" />
      </el-form-item>

      <el-form-item v-if="formData.is_recurring" label="重复模式">
        <el-select v-model="formData.recurrence_pattern" placeholder="选择重复模式" style="width: 100%">
          <el-option label="每日" value="daily" />
          <el-option label="每周" value="weekly" />
          <el-option label="每月" value="monthly" />
        </el-select>
      </el-form-item>

      <el-form-item label="标签">
        <el-input v-model="tagInput" @keyup.enter="addTag" placeholder="输入标签后按回车" />
        <div class="tags-list">
          <el-tag
            v-for="tag in formData.tags"
            :key="tag"
            closable
            @close="removeTag(tag)"
            style="margin-right: 8px; margin-top: 8px"
          >
            {{ tag }}
          </el-tag>
        </div>
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="$emit('close')">取消</el-button>
      <el-button type="primary" @click="onSave">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import type { Task, Quadrant, ClassificationResult } from '@/types'
import { QUADRANT_INFO } from '@/types'
import { useAIStore } from '@/stores/ai'

interface Props {
  visible: boolean
  task?: Task | null
}

const props = defineProps<Props>()

const emit = defineEmits<{
  close: []
  save: [data: any]
}>()

const aiStore = useAIStore()
const tagInput = ref('')
const aiClassifying = ref(false)
const aiRecommendation = ref<ClassificationResult | null>(null)

const formData = ref({
  title: '',
  description: '',
  quadrant: 2 as Quadrant,
  estimated_time: 0,
  preferred_start_time: null as string | null,
  preferred_end_time: null as string | null,
  start_date: null as string | null,
  due_date: null as string | null,
  deadline: null as Date | null,
  is_recurring: false,
  recurrence_pattern: '' as string,
  tags: [] as string[]
})

watch(() => props.task, (task) => {
  if (task) {
    formData.value = {
      title: task.title,
      description: task.description,
      quadrant: task.quadrant,
      estimated_time: task.estimated_time,
      preferred_start_time: task.preferred_start_time || null,
      preferred_end_time: task.preferred_end_time || null,
      start_date: task.start_date ? task.start_date.substring(0, 10) : null,
      due_date: task.due_date ? task.due_date.substring(0, 10) : null,
      deadline: task.deadline ? new Date(task.deadline) : null,
      is_recurring: task.is_recurring || false,
      recurrence_pattern: task.recurrence_pattern || '',
      tags: task.tags
    }
  } else {
    resetForm()
  }
}, { immediate: true })

function resetForm() {
  formData.value = {
    title: '',
    description: '',
    quadrant: 2,
    estimated_time: 0,
    preferred_start_time: null,
    preferred_end_time: null,
    start_date: null,
    due_date: null,
    deadline: null,
    is_recurring: false,
    recurrence_pattern: '',
    tags: []
  }
  aiRecommendation.value = null
}

function addTag() {
  const tag = tagInput.value.trim()
  if (tag && !formData.value.tags.includes(tag)) {
    formData.value.tags.push(tag)
  }
  tagInput.value = ''
}

function removeTag(tag: string) {
  const index = formData.value.tags.indexOf(tag)
  if (index > -1) {
    formData.value.tags.splice(index, 1)
  }
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

async function getAIRecommendation() {
  if (!formData.value.title.trim()) {
    ElMessage.warning('请先输入任务标题')
    return
  }

  aiClassifying.value = true
  try {
    const result = await aiStore.classifyTaskByText(
      formData.value.title,
      formData.value.description
    )
    if (result) {
      aiRecommendation.value = result
    }
  } catch (error) {
    ElMessage.error('AI 推荐失败')
  } finally {
    aiClassifying.value = false
  }
}

function applyRecommendation() {
  if (aiRecommendation.value) {
    formData.value.quadrant = aiRecommendation.value.quadrant as Quadrant
    ElMessage.success('已采纳 AI 推荐')
    aiRecommendation.value = null
  }
}

function onSave() {
  if (!formData.value.title.trim()) {
    return
  }

  const data = {
    title: formData.value.title,
    description: formData.value.description,
    quadrant: formData.value.quadrant,
    estimated_time: formData.value.estimated_time,
    preferred_start_time: formData.value.preferred_start_time || null,
    preferred_end_time: formData.value.preferred_end_time || null,
    start_date: formData.value.start_date ? formData.value.start_date + 'T00:00:00Z' : null,
    due_date: formData.value.due_date ? formData.value.due_date + 'T00:00:00Z' : null,
    deadline: formData.value.deadline?.toISOString() || null,
    is_recurring: formData.value.is_recurring,
    recurrence_pattern: formData.value.recurrence_pattern,
    tags: formData.value.tags
  }

  emit('save', data)
  resetForm()
}
</script>

<style scoped>
.tags-list {
  margin-top: 10px;
}

.title-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.ai-recommendation {
  background: var(--accent-fill);
  border: 1px solid rgba(230, 162, 60, 0.22);
  border-radius: var(--radius-md);
  padding: 16px;
  margin-bottom: 18px;
}

.recommendation-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}

.recommendation-header span:first-child {
  font-family: var(--font-mono);
  font-weight: 500;
  color: var(--accent-primary);
  font-size: 11px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.recommendation-reason {
  margin: 0 0 14px 0;
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.6;
}
</style>
