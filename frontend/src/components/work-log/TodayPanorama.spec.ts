import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import ElementPlus from 'element-plus'
import { ElMessage } from 'element-plus'
import TodayPanorama from './TodayPanorama.vue'
import { useWorkLogStore } from '@/stores/workLog'
import type { WorkLog, WorkItem } from '@/types'

vi.mock('@/api/client', () => ({
  api: {
    deleteWorkItem: vi.fn().mockResolvedValue({ data: { ok: true } }),
    getWorkLog: vi.fn().mockResolvedValue({ data: { id: 'l1', date: '2026-08-02', items: [] } }),
    getTodayContext: vi.fn(),
  },
}))

vi.spyOn(ElMessage, 'success').mockImplementation(() => ({}) as any)
vi.spyOn(ElMessage, 'error').mockImplementation(() => ({}) as any)

const mountOpts = { global: { plugins: [ElementPlus] } }

function makeItem(over: Partial<WorkItem>): WorkItem {
  return {
    id: 'x', work_log_id: 'l1', seq: 1, title: '', content: '',
    problem_solved: '', result: '', impact: '', source: 'manual',
    activity: 'test', start_time: '09:00', end_time: '10:00', quadrant: 1,
    ...over,
  } as WorkItem
}

describe('TodayPanorama', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('shows empty state when no manual items', () => {
    const store = useWorkLogStore()
    store.currentLog = { id: 'l1', date: '2026-08-02', items: [] } as WorkLog
    const wrapper = mount(TodayPanorama, {
      props: { date: '2026-08-02' },
      ...mountOpts,
    })
    expect(wrapper.text()).toContain('还没有记录')
  })

  it('renders manual items sorted by time', async () => {
    const store = useWorkLogStore()
    store.currentLog = {
      id: 'l1', date: '2026-08-02',
      items: [
        makeItem({ id: '1', start_time: '10:00', end_time: '11:00', activity: 'b' }),
        makeItem({ id: '2', start_time: '09:00', end_time: '10:00', activity: 'a' }),
      ],
    } as WorkLog
    const wrapper = mount(TodayPanorama, {
      props: { date: '2026-08-02' },
      ...mountOpts,
    })
    // el-table 异步渲染行内容，需等待 nextTick
    await nextTick()
    await nextTick()
    // 用 .el-table__row 选择行（el-table 不透传 data-test，LRN-019）
    const rows = wrapper.findAll('.el-table__row')
    expect(rows).toHaveLength(2)
    // 09:00 应该排第一（store.todayManualItems 已 sort）
    expect(rows[0].text()).toContain('a')
    expect(rows[0].text()).toContain('09:00')
  })

  it('emits edit with item id when edit clicked', async () => {
    const store = useWorkLogStore()
    store.currentLog = {
      id: 'l1', date: '2026-08-02',
      items: [makeItem({ id: 'x1', activity: '晨会' })],
    } as WorkLog
    const wrapper = mount(TodayPanorama, {
      props: { date: '2026-08-02' },
      ...mountOpts,
    })
    await nextTick()
    await nextTick()
    await wrapper.find('[data-test="edit-btn"]').trigger('click')
    expect(wrapper.emitted('edit')?.[0]).toEqual(['x1'])
  })
})
