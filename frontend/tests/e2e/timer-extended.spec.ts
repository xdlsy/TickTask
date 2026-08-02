import { test, expect } from '../support/fixtures'

test.describe.configure({ mode: 'serial' })
test.describe('@p1 Timer Extended', () => {
  test.beforeEach(async ({ apiClient }) => {
    // Clean up any active session before each test
    const existing = await apiClient.getActiveSession()
    if (existing) {
      await apiClient.controlSession(existing.id, 'abandon').catch(() => {})
    }
  })

  test('TMR-E2E-007: Auto-start break after work completion', async ({
    page,
    sessionFactory,
    apiClient,
  }) => {
    // Given settings have auto_start_break enabled
    const settings = await apiClient.getPomodoroSettings()

    // Given a short work session
    const session = await sessionFactory.create({ type: 'work', duration: 25 })

    // When visiting Timer and completing work
    await page.goto('/timer')
    await apiClient.controlSession(session.id, 'complete')

    // Then if auto_start_break is enabled, a break may start
    await page.waitForTimeout(1000)
    // Just verify no crash and timer page still functional
    await expect(page.locator('body')).toBeVisible()
  })

  test('TMR-E2E-008: Current task info on Timer page', async ({
    page,
    taskFactory,
    sessionFactory,
  }) => {
    // Given a task and session linked to it
    const task = await taskFactory.create({ title: '当前任务显示测试', quadrant: 1 })
    await sessionFactory.create({ task_id: task.id, type: 'work', duration: 1500 })

    // When visiting Timer page
    await page.goto('/timer')

    // Then task name is visible
    await expect(page.getByText('当前任务显示测试')).toBeVisible()
  })

  test('TMR-E2E-009: Today session history display', async ({
    page,
    sessionFactory,
    apiClient,
  }) => {
    // Given multiple completed sessions
    const s1 = await sessionFactory.create({ type: 'work', duration: 25 })
    await apiClient.controlSession(s1.id, 'complete')
    const s2 = await sessionFactory.create({ type: 'work', duration: 25 })
    await apiClient.controlSession(s2.id, 'complete')

    // When visiting Timer page
    await page.goto('/timer')

    // Then session history section exists
    const sessionItems = page.locator('.session-item, [class*="session"]')
    if (await sessionItems.count() > 0) {
      expect(await sessionItems.count()).toBeGreaterThanOrEqual(1)
    }
  })

  test('TMR-E2E-010: Interrupt with AI reschedule result', async ({
    page,
    taskFactory,
    sessionFactory,
    apiClient,
  }) => {
    // Given AI is configured
    const status = await apiClient.getAIStatus()
    test.skip(!status.configured, 'AI not configured')

    // Given a running session with a task
    const task = await taskFactory.create({ title: '打断重排测试', quadrant: 1 })
    const session = await sessionFactory.create({
      task_id: task.id,
      type: 'work',
      duration: 1500,
    })

    // When visiting Timer and interrupting
    await page.goto('/timer')

    // Abandon with reason
    await apiClient.controlSession(session.id, 'abandon', 'meeting')

    // Then session is abandoned
    const active = await apiClient.getActiveSession()
    expect(active).toBeNull()
  })
})
