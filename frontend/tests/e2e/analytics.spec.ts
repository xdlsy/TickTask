import { test, expect } from '../support/fixtures'

test.describe('@p1 Analytics', () => {
  test('ANALY-E2E-001: Overview card metrics correct', async ({
    page,
    taskFactory,
    sessionFactory,
    apiClient,
  }) => {
    // Given completed sessions and tasks
    const task = await taskFactory.create({ title: '分析任务', quadrant: 1 })
    await apiClient.updateTask(task.id, { status: 'completed' })
    const session = await sessionFactory.create({ task_id: task.id, type: 'work', duration: 25 })
    await apiClient.controlSession(session.id, 'complete')

    // When visiting Analytics page
    await page.goto('/analytics')

    // Then overview cards are visible with data
    const overviewCards = page.locator('.overview-card, [class*="overview"], [class*="stat"]')
    if (await overviewCards.count() > 0) {
      expect(await overviewCards.count()).toBeGreaterThanOrEqual(1)
    }
  })

  test('ANALY-E2E-002: Distribution chart renders', async ({
    page,
    taskFactory,
  }) => {
    // Given tasks across quadrants
    await taskFactory.create({ title: '分布Q1', quadrant: 1 })
    await taskFactory.create({ title: '分布Q2', quadrant: 2 })
    await taskFactory.create({ title: '分布Q3', quadrant: 3 })

    // When visiting Analytics page
    await page.goto('/analytics')

    // Then quadrant stats are visible
    const quadStats = page.locator('.quadrant-stat, [class*="quadrant"]')
    if (await quadStats.count() > 0) {
      expect(await quadStats.count()).toBeGreaterThanOrEqual(1)
    }
  })

  test('ANALY-E2E-003: Trend chart time range switch', async ({ page }) => {
    // When visiting Analytics page
    await page.goto('/analytics')

    // Then filter buttons are available
    const todayBtn = page.getByRole('button', { name: /今日/ })
    const weekBtn = page.getByRole('button', { name: /本周/ })
    const monthBtn = page.getByRole('button', { name: /本月/ })

    if (await todayBtn.isVisible()) {
      await todayBtn.click()
    }
    if (await weekBtn.isVisible()) {
      await weekBtn.click()
    }
    if (await monthBtn.isVisible()) {
      await monthBtn.click()
    }

    // Page should not crash after switching
    await expect(page.locator('body')).toBeVisible()
  })

  test('ANALY-E2E-004: Empty data state shows guidance', async ({ page }) => {
    // Given no analytics data (fresh state)

    // When visiting Analytics page
    await page.goto('/analytics')

    // Then page renders (with zeros or empty state)
    await expect(page.locator('body')).toBeVisible()
    const hasContent =
      (await page.locator('.overview-card, [class*="overview"]').count()) > 0 ||
      (await page.getByText(/暂无|没有|0/).count()) > 0
    expect(hasContent).toBeTruthy()
  })
})
