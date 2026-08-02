import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

vi.mock('@/api/client', () => ({
  api: {
    listWorkLogs: vi.fn(),
    getWorkLog: vi.fn(),
    getTodayContext: vi.fn(),
    structureBrainDump: vi.fn(),
    createWorkLog: vi.fn(),
    updateWorkLog: vi.fn(),
    generateWorkReport: vi.fn(),
    listWorkReports: vi.fn(),
    getWorkReport: vi.fn(),
    appendWorkItem: vi.fn(),
    updateWorkItem: vi.fn(),
    deleteWorkItem: vi.fn(),
    updateWorkLogSummary: vi.fn(),
  },
}))

vi.mock('element-plus', () => ({
  ElMessage: { error: vi.fn(), success: vi.fn(), warning: vi.fn() },
}))

import { useWorkLogStore } from './workLog'
import { api } from '@/api/client'
import { ElMessage } from 'element-plus'

const mockApi = api as any
const mockEl = ElMessage as any

const fakeContext = {
  date: '2026-08-02',
  completed_tasks: [],
  pomodoro_sessions: [],
  pomodoro_summary: { count: 0, total_minutes: 0 },
}

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
  mockApi.listWorkLogs.mockResolvedValue({ data: { logs: [] } })
  mockApi.getTodayContext.mockResolvedValue({ data: fakeContext })
  mockApi.structureBrainDump.mockResolvedValue({ data: { items: [], summary: '' } })
  mockApi.createWorkLog.mockResolvedValue({})
  mockApi.updateWorkLog.mockResolvedValue({})
  mockApi.getWorkLog.mockResolvedValue({ data: { id: 'wl-1', date: '2026-08-02', items: [] } })
  mockApi.appendWorkItem.mockResolvedValue({ data: {} })
  mockApi.updateWorkItem.mockResolvedValue({ data: { ok: true } })
  mockApi.deleteWorkItem.mockResolvedValue({ data: { ok: true } })
  mockApi.updateWorkLogSummary.mockResolvedValue({ data: { ok: true } })
})

