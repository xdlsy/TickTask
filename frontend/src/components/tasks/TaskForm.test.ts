import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import TaskForm from './TaskForm.vue'

const aiStore = vi.hoisted(() => ({
  configured: false,
  classifyTask: vi.fn(),
  classifyTaskByText: vi.fn()
}))

vi.mock('@/stores/ai', () => ({
  useAIStore: () => aiStore
}))

const elMsg = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
  warning: vi.fn()
}))

vi.mock('element-plus', () => ({
  ElMessage: elMsg
}))

const elStubs = {
  'el-dialog': true,
  'el-form': true,
  'el-form-item': true,
  'el-input': true,
  'el-button': true,
  'el-radio-group': true,
  'el-radio-button': true,
  'el-tag': true,
  'el-input-number': true,
  'el-date-picker': true
}

describe('TaskForm', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  describe('dialog title', () => {
    it('shows create title when no task', () => {
      const wrapper = mount(TaskForm, {
        props: { visible: true },
        global: { stubs: elStubs }
      })

      expect(wrapper.props('task')).toBeUndefined()
    })

    it('shows edit title when task provided', () => {
      const wrapper = mount(TaskForm, {
        props: { visible: true, task: { id: '1', title: 'Test', quadrant: 1, estimated_time: 30 } },
        global: { stubs: elStubs }
      })

      expect(wrapper.props('task')).toEqual({ id: '1', title: 'Test', quadrant: 1, estimated_time: 30 })
    })
  })

  describe('formData initialization', () => {
    it('initializes with default values when no task', () => {
      const wrapper = mount(TaskForm, {
        props: { visible: true },
        global: { stubs: elStubs }
      })

      expect(wrapper.vm.formData.title).toBe('')
      expect(wrapper.vm.formData.quadrant).toBe(2)
      expect(wrapper.vm.formData.estimated_time).toBe(0)
      expect(wrapper.vm.formData.tags).toEqual([])
    })

    it('populates formData from task prop', async () => {
      const wrapper = mount(TaskForm, {
        props: { visible: true },
        global: { stubs: elStubs }
      })

      await wrapper.setProps({
        visible: true,
        task: {
          id: 't1',
          title: 'My Task',
          description: 'desc',
          quadrant: 1,
          estimated_time: 45,
          deadline: '2026-05-25T00:00:00Z',
          tags: ['urgent']
        } as any
      })

      expect(wrapper.vm.formData.title).toBe('My Task')
      expect(wrapper.vm.formData.description).toBe('desc')
      expect(wrapper.vm.formData.quadrant).toBe(1)
      expect(wrapper.vm.formData.estimated_time).toBe(45)
      expect(wrapper.vm.formData.tags).toEqual(['urgent'])
    })
  })

  describe('addTag', () => {
    it('adds tag from tagInput', () => {
      const wrapper = mount(TaskForm, {
        props: { visible: true },
        global: { stubs: elStubs }
      })

      wrapper.vm.tagInput = 'new-tag'
      wrapper.vm.addTag()

      expect(wrapper.vm.formData.tags).toContain('new-tag')
      expect(wrapper.vm.tagInput).toBe('')
    })

    it('does not add duplicate tags', () => {
      const wrapper = mount(TaskForm, {
        props: { visible: true },
        global: { stubs: elStubs }
      })

      wrapper.vm.tagInput = 'tag1'
      wrapper.vm.addTag()
      wrapper.vm.tagInput = 'tag1'
      wrapper.vm.addTag()

      expect(wrapper.vm.formData.tags).toEqual(['tag1'])
    })

    it('does not add empty tags', () => {
      const wrapper = mount(TaskForm, {
        props: { visible: true },
        global: { stubs: elStubs }
      })

      wrapper.vm.tagInput = '  '
      wrapper.vm.addTag()

      expect(wrapper.vm.formData.tags).toEqual([])
    })
  })

  describe('removeTag', () => {
    it('removes tag from formData', () => {
      const wrapper = mount(TaskForm, {
        props: { visible: true },
        global: { stubs: elStubs }
      })

      wrapper.vm.formData.tags = ['a', 'b', 'c']
      wrapper.vm.removeTag('b')

      expect(wrapper.vm.formData.tags).toEqual(['a', 'c'])
    })

    it('does nothing for non-existent tag', () => {
      const wrapper = mount(TaskForm, {
        props: { visible: true },
        global: { stubs: elStubs }
      })

      wrapper.vm.formData.tags = ['a', 'b']
      wrapper.vm.removeTag('z')

      expect(wrapper.vm.formData.tags).toEqual(['a', 'b'])
    })
  })

  describe('getQuadrantName', () => {
    it('returns quadrant name for valid quadrant', () => {
      const wrapper = mount(TaskForm, {
        props: { visible: true },
        global: { stubs: elStubs }
      })

      expect(wrapper.vm.getQuadrantName(1)).toBe('重要且紧急')
      expect(wrapper.vm.getQuadrantName(3)).toBe('紧急不重要')
    })
  })

  describe('getQuadrantTagType', () => {
    it('returns correct tag type per quadrant', () => {
      const wrapper = mount(TaskForm, {
        props: { visible: true },
        global: { stubs: elStubs }
      })

      expect(wrapper.vm.getQuadrantTagType(1)).toBe('danger')
      expect(wrapper.vm.getQuadrantTagType(2)).toBe('warning')
      expect(wrapper.vm.getQuadrantTagType(3)).toBe('primary')
      expect(wrapper.vm.getQuadrantTagType(4)).toBe('info')
    })
  })

  describe('getAIRecommendation', () => {
    it('warns if title is empty', async () => {
      const wrapper = mount(TaskForm, {
        props: { visible: true },
        global: { stubs: elStubs }
      })

      await wrapper.vm.getAIRecommendation()

      expect(elMsg.warning).toHaveBeenCalledWith('请先输入任务标题')
    })

    it('calls aiStore.classifyTaskByText and stores result', async () => {
      const wrapper = mount(TaskForm, {
        props: { visible: true },
        global: { stubs: elStubs }
      })

      wrapper.vm.formData.title = 'Test'
      const classification = { task_id: '1', important: true, urgent: false, quadrant: 2, reason: 'test' }
      aiStore.classifyTaskByText = vi.fn().mockResolvedValue(classification)

      await wrapper.vm.getAIRecommendation()

      expect(aiStore.classifyTaskByText).toHaveBeenCalledWith('Test', '')
      expect(wrapper.vm.aiRecommendation).toEqual(classification)
      expect(wrapper.vm.aiClassifying).toBe(false)
    })

    it('shows error on classify failure', async () => {
      const wrapper = mount(TaskForm, {
        props: { visible: true },
        global: { stubs: elStubs }
      })

      wrapper.vm.formData.title = 'Test'
      aiStore.classifyTaskByText = vi.fn().mockRejectedValue(new Error('fail'))

      await wrapper.vm.getAIRecommendation()

      expect(elMsg.error).toHaveBeenCalledWith('AI 推荐失败')
    })
  })

  describe('applyRecommendation', () => {
    it('applies recommendation quadrant and clears it', () => {
      const wrapper = mount(TaskForm, {
        props: { visible: true },
        global: { stubs: elStubs }
      })

      wrapper.vm.aiRecommendation = { task_id: '1', important: true, urgent: true, quadrant: 1, reason: 'urgent' }
      wrapper.vm.applyRecommendation()

      expect(wrapper.vm.formData.quadrant).toBe(1)
      expect(wrapper.vm.aiRecommendation).toBeNull()
      expect(elMsg.success).toHaveBeenCalledWith('已采纳 AI 推荐')
    })
  })

  describe('onSave', () => {
    it('does not emit save when title is empty', () => {
      const wrapper = mount(TaskForm, {
        props: { visible: true },
        global: { stubs: elStubs }
      })

      wrapper.vm.onSave()

      expect(wrapper.emitted('save')).toBeFalsy()
    })

    it('emits save with form data and resets', () => {
      const wrapper = mount(TaskForm, {
        props: { visible: true },
        global: { stubs: elStubs }
      })

      wrapper.vm.formData = {
        title: 'New Task',
        description: 'desc',
        quadrant: 1,
        estimated_time: 30,
        deadline: new Date('2026-12-25T00:00:00Z'),
        tags: ['work']
      }

      wrapper.vm.onSave()

      const emitted = wrapper.emitted('save')
      expect(emitted).toBeTruthy()
      expect(emitted![0][0]).toMatchObject({
        title: 'New Task',
        description: 'desc',
        quadrant: 1,
        estimated_time: 30,
        tags: ['work']
      })
      expect(wrapper.vm.formData.title).toBe('')
    })

    it('emits null deadline when not set', () => {
      const wrapper = mount(TaskForm, {
        props: { visible: true },
        global: { stubs: elStubs }
      })

      wrapper.vm.formData = {
        title: 'Task',
        description: '',
        quadrant: 2,
        estimated_time: 0,
        deadline: null,
        tags: []
      }

      wrapper.vm.onSave()

      const emitted = wrapper.emitted('save')
      expect(emitted![0][0].deadline).toBeNull()
    })
  })
})
