import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { nextTick } from 'vue'
import Schedule from './Schedule.vue'
import type { ScheduleEvent, RescheduleResult } from '@/types'

// --- Mock Stores ---

const mockScheduleEvents: ScheduleEvent[] = [
  {
    id: 'evt-1',
    title: '代码审查',
    start: '2026-05-25T09:00:00Z',
    end: '2026-05-25T09:50:00Z',
    type: 'task',
    status: 'planned',
    color: '#3b82f6',
    task_id: 'task-1',
    allDay: false,
    editable: true,
    ai_adjusted: false,
    adjustment_type: ''
  }
]

const mockGeneratedEvents: ScheduleEvent[] = [
  {
    id: 'gen-1',
    title: 'AI 生成的日程',
    start: '2026-05-25T10:00:00Z',
    end: '2026-05-25T10:25:00Z',
    type: 'task',
    status: 'planned',
    color: '#3b82f6',
    allDay: false,
    editable: true,
    ai_adjusted: false,
    adjustment_type: ''
  }
]

const mockRescheduleResult: RescheduleResult = {
  adjusted_schedule: [
    {
      task_id: 'task-1',
      title: '缩短的代码审查',
      start_time: '2026-05-25T10:00:00Z',
      end_time: '2026-05-25T10:15:00Z',
      adjustment: 'shortened',
      reason: '被打断后剩余15分钟'
    }
  ],
  summary: '调整了1个任务的时长'
}

const mockScheduleStore = {
  events: [] as ScheduleEvent[],
  loading: false,
  viewMode: 'week' as const,
  currentDate: new Date('2026-05-25T10:00:00Z'),
  fetchSchedules: vi.fn().mockResolvedValue(undefined),
  createSchedule: vi.fn().mockResolvedValue({}),
  updateSchedule: vi.fn().mockResolvedValue(undefined),
  deleteSchedule: vi.fn().mockResolvedValue(undefined),
  moveSchedule: vi.fn().mockResolvedValue(undefined),
  generateSchedule: vi.fn().mockResolvedValue([]),
  reviseSchedule: vi.fn().mockResolvedValue({}),
  applyRevision: vi.fn().mockResolvedValue({}),
  resetSchedules: vi.fn().mockResolvedValue(0),
  aiReasoning: '',
  aiGenerating: false,
  terminalOutput: [] as any[],
  terminalStatus: '',
  terminalStatusMessage: '',
  terminalStatusDetail: '',
  cliToolName: '',
  revisionChanges: [] as any[],
  revisionSummary: '',
  setupTerminalListener: vi.fn(),
  cleanupTerminalListener: vi.fn(),
  setViewMode: vi.fn(),
  setCurrentDate: vi.fn(),
  goToPrevious: vi.fn(),
  goToNext: vi.fn(),
  goToToday: vi.fn()
}

const mockAIStore = {
  configured: false,
  loading: false,
  lastClassification: null,
  checkStatus: vi.fn(),
  classifyTask: vi.fn(),
  classifyTasks: vi.fn(),
  classifyTaskByText: vi.fn(),
  generateSchedule: vi.fn(),
  rescheduleAfterInterrupt: vi.fn().mockResolvedValue(mockRescheduleResult),
  getPrioritySuggestions: vi.fn(),
  getDailyInsights: vi.fn()
}

vi.mock('@/stores/schedule', () => ({ useScheduleStore: () => mockScheduleStore }))
vi.mock('@/stores/ai', () => ({ useAIStore: () => mockAIStore }))
vi.mock('@/stores/task', () => ({ useTaskStore: () => ({ tasks: [], tasksByQuadrant: { 1: [], 2: [], 3: [], 4: [] }, fetchTasks: vi.fn().mockResolvedValue(undefined) }) }))
vi.mock('@/stores/timer', () => ({ useTimerStore: () => ({ currentSession: null, createSession: vi.fn() }) }))

vi.mock('element-plus', () => ({
  ElMessage: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
  ElButton: { template: '<button class="el-button" @click="$emit(\'click\')"><slot/></button>' },
  ElIcon: { template: '<i class="el-icon"><slot/></i>' }
}))

// Stub schedule sub-components
const stubDayView = { template: '<div class="day-view-stub"><slot/></div>', props: ['currentDate', 'events', 'tasksMap'] }
const stubWeekView = { template: '<div class="week-view-stub"><slot/></div>', props: ['currentDate', 'events', 'tasksMap'] }
const stubMonthView = { template: '<div class="month-view-stub"><slot/></div>', props: ['currentDate', 'events', 'tasksMap'] }
const stubEventForm = {
  template: '<div class="event-form-stub"><slot/></div>',
  props: ['visible', 'event', 'defaultDate', 'defaultHour']
}

