import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ElementPlus from 'element-plus'
import { ElMessage } from 'element-plus'
import WorkItemForm from './WorkItemForm.vue'

vi.mock('@/api/client', () => ({
  api: {
    appendWorkItem: vi.fn().mockResolvedValue({ data: {} }),
    updateWorkItem: vi.fn().mockResolvedValue({ data: { ok: true } }),
    getWorkLog: vi.fn().mockResolvedValue({ data: { id: 'l1', date: '2026-08-03', items: [] } }),
  },
}))

vi.spyOn(ElMessage, 'success').mockImplementation(() => ({}) as any)
vi.spyOn(ElMessage, 'error').mockImplementation(() => ({}) as any)

const mountOpts = { global: { plugins: [ElementPlus] } }

describe('WorkItemForm', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('renders required + optional fields in add mode', () => {
    const wrapper = mount(WorkItemForm, {
      props: { date: '2026-08-03', mode: 'add' },
      ...mountOpts,
    })
    expect(wrapper.find('[data-test="activity-input"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="start-input"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="end-input"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="quadrant-input"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="content-input"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="problem-solved-input"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="result-input"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="impact-input"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="submit-btn"]').text()).toContain('添加')
  })

  it('rejects submit when activity is empty', async () => {
    const wrapper = mount(WorkItemForm, {
      props: { date: '2026-08-03', mode: 'add' },
      ...mountOpts,
    })
    await wrapper.find('[data-test="submit-btn"]').trigger('click')
    expect(ElMessage.error).toHaveBeenCalled()
    expect(wrapper.emitted('added')).toBeFalsy()
  })

  it('submits with all fields and emits added on success', async () => {
    const wrapper = mount(WorkItemForm, {
      props: { date: '2026-08-03', mode: 'add' },
      ...mountOpts,
    })
    await wrapper.find('[data-test="activity-input"]').setValue('晨会')
    await wrapper.find('[data-test="content-input"]').setValue('内容')
    await wrapper.find('[data-test="submit-btn"]').trigger('click')
    // flush all microtasks for async chain (validate -> addQuickEntry -> emit)
    for (let i = 0; i < 6; i++) {
      await Promise.resolve()
    }
    expect(wrapper.emitted('added')).toBeTruthy()
  })

  it('edit mode: prefills all fields from initial prop and shows cancel button', async () => {
    const wrapper = mount(WorkItemForm, {
      props: {
        date: '2026-08-03',
        mode: 'edit',
        itemId: 'item-1',
        initial: {
          activity: '原活动',
          start_time: '09:00',
          end_time: '10:00',
          quadrant: 2,
          content: '原内容',
          problem_solved: '原问题',
          result: '原结果',
          impact: '原影响',
        },
      },
      ...mountOpts,
    })
    expect((wrapper.find('[data-test="activity-input"]').element as HTMLInputElement).value).toBe('原活动')
    expect((wrapper.find('[data-test="content-input"]').element as HTMLTextAreaElement).value).toBe('原内容')
    expect(wrapper.find('[data-test="cancel-btn"]').exists()).toBe(true)
  })

  it('edit mode: cancel emits cancel event', async () => {
    const wrapper = mount(WorkItemForm, {
      props: { date: '2026-08-03', mode: 'edit', itemId: 'x', initial: {} },
      ...mountOpts,
    })
    await wrapper.find('[data-test="cancel-btn"]').trigger('click')
    expect(wrapper.emitted('cancel')).toBeTruthy()
  })
})
