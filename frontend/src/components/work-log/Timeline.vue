<template>
  <div class="timeline">
    <div class="tl-section" v-if="logs.length">
      <div class="tl-section-title">日报</div>
      <div
        v-for="log in logs"
        :key="log.date"
        class="tl-item"
        :class="{ active: selected?.kind === 'log' && selected.date === log.date }"
        @click="$emit('select', { kind: 'log', date: log.date })"
      >
        <span class="tl-date">{{ formatDate(log.date) }}</span>
        <span class="tl-badge" v-if="isToday(log.date)">今</span>
      </div>
    </div>

    <div class="tl-section" v-for="t in reportTypes" :key="t">
      <div class="tl-section-title">{{ reportLabel[t] }}</div>
      <div
        v-for="r in reports[t]"
        :key="r.period_key"
        class="tl-item"
        :class="{ active: selected?.kind === 'report' && selected.type === t && selected.periodKey === r.period_key }"
        @click="$emit('select', { kind: 'report', type: t, periodKey: r.period_key })"
      >
        <span class="tl-date">{{ r.period_key }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { WorkLog, WorkReport, WorkReportType } from '@/types'
import type { SelectedNode } from '@/stores/workLog'

defineProps<{
  logs: WorkLog[]
  reports: Record<WorkReportType, WorkReport[]>
  selected: SelectedNode | null
}>()

defineEmits<{ (e: 'select', node: SelectedNode): void }>()

const reportTypes: WorkReportType[] = ['weekly', 'monthly', 'halfyear', 'yearly']
const reportLabel: Record<WorkReportType, string> = {
  weekly: '周报', monthly: '月报', halfyear: '半年报', yearly: '年报',
}

function formatDate(d: string): string {
  const dt = new Date(d)
  const weekdays = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
  return `${dt.getMonth() + 1}/${dt.getDate()} ${weekdays[dt.getDay()]}`
}

function isToday(d: string): boolean {
  return d === new Date().toISOString().slice(0, 10)
}
</script>

<style scoped>
.timeline {
  width: 240px;
  border-right: 1px solid var(--border-color);
  padding: 20px 16px;
  overflow-y: auto;
  height: 100%;
}
.tl-section {
  margin-bottom: 24px;
}
.tl-section-title {
  font-family: var(--font-display);
  font-size: 11px;
  font-weight: 600;
  color: var(--text-muted);
  letter-spacing: 0.5px;
  text-transform: uppercase;
  margin-bottom: 8px;
  padding-left: 8px;
}
.tl-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 13px;
  color: var(--text-secondary);
  transition: all var(--transition-fast);
}
.tl-item:hover {
  background: rgba(0, 0, 0, 0.04);
  color: var(--text-primary);
}
.tl-item.active {
  background: rgba(184, 69, 44, 0.06);
  color: var(--accent-primary);
  font-weight: 500;
}
.tl-date {
  flex: 1;
}
.tl-badge {
  background: var(--accent-primary);
  color: white;
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 8px;
  font-weight: 600;
}
</style>
