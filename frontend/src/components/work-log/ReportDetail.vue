<template>
  <div class="report-detail" v-if="report">
    <div class="rd-header">
      <div class="rd-header-left">
        <span class="rd-eyebrow">Period Report</span>
        <h2 class="rd-title">{{ periodLabel[report.type] }} <em>{{ report.period_key }}</em></h2>
      </div>
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
  align-items: flex-end;
  gap: 20px;
  margin-bottom: 28px;
  padding-bottom: 20px;
  border-bottom: 1px solid var(--border-color);
}
.rd-header-left {
  display: flex;
  flex-direction: column;
}
.rd-eyebrow {
  display: inline-flex;
  align-items: center;
  gap: 11px;
  font-family: var(--font-mono);
  font-size: 10.5px;
  font-weight: 500;
  letter-spacing: 0.28em;
  text-transform: uppercase;
  color: var(--accent-primary);
  margin-bottom: 10px;
}
.rd-eyebrow::before {
  content: '';
  width: 26px;
  height: 1px;
  background: var(--accent-primary);
  opacity: 0.6;
}
.rd-title {
  font-family: var(--font-display);
  font-variation-settings: 'opsz' 144;
  font-size: 34px;
  font-weight: 380;
  color: var(--text-primary);
  margin: 0;
  letter-spacing: -0.03em;
  line-height: 1.04;
}
.rd-title em {
  font-style: italic;
  font-weight: 360;
  color: var(--text-secondary);
  margin-left: 6px;
}
.rd-range {
  font-family: var(--font-mono);
  font-size: 11.5px;
  color: var(--text-muted);
  letter-spacing: 0.04em;
  white-space: nowrap;
  flex-shrink: 0;
  padding-bottom: 4px;
}
.rd-missing {
  background: var(--crimson-fill);
  border: 1px solid rgba(216, 111, 84, 0.3);
  color: var(--accent-crimson);
  padding: 10px 14px;
  border-radius: var(--radius-md);
  font-family: var(--font-mono);
  font-size: 12px;
  letter-spacing: 0.02em;
  margin-bottom: 24px;
}
.rd-section {
  margin-bottom: 28px;
}
.rd-section-title {
  font-family: var(--font-mono);
  font-size: 10.5px;
  font-weight: 500;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.2em;
  margin-bottom: 10px;
}
.rd-section-body {
  font-family: var(--font-body);
  font-size: 14px;
  line-height: 1.75;
  color: var(--text-primary);
  white-space: pre-wrap;
}
</style>
