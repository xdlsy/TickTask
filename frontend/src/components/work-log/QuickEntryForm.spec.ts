import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ElementPlus from 'element-plus'
import { ElMessage } from 'element-plus'
import QuickEntryForm from './QuickEntryForm.vue'

vi.mock('@/api/client', () => ({
  api: {
    appendWorkItem: vi.fn().mockResolvedValue({ data: {} }),
    updateWorkItem: vi.fn().mockResolvedValue({ data: { ok: true } }),
    deleteWorkItem: vi.fn().mockResolvedValue({ data: { ok: true } }),
    getWorkLog: vi.fn().mockResolvedValue({ data: { id: 'l1', date: '2026-08-02', items: [] } }),
    getTodayContext: vi.fn(),
  },
}))

// 保留 ElementPlus 真实组件（让 data-test 选择器可用），仅 spy ElMessage
vi.spyOn(ElMessage, 'success').mockImplementation(() => ({}) as any)
vi.spyOn(ElMessage, 'error').mockImplementation(() => ({}) as any)

const mountOpts = {
  global: { plugins: [ElementPlus] },
}

describe('QuickEntryForm', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('renders with default props in add mode', () => {
    const wrapper = mount(QuickEntryForm, {
      props: { date: '2026-08-02', mode: 'add' },
      ...mountOpts,
    })
    expect(wrapper.find('[data-test="submit-btn"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="cancel-btn"]').exists()).toBe(false)
  })

  it('shows cancel button in edit mode', () => {
    const wrapper = mount(QuickEntryForm, {
      props: {
        date: '2026-08-02',
        mode: 'edit',
        itemId: 'item-1',
        initial: { activity: 'x', start_time: '09:00', end_time: '10:00', quadrant: 1 },
      },
      ...mountOpts,
    })
    expect(wrapper.find('[data-test="cancel-btn"]').exists()).toBe(true)
  })

  it('emits cancel when mode=edit and cancel clicked', async () => {
    const wrapper = mount(QuickEntryForm, {
      props: {
        date: '2026-08-02',
        mode: 'edit',
        itemId: 'item-1',
        initial: { activity: 'x', start_time: '09:00', end_time: '10:00', quadrant: 1 },
      },
      ...mountOpts,
    })
    await wrapper.find('[data-test="cancel-btn"]').trigger('click')
    expect(wrapper.emitted('cancel')).toBeTruthy()
  })

  it('emits added after successful add submit', async () => {
    const wrapper = mount(QuickEntryForm, {
      props: { date: '2026-08-02', mode: 'add' },
      ...mountOpts,
    })
    // 填表（el-input 渲染原生 input，setValue 可用）
    await wrapper.find('[data-test="activity-input"]').setValue('晨会')
    await wrapper.find('[data-test="submit-btn"]').trigger('click')
    // 等待异步
    await new Promise(r => setTimeout(r, 0))
    expect(wrapper.emitted('added')).toBeTruthy()
  })

  it('does not submit when activity is empty', async () => {
    const wrapper = mount(QuickEntryForm, {
      props: { date: '2026-08-02', mode: 'add' },
      ...mountOpts,
    })
    await wrapper.find('[data-test="submit-btn"]').trigger('click')
    await new Promise(r => setTimeout(r, 0))
    expect(wrapper.emitted('added')).toBeFalsy()
  })
})
