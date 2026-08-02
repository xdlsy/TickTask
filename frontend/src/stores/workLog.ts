import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '@/api/client'
import { ElMessage } from 'element-plus'
import type {
  WorkLog, WorkReport, WorkReportType, TodayContext,
  StructuredWorkLog, SaveWorkLogInput,
  CreateQuickEntryInput, UpdateQuickEntryInput,
} from '@/types'

export type SelectedNode =
  | { kind: 'log'; date: string }
  | { kind: 'report'; type: WorkReportType; periodKey: string }

export const useWorkLogStore = defineStore('workLog', () => {
  const logs = ref<WorkLog[]>([])
  const currentLog = ref<WorkLog | null>(null)
  const todayContext = ref<TodayContext | null>(null)
  const reports = ref<Record<WorkReportType, WorkReport[]>>({
    weekly: [], monthly: [], halfyear: [], yearly: [],
  })
  const currentReport = ref<WorkReport | null>(null)
  const selected = ref<SelectedNode | null>(null)
  const loading = ref(false)

  const todayManualItems = computed(() => {
    const items = currentLog.value?.items ?? []
    return items
      .filter(i => i.source === 'manual')
      .sort((a, b) => (a.start_time ?? '').localeCompare(b.start_time ?? ''))
  })

  async function fetchInitialRange() {
    loading.value = true
    try {
      const today = new Date()
      const to = today.toISOString().slice(0, 10)
      const from = new Date(today.getTime() - 90 * 86400_000).toISOString().slice(0, 10)
      const { data } = await api.listWorkLogs(from, to)
      logs.value = data.logs || []
    } catch (e: any) {
      ElMessage.error('加载日报列表失败：' + (e?.message || ''))
    } finally {
      loading.value = false
    }
  }

  async function fetchLog(date: string) {
    try {
      const { data } = await api.getWorkLog(date)
      currentLog.value = data
    } catch (e: any) {
      if (e?.response?.status === 404) {
        currentLog.value = null
      } else {
        ElMessage.error('加载日报失败：' + (e?.message || ''))
      }
    }
  }

  async function fetchTodayContext(date?: string) {
    try {
      const { data } = await api.getTodayContext(date)
      todayContext.value = data
    } catch (e: any) {
      ElMessage.error('加载今日预填失败：' + (e?.message || ''))
    }
  }

  async function structureBrainDump(text: string): Promise<StructuredWorkLog | null> {
    if (!todayContext.value) return null
    try {
      const { data } = await api.structureBrainDump(text, todayContext.value)
      return data
    } catch (e: any) {
      ElMessage.error('AI 拆条失败：' + (e?.response?.data?.error || e?.message || ''))
      return null
    }
  }

  async function saveWorkLog(payload: SaveWorkLogInput): Promise<boolean> {
    try {
      try {
        await api.createWorkLog(payload)
        ElMessage.success('日报已保存')
      } catch (e: any) {
        if (e?.response?.status === 409) {
          await api.updateWorkLog(payload.date, payload)
          ElMessage.success('日报已更新')
        } else {
          throw e
        }
      }
      await fetchInitialRange()
      await fetchLog(payload.date)
      return true
    } catch (e: any) {
      ElMessage.error('保存失败：' + (e?.response?.data?.error || e?.message || ''))
      return false
    }
  }

  async function addQuickEntry(date: string, payload: CreateQuickEntryInput): Promise<boolean> {
    try {
      await api.appendWorkItem(date, payload)
      await fetchLog(date) // 只刷新 WorkLog，不调 fetchTodayContext
      return true
    } catch (e: any) {
      ElMessage.error('添加失败：' + (e?.response?.data?.error || e?.message || ''))
      return false
    }
  }

  async function updateQuickEntry(date: string, itemId: string, payload: UpdateQuickEntryInput): Promise<boolean> {
    try {
      await api.updateWorkItem(date, itemId, payload)
      await fetchLog(date)
      return true
    } catch (e: any) {
      ElMessage.error('更新失败：' + (e?.response?.data?.error || e?.message || ''))
      return false
    }
  }

  async function deleteQuickEntry(date: string, itemId: string): Promise<boolean> {
    try {
      await api.deleteWorkItem(date, itemId)
      await fetchLog(date)
      return true
    } catch (e: any) {
      ElMessage.error('删除失败：' + (e?.response?.data?.error || e?.message || ''))
      return false
    }
  }

  async function generateReport(type: WorkReportType, periodKey?: string, force = false) {
    try {
      const { data } = await api.generateWorkReport(type, periodKey, force)
      await fetchReports(type)
      ElMessage.success('报告已生成')
      return data
    } catch (e: any) {
      if (e?.response?.status === 409 && !force) {
        throw e
      }
      ElMessage.error('生成报告失败：' + (e?.response?.data?.error || e?.message || ''))
      throw e
    }
  }

  async function fetchReports(type: WorkReportType) {
    try {
      const { data } = await api.listWorkReports(type)
      reports.value[type] = data.reports || []
    } catch (e: any) {
      ElMessage.error('加载报告列表失败：' + (e?.message || ''))
    }
  }

  async function fetchReport(type: WorkReportType, periodKey: string) {
    try {
      const { data } = await api.getWorkReport(type, periodKey)
      currentReport.value = data
    } catch (e: any) {
      ElMessage.error('加载报告失败：' + (e?.message || ''))
    }
  }

  function selectNode(node: SelectedNode) {
    selected.value = node
    if (node.kind === 'log') {
      fetchLog(node.date)
    } else {
      fetchReport(node.type, node.periodKey)
    }
  }

  return {
    logs, currentLog, todayContext, reports, currentReport, selected, loading,
    todayManualItems,
    fetchInitialRange, fetchLog, fetchTodayContext, structureBrainDump,
    saveWorkLog, generateReport, fetchReports, fetchReport, selectNode,
    addQuickEntry, updateQuickEntry, deleteQuickEntry,
  }
})
