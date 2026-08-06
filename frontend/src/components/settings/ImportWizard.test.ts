import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ImportWizard from './ImportWizard.vue'
import { api } from '@/api/client'
import type { ImportPreview } from '@/types'

vi.mock('@/api/client', () => ({
  api: {
    previewImport: vi.fn(),
    applyImport: vi.fn()
  }
}))
vi.mock('element-plus', async () => {
  const actual: any = await vi.importActual('element-plus')
  return { ...actual, ElMessage: { success: vi.fn(), error: vi.fn(), warning: vi.fn() } }
})

const previewFixture: ImportPreview = {
  schema_version: 1,
  schema_warning: '',
  modules: {
    tasks: { new: 2, identical: 1, conflict: 1, orphan: 0, conflicts: [{ id: 't1', fields: [{ field: 'status', current: 'todo', imported: 'done' }] }], settings_conflicts: [] },
    sessions: { new: 0, identical: 0, conflict: 0, orphan: 0, conflicts: [], settings_conflicts: [] },
    schedules: { new: 0, identical: 0, conflict: 0, orphan: 0, conflicts: [], settings_conflicts: [] },
    work_logs: { new: 0, identical: 0, conflict: 0, orphan: 0, conflicts: [], settings_conflicts: [] },
    work_reports: { new: 0, identical: 0, conflict: 0, orphan: 0, conflicts: [], settings_conflicts: [] },
    settings: { new: 0, identical: 0, conflict: 0, orphan: 0, conflicts: [], settings_conflicts: [
      { section: 'ai', field: 'api_key', current: 'secret-current', imported: 'secret-imported' },
      { section: 'pomodoro', field: 'work_duration', current: 1500, imported: 1800 }
    ] }
  }
}

function makeFile(): File {
  const env = {
    app: 'ticktask', schema_version: 1, exported_at: '2026-08-07T00:00:00Z',
    data: {
      tasks: [], sessions: [], schedules: [], work_logs: [], work_reports: [],
      settings: {
        pomodoro: { work_duration: 1500, short_break_duration: 300, long_break_duration: 900, long_break_after: 4, auto_start_break: false, auto_start_work: false, enable_sound: true, buffer_ratio: 20, task_time_preferences: '{"management":"any","dev":"any"}' },
        ai: { provider: 'openai', api_key: '', base_url: '', model: '' }
      }
    }
  }
  return new File([JSON.stringify(env)], 'b.json', { type: 'application/json' })
}

beforeEach(() => {
  setActivePinia(createPinia())
  ;(api.previewImport as any).mockResolvedValue({ data: previewFixture })
  ;(api.applyImport as any).mockResolvedValue({ data: { applied: { tasks: { inserted: 2, updated: 1, deleted: 0 } } } })
})

async function toPreview(w: ReturnType<typeof mount>) {
  ;(w.vm as any).onFileSelected(makeFile())
  await flushPromises()
}

describe('ImportWizard', () => {
  it('advances to preview step with module counts', async () => {
    const w = mount(ImportWizard)
    await toPreview(w)
    expect((w.vm as any).step).toBe('preview')
    expect((w.vm as any).preview.modules.tasks.new).toBe(2)
    expect((w.vm as any).preview.modules.tasks.conflict).toBe(1)
  })

  it('masks api_key but shows numeric settings values', async () => {
    const w = mount(ImportWizard)
    await toPreview(w)
    expect((w.vm as any).displayValue('ai', 'api_key', 'secret')).toBe('••••')
    expect((w.vm as any).displayValue('pomodoro', 'work_duration', 1800)).toBe('1800')
  })

  it('changing task policy updates apply payload', async () => {
    const w = mount(ImportWizard)
    await toPreview(w)
    ;(w.vm as any).setPolicy('tasks', 'merge_current')
    expect((w.vm as any).applyPayload.modules.tasks.policy).toBe('merge_current')
  })

  it('settings choice resolves into apply payload data', async () => {
    const w = mount(ImportWizard)
    await toPreview(w)
    ;(w.vm as any).setSettingsChoice('ai', 'api_key', 'file')
    expect((w.vm as any).applyPayload.data.settings.ai.api_key).toBe('secret-imported')
  })

  it('replace with confirm cancelled does not apply', async () => {
    const w = mount(ImportWizard)
    await toPreview(w)
    ;(w.vm as any).setPolicy('tasks', 'replace')
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(false)
    await (w.vm as any).clickApply()
    confirmSpy.mockRestore()
    expect(api.applyImport).not.toHaveBeenCalled()
  })

  it('apply success emits applied and calls api', async () => {
    const w = mount(ImportWizard)
    await toPreview(w)
    await (w.vm as any).clickApply()
    await flushPromises()
    expect(api.applyImport).toHaveBeenCalled()
    expect(w.emitted('applied')).toBeTruthy()
  })

  it('apply failure does not emit applied', async () => {
    ;(api.applyImport as any).mockRejectedValueOnce(new Error('boom'))
    const w = mount(ImportWizard)
    await toPreview(w)
    await (w.vm as any).clickApply()
    await flushPromises()
    expect(w.emitted('applied')).toBeFalsy()
  })
})
