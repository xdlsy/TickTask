<template>
  <el-dialog
    :model-value="visible"
    :title="isEdit ? '编辑日程' : '新建日程'"
    @close="$emit('close')"
    width="500px"
  >
    <el-form :model="form" label-width="80px">
      <el-form-item label="标题" required>
        <el-input v-model="form.title" placeholder="请输入日程标题" />
      </el-form-item>

      <el-form-item label="日期">
        <el-date-picker
          v-model="form.date"
          type="date"
          placeholder="选择日期"
          style="width: 100%"
        />
      </el-form-item>

      <el-form-item label="时间">
        <div class="time-range">
          <el-time-select
            v-model="form.start_time"
            placeholder="开始时间"
            start="00:00"
            step="00:30"
            end="23:30"
            :clearable="false"
            :max-time="form.end_time"
            class="time-select"
            @change="onStartTimeChange"
          />
          <span class="time-separator">至</span>
          <el-time-select
            v-model="form.end_time"
            placeholder="结束时间"
            start="00:00"
            step="00:30"
            end="24:00"
            :clearable="false"
            :min-time="form.start_time"
            class="time-select"
            @change="onEndTimeChange"
          />
        </div>
      </el-form-item>

      <el-form-item label="类型">
        <el-select v-model="form.type" style="width: 100%">
          <el-option label="任务" value="task" />
          <el-option label="番茄钟" value="pomodoro" />
          <el-option label="休息" value="break" />
          <el-option label="自定义" value="custom" />
        </el-select>
      </el-form-item>

      <el-form-item label="颜色">
        <div class="color-picker">
          <div
            v-for="color in colors"
            :key="color"
            class="color-option"
            :style="{ backgroundColor: color }"
            :class="{ active: form.color === color }"
            @click="form.color = color"
          ></div>
        </div>
      </el-form-item>

      <el-form-item label="描述">
        <el-input
          v-model="form.description"
          type="textarea"
          :rows="3"
          placeholder="日程描述（可选）"
        />
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="$emit('close')">取消</el-button>
      <el-button v-if="isEdit" type="danger" @click="handleDelete">删除</el-button>
      <el-button type="primary" @click="handleSave">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { ElMessage } from 'element-plus'
import type { ScheduleEvent, CreateScheduleDTO, UpdateScheduleDTO, ScheduleType } from '@/types'

const props = defineProps<{
  visible: boolean
  event?: ScheduleEvent | null
  defaultDate?: string
  defaultHour?: number
}>()

const emit = defineEmits<{
  close: []
  save: [data: CreateScheduleDTO]
  update: [id: string, data: UpdateScheduleDTO]
  delete: [id: string]
}>()

const isEdit = computed(() => !!props.event)

const colors = [
  '#ef4444', '#f97316', '#f59e0b', '#84cc16',
  '#22c55e', '#14b8a6', '#06b6d4', '#3b82f6',
  '#6366f1', '#8b5cf6', '#a855f7', '#ec4899'
]

const form = ref({
  title: '',
  date: new Date(),
  start_time: '09:00',
  end_time: '10:00',
  type: 'task' as ScheduleType,
  color: '#3b82f6',
  description: ''
})

// 重置表单到初始状态
function resetForm() {
  form.value = {
    title: '',
    date: new Date(),
    start_time: '09:00',
    end_time: '10:00',
    type: 'task' as ScheduleType,
    color: '#3b82f6',
    description: ''
  }
}

// 初始化表单（编辑或新建模式）
function initForm() {
  if (props.event) {
    // 编辑模式：从事件读取数据
    form.value.title = props.event.title
    form.value.date = new Date(props.event.start)
    form.value.start_time = formatTimeForInput(props.event.start)
    form.value.end_time = formatTimeForInput(props.event.end)
    form.value.type = props.event.type
    form.value.color = props.event.color
    form.value.description = ''
  } else {
    // 新建模式：使用默认值
    resetForm()
    if (props.defaultDate) {
      form.value.date = new Date(props.defaultDate)
    }
    if (props.defaultHour !== undefined) {
      form.value.start_time = `${props.defaultHour.toString().padStart(2, '0')}:00`
      form.value.end_time = `${(props.defaultHour + 1).toString().padStart(2, '0')}:00`
    }
  }
}

watch(() => props.visible, (visible) => {
  if (visible) {
    initForm()
  }
}, { immediate: true })

function formatTimeForInput(dateStr: string): string {
  const date = new Date(dateStr)
  return `${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}`
}

function onStartTimeChange(val: string) {
  // 确保开始时间值被正确保存
  form.value.start_time = val
}

function onEndTimeChange(val: string) {
  // 确保结束时间值被正确保存
  form.value.end_time = val
}

function combineDateTime(date: Date, time: string): string {
  const [hours, minutes] = time.split(':').map(Number)
  const result = new Date(date)
  result.setHours(hours, minutes, 0, 0)
  return result.toISOString()
}

function handleSave() {
  if (!form.value.title.trim()) {
    ElMessage.warning('请输入日程标题')
    return
  }

  const startTime = combineDateTime(form.value.date, form.value.start_time)
  const endTime = combineDateTime(form.value.date, form.value.end_time)

  if (props.event) {
    emit('update', props.event.id, {
      title: form.value.title,
      start_time: startTime,
      end_time: endTime,
      color: form.value.color,
      description: form.value.description
    })
  } else {
    emit('save', {
      title: form.value.title,
      start_time: startTime,
      end_time: endTime,
      type: form.value.type,
      color: form.value.color,
      description: form.value.description
    })
  }
}

function handleDelete() {
  if (props.event) {
    emit('delete', props.event.id)
  }
}
</script>

<style scoped>
.time-range {
  display: flex;
  align-items: center;
  gap: 12px;
}

.time-select {
  width: 140px;
}

.time-separator {
  color: #6b7280;
}

.color-picker {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.color-option {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
  border: 2px solid transparent;
}

.color-option:hover {
  transform: scale(1.1);
}

.color-option.active {
  border-color: #1f2937;
  box-shadow: 0 0 0 2px #fff, 0 0 0 4px #1f2937;
}
</style>