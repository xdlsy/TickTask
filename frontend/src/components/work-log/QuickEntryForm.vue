<template>
  <div class="quick-entry-form">
    <el-form inline :model="form" @submit.prevent="onSubmit">
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
          style="width: 200px"
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
          style="width: 110px"
        />
        <span style="margin: 0 6px">-</span>
        <el-time-select
          data-test="end-input"
          v-model="form.end_time"
          :min-time="form.start_time"
          placeholder="结束"
          start="00:30"
          step="00:30"
          end="24:00"
          :clearable="false"
          style="width: 110px"
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

      <el-form-item>
        <el-button
          data-test="submit-btn"
          type="primary"
          :loading="saving"
          @click="onSubmit"
        >
          {{ mode === 'edit' ? '保存' : '添加' }}
        </el-button>
        <el-button
          v-if="mode === 'edit'"
          data-test="cancel-btn"
          @click="$emit('cancel')"
        >
          取消
        </el-button>
      </el-form-item>
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
      })
      if (ok) {
        ElMessage.success('已添加')
        form.activity = ''
        emit('added')
      }
    }
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.quick-entry-form {
  background: var(--bg-card, #FFFEFC);
  border: 1px solid var(--border-color, #e5e5e5);
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 16px;
}
</style>
