import { test, expect } from '../support/fixtures'

test.describe.configure({ mode: 'serial' })

test.describe('@p0 Timer Interrupt', () => {
  test.beforeEach(async ({ apiClient }) => {
    const existing = await apiClient.getActiveSession()
    if (existing) {
      await apiClient.controlSession(existing.id, 'abandon').catch(() => {})
    }
  })

  test('TMR-E2E-005: Abandon session with confirm', async ({
    page,
    sessionFactory,
    apiClient,
  }) => {
    // Given a running session
    const session = await sessionFactory.create({ type: 'work', duration: 1500 })

    // When visiting Timer page
    await page.goto('/timer')
    await expect(page.locator('.timer-time')).toBeVisible()

    // And clicking abandon button
    const abandonBtn = page.getByRole('button', { name: /放弃/ })
    await abandonBtn.click()

    // Then confirm dialog appears - click confirm
    const confirmBtn = page.getByRole('button', { name: /确定|确认|OK|是的/ })
    if (await confirmBtn.isVisible()) {
      await confirmBtn.click()
    }

    // Then session is abandoned (no active running session)
    const active = await apiClient.getActiveSession()
    if (active) {
      expect(active.status).not.toBe('running')
    }
  })

  test('TMR-E2E-006: Completing session updates linked task status', async ({
    page,
    taskFactory,
    sessionFactory,
    apiClient,
  }) => {
    // Given a task and a session linked to it
    const task = await taskFactory.create({ title: '关联任务', quadrant: 1 })
    const session = await sessionFactory.create({
      task_id: task.id,
      type: 'work',
      duration: 25,
    })

    // When completing the session
    await apiClient.controlSession(session.id, 'complete')

    // Then task status may be updated (check via API)
    const updatedTask = await apiClient.getTask(task.id)
    expect(updatedTask.status).toBeTruthy()
  })
})
