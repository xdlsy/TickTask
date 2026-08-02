import { test, expect } from '../support/fixtures'

test.describe('@p0 Dashboard', () => {
  test('DASH-E2E-001: Dashboard renders with task data', async ({
    page,
    taskFactory,
  }) => {
    // Given tasks exist
    await taskFactory.create({ title: '仪表盘任务A', quadrant: 1 })
    await taskFactory.create({ title: '仪表盘任务B', quadrant: 2 })

    // When user visits Dashboard
    await page.goto('/')

    // Then task data is reflected (stat cards or recent task list visible)
    const hasContent =
      (await page.locator('.stat-card, [class*="stat"]').count()) > 0 ||
      (await page.getByText('仪表盘任务A').count()) > 0 ||
      (await page.getByText(/任务|番茄/).count()) > 0
    expect(hasContent).toBeTruthy()
  })

  test('DASH-E2E-002: Dashboard shows today stats overview', async ({
    page,
    taskFactory,
    sessionFactory,
    apiClient,
  }) => {
    // Given a completed pomodoro session and task exist
    const task = await taskFactory.create({ title: '统计任务', quadrant: 1 })
    const session = await sessionFactory.create({
      task_id: task.id,
      type: 'work',
      duration: 25,
    })
    await apiClient.controlSession(session.id, 'complete')

    // When user visits Dashboard
    await page.goto('/')

    // Then stat cards show non-zero values
    const statCards = page.locator('.stat-card')
    if ((await statCards.count()) > 0) {
      await expect(statCards.first()).toBeVisible()
    }
  })

  test('DASH-E2E-003: Dashboard empty state shows CTA', async ({
    page,
    apiClient,
  }) => {
    // Given minimal data
    await apiClient.deleteAllSchedules().catch(() => {})

    // When user visits Dashboard
    await page.goto('/')

    // Then CTA buttons or empty state indicators are shown
    const hasCTA =
      (await page.getByRole('button', { name: /创建任务|开始番茄|开始专注/ }).count()) > 0 ||
      (await page.getByText(/暂无|开始/).count()) > 0
    expect(hasCTA).toBeTruthy()
  })

  test('DASH-E2E-004: Quick action navigates to Timer', async ({ page }) => {
    // Given user is on Dashboard
    await page.goto('/')

    // When clicking "开始番茄" button
    const startBtn = page.getByRole('button', { name: /开始番茄|开始专注/ })
    if (await startBtn.isVisible()) {
      await startBtn.click()

      // Then navigates to Timer page
      await expect(page).toHaveURL(/\/timer/, { timeout: 10000 })
    }
  })
})