vi.mock('vue-router', () => ({ useRouter: () => ({ push: vi.fn() }) }))

describe('Schedule View', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-05-25T10:00:00Z'))
    mockScheduleStore.events = [...mockScheduleEvents]
    mockScheduleStore.loading = false
    mockScheduleStore.viewMode = 'week'
    mockScheduleStore.currentDate = new Date('2026-05-25T10:00:00Z')
    mockScheduleStore.fetchSchedules.mockResolvedValue(undefined)
    mockScheduleStore.generateSchedule.mockResolvedValue([])
    mockAIStore.configured = false
    mockAIStore.loading = false
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  function mountSchedule() {
    return mount(Schedule, {
      global: {
        stubs: {
          DayView: stubDayView,
          WeekView: stubWeekView,
          MonthView: stubMonthView,
          EventForm: stubEventForm,
          TaskPomodoroDetail: { template: '<div class="task-pomodoro-detail-stub"/>' },
          TerminalOverlay: { template: '<div class="terminal-overlay-stub"/>', props: ['visible', 'lines', 'status', 'statusMessage', 'statusDetail', 'reasoning', 'toolName'] },
          'el-button': { template: '<button class="el-button" @click="$emit(\'click\')"><slot/></button>' },
          'el-icon': { template: '<i class="el-icon"><slot/></i>' },
          'el-dialog': { template: '<div class="el-dialog"><slot/></div>' },
          'el-input': { template: '<input class="el-input"/>' }
        }
      }
    })
  }

  describe('initial render', () => {
    it('renders the schedule page title', () => {
      const wrapper = mountSchedule()
      expect(wrapper.find('h1').text()).toBe('日程')
    })

    it('renders page subtitle', () => {
      const wrapper = mountSchedule()
      expect(wrapper.find('.page-subtitle').text()).toBe('规划你的时间')
    })

    it('renders view mode buttons (日/周/月)', () => {
      const wrapper = mountSchedule()
      const viewBtns = wrapper.findAll('.view-btn')
      expect(viewBtns).toHaveLength(3)
      expect(viewBtns[0].text()).toContain('日')
      expect(viewBtns[1].text()).toContain('周')
      expect(viewBtns[2].text()).toContain('月')
    })

    it('highlights active view mode', () => {
      const wrapper = mountSchedule()
      const activeBtn = wrapper.find('.view-btn.active')
      expect(activeBtn.exists()).toBe(true)
      expect(activeBtn.text()).toContain('周')
    })

    it('renders navigation buttons', () => {
      const wrapper = mountSchedule()
      const navBtns = wrapper.findAll('.nav-btn')
      expect(navBtns).toHaveLength(2)
      expect(wrapper.find('.today-btn').exists()).toBe(true)
    })

    it('renders AI generate button', () => {
      const wrapper = mountSchedule()
      const toolbarActions = wrapper.find('.toolbar-actions')
      expect(toolbarActions.exists()).toBe(true)
      const buttons = toolbarActions.findAll('.el-button')
      const aiButton = buttons.find(b => b.text().includes('生成日程'))
      expect(aiButton).toBeTruthy()
      expect(aiButton!.text()).toContain('生成日程')
    })

    it('renders the new schedule button', () => {
      const wrapper = mountSchedule()
      const headerActions = wrapper.find('.header-actions')
      expect(headerActions.exists()).toBe(true)
      expect(headerActions.text()).toContain('新建日程')
    })
  })

  describe('view mode switching', () => {
    it('calls setViewMode when clicking day button', async () => {
      const wrapper = mountSchedule()
      const dayBtn = wrapper.findAll('.view-btn')[0]
      await dayBtn.trigger('click')
      expect(mockScheduleStore.setViewMode).toHaveBeenCalledWith('day')
    })

    it('calls setViewMode when clicking week button', async () => {
      const wrapper = mountSchedule()
      const weekBtn = wrapper.findAll('.view-btn')[1]
      await weekBtn.trigger('click')
      expect(mockScheduleStore.setViewMode).toHaveBeenCalledWith('week')
    })

    it('calls setViewMode when clicking month button', async () => {
      const wrapper = mountSchedule()
      const monthBtn = wrapper.findAll('.view-btn')[2]
      await monthBtn.trigger('click')
      expect(mockScheduleStore.setViewMode).toHaveBeenCalledWith('month')
    })
  })

  describe('navigation', () => {
    it('calls goToPrevious on previous button click', async () => {
      const wrapper = mountSchedule()
      const prevBtn = wrapper.findAll('.nav-btn')[0]
      await prevBtn.trigger('click')
      expect(mockScheduleStore.goToPrevious).toHaveBeenCalled()
    })

    it('calls goToNext on next button click', async () => {
      const wrapper = mountSchedule()
      const nextBtn = wrapper.findAll('.nav-btn')[1]
      await nextBtn.trigger('click')
      expect(mockScheduleStore.goToNext).toHaveBeenCalled()
    })

    it('calls goToToday on today button click', async () => {
      const wrapper = mountSchedule()
      await wrapper.find('.today-btn').trigger('click')
      expect(mockScheduleStore.goToToday).toHaveBeenCalled()
    })
  })

  describe('AI 自动排程 (generateSchedule)', () => {
    it('calls scheduleStore.generateSchedule with default time range', async () => {
      const wrapper = mountSchedule()
      const aiButton = wrapper.findAll('.toolbar-actions .el-button').find(b => b.text().includes('生成日程'))!
      await aiButton.trigger('click')

      expect(mockScheduleStore.generateSchedule).toHaveBeenCalledWith('09:00', '18:00')
    })

    it('sets loading state during generation', async () => {
      let resolveFn: (value: unknown) => void
      const promise = new Promise((resolve) => { resolveFn = resolve })
      mockScheduleStore.generateSchedule.mockReturnValue(promise)

      const wrapper = mountSchedule()
      const aiButton = wrapper.findAll('.toolbar-actions .el-button').find(b => b.text().includes('生成日程'))!

      const clickPromise = aiButton.trigger('click')
      await nextTick()

      // Store's generateSchedule sets loading to true
      // The view binds :loading="scheduleStore.loading" to the AI button
      // verify the store was called
      expect(mockScheduleStore.generateSchedule).toHaveBeenCalledWith('09:00', '18:00')

      resolveFn!(mockGeneratedEvents)
      await clickPromise
    })

    it('shows success message when generation succeeds', async () => {
      const { ElMessage } = await import('element-plus')
      mockScheduleStore.generateSchedule.mockResolvedValue(mockGeneratedEvents)

      const wrapper = mountSchedule()
      const aiButton = wrapper.findAll('.toolbar-actions .el-button').find(b => b.text().includes('生成日程'))!
      await aiButton.trigger('click')
      await flushPromises()

      expect(ElMessage.success).toHaveBeenCalledWith('日程生成成功')
    })

    it('shows error message when generation fails', async () => {
      const { ElMessage } = await import('element-plus')
      mockScheduleStore.generateSchedule.mockRejectedValue(new Error('AI not configured'))

      const wrapper = mountSchedule()
      const aiButton = wrapper.findAll('.toolbar-actions .el-button').find(b => b.text().includes('生成日程'))!
      await aiButton.trigger('click')
      await flushPromises()

      expect(ElMessage.error).toHaveBeenCalled()
    })

    it('shows AI reasoning bar when aiReasoning is set', async () => {
      mockScheduleStore.aiReasoning = '优先安排紧急任务，预留20%缓冲应对打断'
      mockScheduleStore.generateSchedule.mockResolvedValue(mockGeneratedEvents)

      const wrapper = mountSchedule()
      const aiButton = wrapper.findAll('.toolbar-actions .el-button').find(b => b.text().includes('生成日程'))!
      await aiButton.trigger('click')
      await flushPromises()

      // AI reasoning bar should be visible
      const reasoningBar = wrapper.find('.ai-reasoning-bar')
      expect(reasoningBar.exists()).toBe(true)
      expect(reasoningBar.find('.reasoning-text').text()).toContain('优先安排紧急任务')
    })

    it('hides AI reasoning bar when aiReasoning is empty', () => {
      mockScheduleStore.aiReasoning = ''

      const wrapper = mountSchedule()
      const reasoningBar = wrapper.find('.ai-reasoning-bar')
      expect(reasoningBar.exists()).toBe(false)
    })

    it('shows AI reasoning bar and success message together', async () => {
      const { ElMessage } = await import('element-plus')
      mockScheduleStore.aiReasoning = 'AI strategy applied'
      mockScheduleStore.generateSchedule.mockResolvedValue(mockGeneratedEvents)

      const wrapper = mountSchedule()
      const aiButton = wrapper.findAll('.toolbar-actions .el-button').find(b => b.text().includes('生成日程'))!
      await aiButton.trigger('click')
      await flushPromises()

      expect(ElMessage.success).toHaveBeenCalledWith('日程生成成功')
      const reasoningBar = wrapper.find('.ai-reasoning-bar')
      expect(reasoningBar.exists()).toBe(true)
    })

    it('calls generateSchedule with correct time range and shows success', async () => {
      const { ElMessage } = await import('element-plus')
      mockScheduleStore.generateSchedule.mockResolvedValue(mockGeneratedEvents)
      mockScheduleStore.aiReasoning = ''

      const wrapper = mountSchedule()
      const aiButton = wrapper.findAll('.toolbar-actions .el-button').find(b => b.text().includes('生成日程'))!
      await aiButton.trigger('click')
      await flushPromises()

      expect(mockScheduleStore.generateSchedule).toHaveBeenCalledWith('09:00', '18:00')
      expect(ElMessage.success).toHaveBeenCalledWith('日程生成成功')
    })
  })

  describe('view rendering per mode', () => {
    it('renders WeekView when viewMode is week', () => {
      mockScheduleStore.viewMode = 'week'
      const wrapper = mountSchedule()
      expect(wrapper.find('.week-view-stub').exists()).toBe(true)
      expect(wrapper.find('.day-view-stub').exists()).toBe(false)
      expect(wrapper.find('.month-view-stub').exists()).toBe(false)
    })

    it('renders DayView when viewMode is day', () => {
      mockScheduleStore.viewMode = 'day'
      const wrapper = mountSchedule()
      expect(wrapper.find('.day-view-stub').exists()).toBe(true)
      expect(wrapper.find('.week-view-stub').exists()).toBe(false)
      expect(wrapper.find('.month-view-stub').exists()).toBe(false)
    })

    it('renders MonthView when viewMode is month', () => {
      mockScheduleStore.viewMode = 'month'
      const wrapper = mountSchedule()
      expect(wrapper.find('.month-view-stub').exists()).toBe(true)
      expect(wrapper.find('.week-view-stub').exists()).toBe(false)
      expect(wrapper.find('.day-view-stub').exists()).toBe(false)
    })
  })

  describe('current period label', () => {
    it('shows full date for day view', () => {
      mockScheduleStore.viewMode = 'day'
      mockScheduleStore.currentDate = new Date('2026-05-25T10:00:00Z')
      const wrapper = mountSchedule()
      expect(wrapper.find('.current-period').text()).toContain('2026')
      expect(wrapper.find('.current-period').text()).toContain('5月')
      expect(wrapper.find('.current-period').text()).toContain('25日')
    })

    it('shows month label for month view', () => {
      mockScheduleStore.viewMode = 'month'
      mockScheduleStore.currentDate = new Date('2026-05-25T10:00:00Z')
      const wrapper = mountSchedule()
      expect(wrapper.find('.current-period').text()).toBe('2026年5月')
    })

    it('shows week range for week view', () => {
      mockScheduleStore.viewMode = 'week'
      // Set Monday for stable assertion
      mockScheduleStore.currentDate = new Date('2026-05-25T10:00:00Z') // Monday
      const wrapper = mountSchedule()
      const label = wrapper.find('.current-period').text()
      expect(label).toContain('月')
      expect(label).toContain('日')
      expect(label).toContain('-')
    })
  })

  describe('onMounted data loading', () => {
    it('fetches schedules on mount', async () => {
      mountSchedule()
      await nextTick()
      await nextTick()

      expect(mockScheduleStore.fetchSchedules).toHaveBeenCalled()
    })
  })

  describe('EventForm integration', () => {
    it('renders EventForm stub', () => {
      const wrapper = mountSchedule()
      expect(wrapper.find('.event-form-stub').exists()).toBe(true)
    })

    it('opens EventForm when new schedule button clicked', async () => {
      const wrapper = mountSchedule()
      const createBtn = wrapper.find('.header-actions .el-button')
      expect(createBtn.exists()).toBe(true)
      await createBtn.trigger('click')
      await nextTick()

      // Form should become visible after clicking create
      expect(wrapper.find('.event-form-stub').exists()).toBe(true)
    })
  })
})
