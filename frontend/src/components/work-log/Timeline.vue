<template>
  <div class="timeline">
    <div class="tl-section" v-if="logs.length">
      <div class="tl-section-title">日报</div>
      <div class="tl-list">
        <div
          v-for="log in logs"
          :key="log.date"
          class="tl-item"
          :class="{ active: selected?.kind === 'log' && selected.date === log.date }"
          @click="$emit('select', { kind: 'log', date: log.date })"
        >
          <span class="tl-dot"></span>
          <span class="tl-date">{{ formatDate(log.date) }}</span>
          <span class="tl-badge" v-if="isToday(log.date)">今</span>
        </div>
      </div>
    </div>

    <div class="tl-section" v-for="t in reportTypes" :key="t">
      <div class="tl-section-title">{{ reportLabel[t] }}</div>
      <div class="tl-list">
        <div
          v-for="r in reports[t]"
          :key="r.period_key"
          class="tl-item"
          :class="{ active: selected?.kind === 'report' && selected.type === t && selected.periodKey === r.period_key }"
          @click="$emit('select', { kind: 'report', type: t, periodKey: r.period_key })"
        >
          <span class="tl-dot"></span>
          <span class="tl-date">{{ r.period_key }}</span>
        </div>
        <div v-if="!reports[t] || reports[t].length === 0" class="tl-empty">—</div>
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
  padding: 24px 18px;
  overflow-y: auto;
  height: 100%;
  background: transparent;
}
.tl-section {
  margin-bottom: 26px;
}
.tl-section-title {
  font-family: var(--font-mono);
  font-size: 10px;
  font-weight: 500;
  color: var(--text-muted);
  letter-spacing: 0.22em;
  text-transform: uppercase;
  margin-bottom: 12px;
  padding-left: 2px;
}

/* 时间轴列表:左侧细发丝轨道 + 节点圆点 */
.tl-list {
  position: relative;
  padding-left: 4px;
}
.tl-list::before {
  content: '';
  position: absolute;
  left: 7px;
  top: 4px;
  bottom: 4px;
  width: 1px;
  background: var(--border-color);
}
.tl-item {
  position: relative;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 8px 6px 16px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-secondary);
  transition: color var(--transition-fast), background var(--transition-fast), padding-left var(--transition-fast);
}
.tl-dot {
  position: absolute;
  left: 4px;
  top: 50%;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--bg-secondary);
  border: 1px solid var(--border-strong);
  transform: translateY(-50%);
  z-index: 1;
  transition: all var(--transition-fast);
}
.tl-item:hover {
  background: rgba(239, 231, 215, 0.04);
  color: var(--text-primary);
  padding-left: 19px;
}
.tl-item:hover .tl-dot {
  border-color: var(--accent-primary);
}
.tl-item.active {
  background: var(--accent-fill);
  color: var(--accent-primary);
  font-weight: 600;
}
.tl-item.active .tl-dot {
  background: var(--accent-primary);
  border-color: var(--accent-primary);
  box-shadow: 0 0 10px rgba(230, 162, 60, 0.6);
}
.tl-date {
  flex: 1;
}
.tl-badge {
  background: var(--accent-primary);
  color: var(--bg-primary);
  font-size: 9.5px;
  padding: 1px 7px;
  border-radius: 999px;
  font-weight: 600;
  letter-spacing: 0.04em;
}
.tl-empty {
  padding: 4px 8px 4px 16px;
  color: var(--text-muted);
  opacity: 0.4;
  font-family: var(--font-mono);
  font-size: 12px;
}
</style>
