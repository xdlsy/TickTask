import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ElementPlus from 'element-plus'
import { ElMessage } from 'element-plus'
import BatchTableEditor from './BatchTableEditor.vue'
import type { DraftWorkItem } from './BatchTableEditor.vue'

vi.mock('@/api/client', () => ({
  api: {
    appendWorkItem: vi.fn(),
    updateWorkLogSummary: vi.fn(),
    getWorkLog: vi.fn().mockResolvedValue({ data: { id: 'l1', date: '2026-08-03', items: [] } }),
  },
}))

vi.spyOn(ElMessage, 'success').mockImplementation(() => ({}) as any)
vi.spyOn(ElMessage, 'error').mockImplementation(() => ({}) as any)
vi.spyOn(ElMessage, 'warning').mockImplementation(() => ({}) as any)

const mountOpts = { global: { plugins: [ElementPlus] } }

const sampleDraft = (): DraftWorkItem[] => [
  {
    activity: '',
    start_time: '09:00',
    end_time: '10:00',
    quadrant: 2,
    content: '内容1',
    problem_solved: '问题1',
    result: '结果1',
    impact: '影响1',
  },
]

describe('BatchTableEditor', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('renders one row per draft item plus an add-row', () => {
    const wrapper = mount(BatchTableEditor, {
      props: {
        date: '2026-08-03',
        items: sampleDraft(),
        summary: '',
      },
      ...mountOpts,
    })
    expect(wrapper.findAll('[data-test="draft-row"]')).toHaveLength(1)
    expect(wrapper.find('[data-test="add-row"]').exists()).toBe(true)
  })

  it('clicking add-row emits update:items with new empty row', async () => {
    const wrapper = mount(BatchTableEditor, {
      props: {
        date: '2026-08-03',
        items: sampleDraft(),
        summary: '',
      },
      ...mountOpts,
    })
    await wrapper.find('[data-test="add-row"]').trigger('click')
    const emitted = wrapper.emitted('update:items')
    expect(emitted).toBeTruthy()
    expect(emitted![0][0]).toHaveLength(2)
  })

  it('delete row removes it from items', async () => {
    const wrapper = mount(BatchTableEditor, {
      props: {
        date: '2026-08-03',
        items: sampleDraft(),
        summary: '',
      },
      ...mountOpts,
    })
    await wrapper.find('[data-test="delete-btn"]').trigger('click')
    const emitted = wrapper.emitted('update:items')
    expect(emitted).toBeTruthy()
    expect(emitted![0][0]).toHaveLength(0)
  })

  it('batch save validates empty activity and shows error', async () => {
    const wrapper = mount(BatchTableEditor, {
      props: {
        date: '2026-08-03',
        items: sampleDraft(), // activity is empty
        summary: '',
      },
      ...mountOpts,
    })
    await wrapper.find('[data-test="save-btn"]').trigger('click')
    for (let i = 0; i < 4; i++) await Promise.resolve()
    expect(ElMessage.error).toHaveBeenCalled()
  })

  it('batch save with all valid triggers save emit', async () => {
    const items = sampleDraft()
    items[0].activity = '填好了'
    const wrapper = mount(BatchTableEditor, {
      props: {
        date: '2026-08-03',
        items,
        summary: '今日小结',
      },
      ...mountOpts,
    })
    const { api } = await import('@/api/client')
    ;(api.appendWorkItem as any).mockResolvedValue({ data: {} })
    ;(api.updateWorkLogSummary as any).mockResolvedValue({ data: { ok: true } })

    await wrapper.find('[data-test="save-btn"]').trigger('click')
    for (let i = 0; i < 8; i++) await Promise.resolve()

    expect(api.appendWorkItem).toHaveBeenCalledTimes(1)
    expect(wrapper.emitted('save')).toBeTruthy()
  })

  it('discard button emits discard event', async () => {
    const wrapper = mount(BatchTableEditor, {
      props: {
        date: '2026-08-03',
        items: sampleDraft(),
        summary: '',
      },
      ...mountOpts,
    })
    await wrapper.find('[data-test="discard-btn"]').trigger('click')
    expect(wrapper.emitted('discard')).toBeTruthy()
  })
})
