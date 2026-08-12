import { test, expect } from '../support/fixtures'

test.describe('@p2 Analytics · 图表 tooltip', () => {
  test('ANALY-UI-001: hover 趋势柱显时长 tooltip', async ({ page, taskFactory, sessionFactory, apiClient }) => {
    const task = await taskFactory.create({ title: `图表-${Date.now()}`, quadrant: 2 })
    const session = await sessionFactory.create({ task_id: task.id, type: 'work', duration: 25 })
    await apiClient.controlSession(session.id, 'complete')

    await page.goto('/analytics')
    const bar = page.locator('.chart-bar-wrapper').first()
    await expect(bar).toBeVisible({ timeout: 15000 })

    await bar.hover()
    await expect(page.locator('.bar-tooltip').first()).not.toBeEmpty({ timeout: 5000 })
  })
})
