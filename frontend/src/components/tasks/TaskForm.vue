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

      <el-form-item label="截止时间">
        <el-date-picker
          v-model="formData.deadline"
          type="datetime"
          placeholder="选择截止时间"
          style="width: 100%"
        />
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
  deadline: null as Date | null,
  tags: [] as string[]
})

watch(() => props.task, (task) => {
  if (task) {
    formData.value = {
      title: task.title,
      description: task.description,
      quadrant: task.quadrant,
      estimated_time: task.estimated_time,
      deadline: task.deadline ? new Date(task.deadline) : null,
      tags: task.tags
    }
  } else {
    resetForm()
  }
})

function resetForm() {
  formData.value = {
    title: '',
    description: '',
    quadrant: 2,
    estimated_time: 0,
    deadline: null,
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
    deadline: formData.value.deadline?.toISOString() || null,
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
  background: rgba(196, 103, 61, 0.06);
  border: 1px solid rgba(196, 103, 61, 0.2);
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
  font-weight: 600;
  color: var(--accent-primary);
  font-size: 14px;
}

.recommendation-reason {
  margin: 0 0 14px 0;
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.6;
}
</style>
