import { test, expect } from '../support/fixtures'

test.describe('@p2 Tasks · 拖拽跨象限', () => {
  test('DRAG-UI-001: 拖拽任务从 Q2 到 Q1', async ({ page, taskFactory }) => {
    const title = `拖拽-${Date.now()}`
    await taskFactory.create({ title, quadrant: 2 })

    await page.goto('/tasks') // 默认四象限
    const row = page.locator('.task-row', { hasText: title })
    await expect(row).toBeVisible({ timeout: 10000 })

    await row.dragTo(page.locator('.quad-q1'))

    await expect(page.locator('.quad-q1').locator('.task-row', { hasText: title })).toBeVisible({ timeout: 10000 })
    await expect(page.locator('.quad-q2').locator('.task-row', { hasText: title })).toHaveCount(0)
  })
})
