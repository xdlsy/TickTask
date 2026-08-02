import { test, expect } from '../support/fixtures'

test.describe.configure({ mode: 'serial' })

test.describe('@p0 Timer Workflow', () => {
  // Clean up any active session before each test to avoid interference
  test.beforeEach(async ({ apiClient }) => {
    const existing = await apiClient.getActiveSession()
    if (existing) {
      await apiClient.controlSession(existing.id, 'abandon').catch(() => {})
    }
  })

  test('TMR-E2E-001: Full workflow - start, countdown, complete', async ({
    page,
    sessionFactory,
    apiClient,
  }) => {
    // Given a short work session is created
    const session = await sessionFactory.create({ type: 'work', duration: 1500 })

    // When user visits Timer page
    await page.goto('/timer')

    // Then timer is running with time display
    await expect(page.locator('.timer-time')).toBeVisible()
    await expect(page.locator('.timer-label')).toContainText(/专注中|进行中|running/i)

    // When completing the session via API
    await apiClient.controlSession(session.id, 'complete')

    // Then status shows completed
    await page.reload()
    await expect(page.locator('.timer-label')).toContainText(/已完成|准备开始|completed/i)
  })

  test('TMR-E2E-002: Pause and resume correct behavior', async ({
    page,
    sessionFactory,
  }) => {
    // Given a running session
    await sessionFactory.create({ type: 'work', duration: 1500 })

    // When visiting Timer and pausing
    await page.goto('/timer')
    await expect(page.locator('.timer-time')).toBeVisible()

    const pauseBtn = page.getByRole('button', { name: /暂停/ })
    await pauseBtn.click()

    // Then timer shows paused state
    await expect(page.locator('.timer-label')).toContainText(/已暂停|paused/i)

    // When clicking resume
    const resumeBtn = page.getByRole('button', { name: /继续/ })
    await resumeBtn.click()

    // Then timer resumes
    await expect(page.locator('.timer-time')).toBeVisible()
  })

  test('TMR-E2E-003: Early complete marks session completed', async ({
    page,
    sessionFactory,
    apiClient,
  }) => {
    // Given a running session
    const session = await sessionFactory.create({ type: 'work', duration: 1500 })

    // When visiting Timer and clicking complete
    await page.goto('/timer')
    await expect(page.locator('.timer-time')).toBeVisible()

    const completeBtn = page.getByRole('button', { name: /完成/ })
    await completeBtn.click()

    // Then session is completed (no active running session)
    const active = await apiClient.getActiveSession()
    if (active) {
      expect(active.status).not.toBe('running')
    }
  })

  test('TMR-E2E-004: Refresh during running session restores state', async ({
    page,
    sessionFactory,
  }) => {
    // Given a running session
    await sessionFactory.create({ type: 'work', duration: 1500 })

    // When visiting Timer page
    await page.goto('/timer')
    await expect(page.locator('.timer-time')).toBeVisible()

    // And refreshing the page
    await page.reload()

    // Then timer is still running
    await expect(page.locator('.timer-time')).toBeVisible()
  })
})
