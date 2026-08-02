<template>
  <div class="work-item-form">
    <el-form :model="form" @submit.prevent="onSubmit">
      <div class="wif-required">
        <el-form-item label="日期">
          <el-date-picker
            data-test="date-input"
            v-model="form.date"
            type="date"
            value-format="YYYY-MM-DD"
            :disabled="mode === 'edit'"
            style="width: 140px"
          />
        </el-form-item>

        <el-form-item label="活动">
          <el-input
            data-test="activity-input"
            v-model="form.activity"
            placeholder="做了什么"
            style="width: 220px"
          />
        </el-form-item>

        <el-form-item label="时段">
          <el-time-select
            data-test="start-input"
            v-model="form.start_time"
            :max-time="form.end_time"
            placeholder="开始"
            start="00:00"
            step="00:30"
            end="23:30"
            :clearable="false"
            style="width: 100px"
          />
          <span style="margin: 0 4px">-</span>
          <el-time-select
            data-test="end-input"
            v-model="form.end_time"
            :min-time="form.start_time"
            placeholder="结束"
            start="00:30"
            step="00:30"
            end="24:00"
            :clearable="false"
            style="width: 100px"
          />
        </el-form-item>

        <el-form-item label="象限">
          <el-radio-group v-model="form.quadrant" data-test="quadrant-input">
            <el-radio-button
              v-for="q in [1, 2, 3, 4] as Quadrant[]"
              :key="q"
              :label="q"
            >
              Q{{ q }}
            </el-radio-button>
          </el-radio-group>
        </el-form-item>
      </div>

      <div class="wif-optional">
        <div class="wif-optional-title">补充详情（可选）</div>
        <div class="wif-optional-grid">
          <div class="wif-field">
            <span class="wif-label">内容</span>
            <el-input
              data-test="content-input"
              v-model="form.content"
              type="textarea"
              :rows="2"
              placeholder="一句话描述"
            />
          </div>
          <div class="wif-field">
            <span class="wif-label">解决了什么问题</span>
            <el-input
              data-test="problem-solved-input"
              v-model="form.problem_solved"
              type="textarea"
              :rows="2"
            />
          </div>
          <div class="wif-field">
            <span class="wif-label">已产生的结果</span>
            <el-input
              data-test="result-input"
              v-model="form.result"
              type="textarea"
              :rows="2"
            />
          </div>
          <div class="wif-field">
            <span class="wif-label">对后续的影响</span>
            <el-input
              data-test="impact-input"
              v-model="form.impact"
              type="textarea"
              :rows="2"
            />
          </div>
        </div>
      </div>

      <div class="wif-actions">
        <el-button
          v-if="mode === 'edit'"
          data-test="cancel-btn"
          @click="$emit('cancel')"
        >
          取消
        </el-button>
        <el-button
          data-test="submit-btn"
          type="primary"
          :loading="saving"
          @click="onSubmit"
        >
          {{ mode === 'edit' ? '保存' : '添加' }}
        </el-button>
      </div>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useWorkLogStore } from '@/stores/workLog'
import type { Quadrant } from '@/types'

interface InitialData {
  activity?: string
  start_time?: string
  end_time?: string
  quadrant?: Quadrant
  content?: string
  problem_solved?: string
  result?: string
  impact?: string
}

const props = withDefaults(defineProps<{
  date: string
  mode?: 'add' | 'edit'
  itemId?: string
  initial?: InitialData
}>(), {
  mode: 'add',
  itemId: '',
  initial: () => ({}) as InitialData,
})

const emit = defineEmits<{
  added: []
  saved: []
  cancel: []
}>()

const store = useWorkLogStore()

const form = reactive({
  date: props.date,
  activity: props.initial.activity ?? '',
  start_time: props.initial.start_time ?? '09:00',
  end_time: props.initial.end_time ?? '10:00',
  quadrant: props.initial.quadrant ?? (2 as Quadrant),
  content: props.initial.content ?? '',
  problem_solved: props.initial.problem_solved ?? '',
  result: props.initial.result ?? '',
  impact: props.initial.impact ?? '',
})

watch(() => props.date, (d) => { if (props.mode === 'add') form.date = d })

const saving = ref(false)

async function onSubmit() {
  if (!form.activity.trim()) {
    ElMessage.error('活动名称必填')
    return
  }
  if (!form.start_time || !form.end_time) {
    ElMessage.error('时间段必填')
    return
  }
  if (form.start_time >= form.end_time) {
    ElMessage.error('结束时间必须晚于开始时间')
    return
  }
  saving.value = true
  try {
    if (props.mode === 'edit') {
      const ok = await store.updateQuickEntry(form.date, props.itemId, {
        activity: form.activity,
        start_time: form.start_time,
        end_time: form.end_time,
        quadrant: form.quadrant,
        content: form.content,
        problem_solved: form.problem_solved,
        result: form.result,
        impact: form.impact,
      })
      if (ok) {
        ElMessage.success('已更新')
        emit('saved')
      }
    } else {
      const ok = await store.addQuickEntry(form.date, {
        activity: form.activity,
        start_time: form.start_time,
        end_time: form.end_time,
        quadrant: form.quadrant,
        content: form.content,
        problem_solved: form.problem_solved,
        result: form.result,
        impact: form.impact,
      })
      if (ok) {
        ElMessage.success('已添加')
        form.activity = ''
        form.content = ''
        form.problem_solved = ''
        form.result = ''
        form.impact = ''
        emit('added')
      }
    }
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.work-item-form {
  background: var(--bg-card, #FFFEFC);
  border: 1px solid var(--border-color, #e5e5e5);
  border-radius: var(--radius-md);
  padding: 16px 20px;
  margin-bottom: 16px;
}
.wif-required {
  display: grid;
  grid-template-columns: 140px 220px 1fr 220px;
  gap: 12px;
  align-items: end;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--border-color);
  margin-bottom: 14px;
}
.wif-required :deep(.el-form-item) {
  margin-bottom: 0;
}
.wif-optional {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  padding: 12px 14px;
}
.wif-optional-title {
  font-size: 11px;
  color: var(--text-muted);
  letter-spacing: 0.5px;
  text-transform: uppercase;
  margin-bottom: 10px;
  font-weight: 500;
}
.wif-optional-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px 14px;
}
.wif-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.wif-label {
  font-size: 11px;
  color: var(--text-muted);
  font-weight: 500;
}
.wif-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 14px;
}
</style>