describe('WorkLog Store', () => {
  it('initializes with empty state', () => {
    const store = useWorkLogStore()
    expect(store.logs).toEqual([])
    expect(store.currentLog).toBeNull()
    expect(store.todayContext).toBeNull()
    expect(store.selected).toBeNull()
  })

  it('fetchInitialRange loads logs', async () => {
    mockApi.listWorkLogs.mockResolvedValue({
      data: { logs: [{ id: '1', date: '2026-08-02', items: [] }] },
    })
    const store = useWorkLogStore()
    await store.fetchInitialRange()
    expect(store.logs).toHaveLength(1)
    expect(store.loading).toBe(false)
  })

  it('fetchInitialRange surfaces error on failure', async () => {
    mockApi.listWorkLogs.mockRejectedValue(new Error('network'))
    const store = useWorkLogStore()
    await store.fetchInitialRange()
    expect(mockEl.error).toHaveBeenCalled()
    expect(store.loading).toBe(false)
  })

  it('fetchLog 404 sets currentLog null', async () => {
    mockApi.getWorkLog.mockRejectedValue({ response: { status: 404 } })
    const store = useWorkLogStore()
    await store.fetchLog('2026-08-02')
    expect(store.currentLog).toBeNull()
  })

  it('fetchLog non-404 surfaces error', async () => {
    mockApi.getWorkLog.mockRejectedValue({ response: { status: 500 }, message: 'boom' })
    const store = useWorkLogStore()
    await store.fetchLog('2026-08-02')
    expect(mockEl.error).toHaveBeenCalled()
  })

  it('fetchTodayContext populates todayContext', async () => {
    const store = useWorkLogStore()
    await store.fetchTodayContext()
    expect(store.todayContext?.date).toBe('2026-08-02')
  })

  it('structureBrainDump returns null without todayContext', async () => {
    const store = useWorkLogStore()
    const out = await store.structureBrainDump('text')
    expect(out).toBeNull()
  })

  it('structureBrainDump returns structured data', async () => {
    const store = useWorkLogStore()
    await store.fetchTodayContext()
    mockApi.structureBrainDump.mockResolvedValue({
      data: { items: [{ title: 'T1', content: 'c1', problem_solved: '', result: '', impact: '' }], summary: 's' },
    })
    const out = await store.structureBrainDump('xxx')
    expect(out?.items).toHaveLength(1)
    expect(out?.summary).toBe('s')
  })

  it('structureBrainDump surfaces error on AI fail', async () => {
    mockApi.structureBrainDump.mockRejectedValue({ response: { data: { error: 'bad' } } })
    const store = useWorkLogStore()
    await store.fetchTodayContext()
    const out = await store.structureBrainDump('xxx')
    expect(out).toBeNull()
    expect(mockEl.error).toHaveBeenCalled()
  })

  it('saveWorkLog POST then PUT on 409', async () => {
    mockApi.createWorkLog.mockRejectedValue({ response: { status: 409 } })
    const store = useWorkLogStore()
    const ok = await store.saveWorkLog({ date: '2026-08-02', summary: '', raw_brain_dump: '', items: [] })
    expect(ok).toBe(true)
    expect(mockApi.updateWorkLog).toHaveBeenCalledWith('2026-08-02', expect.any(Object))
  })

  it('saveWorkLog POST success does not call PUT', async () => {
    const store = useWorkLogStore()
    const ok = await store.saveWorkLog({ date: '2026-08-02', summary: '', raw_brain_dump: '', items: [] })
    expect(ok).toBe(true)
    expect(mockApi.updateWorkLog).not.toHaveBeenCalled()
  })

  it('saveWorkLog returns false on non-409 error', async () => {
    mockApi.createWorkLog.mockRejectedValue({ response: { status: 500, data: { error: 'db' } } })
    const store = useWorkLogStore()
    const ok = await store.saveWorkLog({ date: '2026-08-02', summary: '', raw_brain_dump: '', items: [] })
    expect(ok).toBe(false)
    expect(mockEl.error).toHaveBeenCalled()
  })

  it('generateReport throws on 409 without force', async () => {
    mockApi.generateWorkReport.mockRejectedValue({ response: { status: 409 } })
    const store = useWorkLogStore()
    await expect(store.generateReport('weekly', '2026-W31', false)).rejects.toBeDefined()
  })

  it('generateReport returns data on success', async () => {
    mockApi.generateWorkReport.mockResolvedValue({
      data: { id: 'r1', type: 'weekly', period_key: '2026-W31' },
    })
    const store = useWorkLogStore()
    const r = await store.generateReport('weekly', '2026-W31', false)
    expect(r?.id).toBe('r1')
    expect(mockApi.listWorkReports).toHaveBeenCalledWith('weekly')
  })

  it('selectNode with log kind fetches log', async () => {
    const store = useWorkLogStore()
    store.selectNode({ kind: 'log', date: '2026-08-02' })
    expect(store.selected).toEqual({ kind: 'log', date: '2026-08-02' })
    expect(mockApi.getWorkLog).toHaveBeenCalledWith('2026-08-02')
  })

  it('selectNode with report kind fetches report', async () => {
    mockApi.getWorkReport.mockResolvedValue({ data: { id: 'r1', type: 'weekly', period_key: '2026-W31' } })
    const store = useWorkLogStore()
    store.selectNode({ kind: 'report', type: 'weekly', periodKey: '2026-W31' })
    expect(store.selected).toEqual({ kind: 'report', type: 'weekly', periodKey: '2026-W31' })
    expect(mockApi.getWorkReport).toHaveBeenCalledWith('weekly', '2026-W31')
  })
})

