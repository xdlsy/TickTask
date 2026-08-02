<template>
  <div class="report-detail" v-if="report">
    <div class="rd-header">
      <h2 class="rd-title">{{ periodLabel[report.type] }} {{ report.period_key }}</h2>
      <span class="rd-range">{{ report.start_date }} ~ {{ report.end_date }}</span>
    </div>

    <div v-if="report.missing_days" class="rd-missing">
      ⚠️ 缺失天：{{ report.missing_days }}
    </div>

    <div class="rd-section" v-for="f in fields" :key="f.key">
      <div class="rd-section-title">{{ f.label }}</div>
      <div class="rd-section-body">{{ summary[f.key] || '（待补充）' }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { WorkReport, WorkReportType, ReportSummary } from '@/types'

const props = defineProps<{ report: WorkReport | null }>()

const periodLabel: Record<WorkReportType, string> = {
  weekly: '周报', monthly: '月报', halfyear: '半年报', yearly: '年报',
}

const fields: Array<{ key: keyof ReportSummary; label: string }> = [
  { key: 'core_work', label: '核心工作 / 重大成果' },
  { key: 'main_progress', label: '主要进展 / 趋势' },
  { key: 'open_issues', label: '遗留问题 / 关键问题' },
  { key: 'next_focus', label: '下阶段关注' },
]

const emptySummary: ReportSummary = {
  core_work: '', main_progress: '', open_issues: '', next_focus: '',
}

const summary = computed<ReportSummary>(() => {
  if (!props.report?.summary_json) return emptySummary
  try {
    return JSON.parse(props.report.summary_json)
  } catch {
    return emptySummary
  }
})
</script>

<style scoped>
.report-detail {
  max-width: 720px;
}
.rd-header {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border-color);
}
.rd-title {
  font-family: var(--font-display);
  font-size: 24px;
  font-weight: 600;
  color: var(--text-primary);
}
.rd-range {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-muted);
}
.rd-missing {
  background: rgba(184, 69, 44, 0.06);
  color: var(--accent-primary);
  padding: 8px 12px;
  border-radius: var(--radius-sm);
  font-size: 12px;
  margin-bottom: 20px;
}
.rd-section {
  margin-bottom: 24px;
}
.rd-section-title {
  font-family: var(--font-display);
  font-size: 12px;
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 8px;
}
.rd-section-body {
  font-size: 14px;
  line-height: 1.7;
  color: var(--text-primary);
  white-space: pre-wrap;
}
</style>
