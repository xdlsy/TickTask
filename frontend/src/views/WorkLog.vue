<template>
  <div class="work-log-page">
    <div class="page-header">
      <h1 class="page-title">工作日志</h1>
      <div class="page-actions">
        <button class="action-btn" @click="goToday">今日</button>
        <ReportActions @generate="onGenerateReport" />
      </div>
    </div>

    <div class="page-body">
      <Timeline
        :logs="store.logs"
        :reports="store.reports"
        :selected="store.selected"
        @select="store.selectNode"
      />

      <div class="detail-area">
        <!-- 日报视图 -->
        <template v-if="!store.selected || store.selected.kind === 'log'">
          <QuickEntryForm
            v-if="!editingItemId"
            :date="currentDate"
            mode="add"
            @added="onQuickAdded"
          />
          <QuickEntryForm
            v-else
            :key="editingItemId"
            :date="currentDate"
            mode="edit"
            :item-id="editingItemId"
            :initial="editingInitial"
            @saved="onEditSaved"
            @cancel="onEditCancel"
          />
          <TodayPanorama :date="currentDate" @edit="onPanoramaEdit" />

          <TodayContextCard :context="store.todayContext" />

          <BrainDumpInput
            :loading="structuring"
            @structure="onStructure"
          />

          <WorkItemList
            v-if="draftItems.length || draftSummary"
            :items="draftItems"
            :summary="draftSummary"
            @update:items="draftItems = $event"
            @update:summary="draftSummary = $event"
          />

          <div class="save-bar" v-if="draftItems.length">
            <button class="save-btn" :disabled="saving" @click="onSave">
              {{ saving ? '保存中…' : (isUpdate ? '更新今日日报' : '保存日报') }}
            </button>
          </div>
        </template>

        <!-- 报告视图 -->
        <template v-else>
          <ReportDetail :report="store.currentReport" />
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useWorkLogStore } from '@/stores/workLog'
import Timeline from '@/components/work-log/Timeline.vue'
import TodayContextCard from '@/components/work-log/TodayContextCard.vue'
import BrainDumpInput from '@/components/work-log/BrainDumpInput.vue'
import WorkItemList from '@/components/work-log/WorkItemList.vue'
import ReportActions from '@/components/work-log/ReportActions.vue'
import ReportDetail from '@/components/work-log/ReportDetail.vue'
import QuickEntryForm from '@/components/work-log/QuickEntryForm.vue'
import TodayPanorama from '@/components/work-log/TodayPanorama.vue'
import { ElMessageBox } from 'element-plus'
import type { StructuredWorkLog, SaveWorkLogInput, WorkReportType } from '@/types'

interface DraftItem {
  title: string
  content: string
  problem_solved: string
  result: string
  impact: string
}

const store = useWorkLogStore()

const structuring = ref(false)
const saving = ref(false)
const draftItems = ref<DraftItem[]>([])
const draftSummary = ref('')
const currentDate = ref(new Date().toISOString().slice(0, 10))
const editingItemId = ref<string | null>(null)

const isUpdate = computed(() =>
  store.logs.some(l => l.date === currentDate.value),
)

const editingInitial = computed(() => {
  if (!editingItemId.value || !store.currentLog) return {}
  const it = store.currentLog.items.find(i => i.id === editingItemId.value)
  if (!it) return {}
  return {
    activity: it.activity ?? '',
    start_time: it.start_time ?? '09:00',
    end_time: it.end_time ?? '10:00',
    quadrant: it.quadrant ?? 2,
  }
})

async function loadInitial() {
  await Promise.all([
    store.fetchInitialRange(),
    store.fetchTodayContext(currentDate.value),
  ])
}

async function onStructure(text: string) {
  structuring.value = true
  try {
    const out: StructuredWorkLog | null = await store.structureBrainDump(text)
    if (out) {
      draftItems.value = out.items.map(it => ({ ...it }))
      draftSummary.value = out.summary
    }
  } finally {
    structuring.value = false
  }
}

async function onSave() {
  saving.value = true
  try {
    const payload: SaveWorkLogInput = {
      date: currentDate.value,
      summary: draftSummary.value,
      raw_brain_dump: '',
      items: draftItems.value.map((it, idx) => ({ seq: idx + 1, ...it })),
    }
    await store.saveWorkLog(payload)
  } finally {
    saving.value = false
  }
}