describe('WorkLog Store - quick entry', () => {
  beforeEach(() => {
    // already covered by top-level beforeEach, but be explicit
    mockApi.getWorkLog.mockResolvedValue({ data: { id: 'wl-1', date: '2026-08-02', items: [] } })
  })

  it('todayManualItems filters source=manual and sorts by start_time', () => {
    setActivePinia(createPinia())
    const store = useWorkLogStore()
    store.currentLog = {
      id: 'wl-1', date: '2026-08-02', summary: '', raw_brain_dump: '',
      created_at: '', updated_at: '',
      items: [
        { id: '1', work_log_id: 'wl-1', seq: 1, title: '', content: '', problem_solved: '', result: '', impact: '', source: 'ai', activity: null, start_time: null, end_time: null, quadrant: null },
        { id: '2', work_log_id: 'wl-1', seq: 2, title: '', content: '', problem_solved: '', result: '', impact: '', source: 'manual', activity: 'b', start_time: '10:00', end_time: '11:00', quadrant: 2 },
        { id: '3', work_log_id: 'wl-1', seq: 3, title: '', content: '', problem_solved: '', result: '', impact: '', source: 'manual', activity: 'a', start_time: '09:00', end_time: '10:00', quadrant: 1 },
      ],
    } as any
    const got = store.todayManualItems
    expect(got).toHaveLength(2)
    expect(got[0].id).toBe('3')  // 09:00 排前
    expect(got[1].id).toBe('2')
  })

  it('addQuickEntry calls api and refetches currentLog', async () => {
    setActivePinia(createPinia())
    mockApi.appendWorkItem.mockResolvedValueOnce({ data: {} })
    mockApi.getWorkLog.mockResolvedValueOnce({ data: { id: 'wl-1', date: '2026-08-02', items: [] } })
    const store = useWorkLogStore()
    await store.addQuickEntry('2026-08-02', {
      activity: '晨会', start_time: '09:00', end_time: '10:00', quadrant: 1,
    })
    expect(mockApi.appendWorkItem).toHaveBeenCalledWith('2026-08-02', {
      activity: '晨会', start_time: '09:00', end_time: '10:00', quadrant: 1,
    })
    expect(mockApi.getWorkLog).toHaveBeenCalledWith('2026-08-02')
  })

  it('updateQuickEntry calls api and refetches', async () => {
    setActivePinia(createPinia())
    mockApi.updateWorkItem.mockResolvedValueOnce({ data: { ok: true } })
    mockApi.getWorkLog.mockResolvedValueOnce({ data: { id: 'wl-1', date: '2026-08-02', items: [] } })
    const store = useWorkLogStore()
    await store.updateQuickEntry('2026-08-02', 'item-1', { activity: 'new' })
    expect(mockApi.updateWorkItem).toHaveBeenCalledWith('2026-08-02', 'item-1', { activity: 'new' })
  })

  it('deleteQuickEntry calls api and refetches', async () => {
    setActivePinia(createPinia())
    mockApi.deleteWorkItem.mockResolvedValueOnce({ data: { ok: true } })
    mockApi.getWorkLog.mockResolvedValueOnce({ data: { id: 'wl-1', date: '2026-08-02', items: [] } })
    const store = useWorkLogStore()
    await store.deleteQuickEntry('2026-08-02', 'item-1')
    expect(mockApi.deleteWorkItem).toHaveBeenCalledWith('2026-08-02', 'item-1')
  })

  it('addQuickEntry does not call getTodayContext on success', async () => {
    setActivePinia(createPinia())
    mockApi.appendWorkItem.mockResolvedValueOnce({ data: {} })
    mockApi.getWorkLog.mockResolvedValueOnce({ data: { id: 'wl-1', date: '2026-08-02', items: [] } })
    const store = useWorkLogStore()
    await store.addQuickEntry('2026-08-02', { activity: 'x', start_time: '09:00', end_time: '10:00', quadrant: 1 })
    expect(mockApi.getTodayContext).not.toHaveBeenCalled()
  })
})

describe('WorkLog Store - addWorkItemsBatch', () => {
  beforeEach(() => {
    mockApi.getWorkLog.mockResolvedValue({ data: { id: 'wl-1', date: '2026-08-03', items: [] } })
  })

  it('all success returns successCount=N, failureIndices=[]', async () => {
    setActivePinia(createPinia())
    mockApi.appendWorkItem.mockResolvedValue({ data: {} })
    const store = useWorkLogStore()
    const items = [
      { activity: 'a', start_time: '09:00', end_time: '10:00', quadrant: 1 as const },
      { activity: 'b', start_time: '10:00', end_time: '11:00', quadrant: 2 as const },
    ]
    const result = await store.addWorkItemsBatch('2026-08-03', items)
    expect(result.successCount).toBe(2)
    expect(result.failureIndices).toEqual([])
    expect(mockApi.appendWorkItem).toHaveBeenCalledTimes(2)
  })

  it('mid-failure returns failureIndices=[i], partial success count', async () => {
    setActivePinia(createPinia())
    mockApi.appendWorkItem
      .mockResolvedValueOnce({ data: {} })
      .mockRejectedValueOnce(new Error('server error'))
    const store = useWorkLogStore()
    const items = [
      { activity: 'a', start_time: '09:00', end_time: '10:00', quadrant: 1 as const },
      { activity: 'b', start_time: '10:00', end_time: '11:00', quadrant: 2 as const },
    ]
    const result = await store.addWorkItemsBatch('2026-08-03', items)
    expect(result.successCount).toBe(1)
    expect(result.failureIndices).toEqual([1])
  })
})

describe('WorkLog Store - updateSummary', () => {
  it('success does not show warning toast', async () => {
    setActivePinia(createPinia())
    mockApi.updateWorkLogSummary.mockResolvedValue({ data: { ok: true } })
    const store = useWorkLogStore()
    await store.updateSummary('2026-08-03', '今日小结')
    expect(mockApi.updateWorkLogSummary).toHaveBeenCalledWith('2026-08-03', '今日小结')
    expect(mockEl.warning).not.toHaveBeenCalled()
  })

  it('failure shows warning toast but does not throw', async () => {
    setActivePinia(createPinia())
    mockApi.updateWorkLogSummary.mockRejectedValue(new Error('fail'))
    const store = useWorkLogStore()
    await expect(store.updateSummary('2026-08-03', 'x')).resolves.toBeUndefined()
    expect(mockEl.warning).toHaveBeenCalled()
  })
})
