<template>
  <div class="today-context-card" v-if="context">
    <div class="ctx-header">
      <span class="ctx-title">今日预填</span>
      <span class="ctx-date">{{ context.date }}</span>
    </div>
    <div class="ctx-row" v-if="context.completed_tasks.length">
      <span class="ctx-label">已完成任务</span>
      <span class="ctx-value">{{ taskTitles }}</span>
    </div>
    <div class="ctx-row">
      <span class="ctx-label">番茄钟</span>
      <span class="ctx-value">
        {{ context.pomodoro_summary.count }} 个 · {{ context.pomodoro_summary.total_minutes }} 分钟
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { TodayContext } from '@/types'

const props = defineProps<{ context: TodayContext | null }>()

const taskTitles = computed(() =>
  (props.context?.completed_tasks || []).map(t => t.title).join('、'),
)
</script>

<style scoped>
.today-context-card {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: 16px 20px;
  margin-bottom: 16px;
}
.ctx-header {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin-bottom: 10px;
}
.ctx-title {
  font-family: var(--font-display);
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}
.ctx-date {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-muted);
}
.ctx-row {
  display: flex;
  gap: 12px;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text-secondary);
  padding: 4px 0;
}
.ctx-label {
  flex: 0 0 80px;
  color: var(--text-muted);
}
.ctx-value {
  flex: 1;
  color: var(--text-primary);
}
</style>