function goToday() {
  currentDate.value = new Date().toISOString().slice(0, 10)
  store.selectNode({ kind: 'log', date: currentDate.value })
}

function onPanoramaEdit(itemId: string) {
  editingItemId.value = itemId
}

function onEditCancel() {
  editingItemId.value = null
}

function onEditSaved() {
  editingItemId.value = null
}

function onQuickAdded() {
  // store 已经 fetchLog 过，panorama 通过 computed 自动刷新
}

function computePeriodKey(type: WorkReportType): string {
  const now = new Date()
  if (type === 'weekly') {
    const tmp = new Date(now)
    tmp.setHours(0, 0, 0, 0)
    tmp.setDate(tmp.getDate() + 4 - (tmp.getDay() || 7))
    const yearStart = new Date(tmp.getFullYear(), 0, 1)
    const week = Math.ceil((((tmp.getTime() - yearStart.getTime()) / 86400000) + 1) / 7)
    return `${tmp.getFullYear()}-W${String(week).padStart(2, '0')}`
  }
  if (type === 'monthly') {
    return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
  }
  if (type === 'halfyear') {
    return `${now.getFullYear()}-H${now.getMonth() < 6 ? 1 : 2}`
  }
  return `${now.getFullYear()}`
}

function labelOf(type: WorkReportType): string {
  return { weekly: '周报', monthly: '月报', halfyear: '半年报', yearly: '年报' }[type]
}

async function onGenerateReport(type: WorkReportType) {
  const periodKey = computePeriodKey(type)
  try {
    await store.generateReport(type, periodKey, false)
    store.selectNode({ kind: 'report', type, periodKey })
  } catch (e: any) {
    if (e?.response?.status === 409) {
      try {
        await ElMessageBox.confirm(
          `${labelOf(type)} ${periodKey} 已存在，是否覆盖重新生成？`,
          '覆盖确认',
          { confirmButtonText: '覆盖', cancelButtonText: '取消' },
        )
        await store.generateReport(type, periodKey, true)
        store.selectNode({ kind: 'report', type, periodKey })
      } catch {
        // user cancelled
      }
    }
  }
}

watch(currentDate, (d) => {
  store.fetchTodayContext(d)
})

watch(() => store.selected, (s) => {
  if (s?.kind === 'log' && s.date !== currentDate.value) {
    currentDate.value = s.date
  }
})

watch(() => store.currentLog, (log) => {
  if (!log || log.date !== currentDate.value) return
  draftItems.value = log.items.map(it => ({
    title: it.title,
    content: it.content,
    problem_solved: it.problem_solved,
    result: it.result,
    impact: it.impact,
  }))
  draftSummary.value = log.summary
})

onMounted(async () => {
  await loadInitial()
  await Promise.all([
    store.fetchReports('weekly'),
    store.fetchReports('monthly'),
    store.fetchReports('halfyear'),
    store.fetchReports('yearly'),
  ])
})
</script>

<style scoped>
.work-log-page {
  height: 100%;
  display: flex;
  flex-direction: column;
}
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}
.page-title {
  font-family: var(--font-display);
  font-size: 28px;
  font-weight: 600;
  color: var(--text-primary);
  letter-spacing: -0.5px;
}
.action-btn {
  background: transparent;
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
  padding: 6px 14px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 13px;
}
.action-btn:hover {
  border-color: var(--accent-primary);
  color: var(--accent-primary);
}
.page-body {
  flex: 1;
  display: flex;
  border-top: 1px solid var(--border-color);
  margin: 0 -40px -40px -40px;
  padding: 0;
}
.detail-area {
  flex: 1;
  padding: 24px 32px;
  overflow-y: auto;
}
.save-bar {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
.save-btn {
  background: var(--accent-primary);
  color: white;
  border: none;
  border-radius: var(--radius-sm);
  padding: 8px 20px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
}
.save-btn:hover:not(:disabled) {
  background: var(--accent-secondary);
}
.save-btn:disabled {
  background: var(--text-muted);
  cursor: not-allowed;
}
.report-placeholder {
  padding: 40px;
  color: var(--text-muted);
  font-style: italic;
}
</style>
