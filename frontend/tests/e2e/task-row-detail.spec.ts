import { test, expect } from '../support/fixtures'

test.describe('@p2 Tasks · row 点按详情', () => {
  test('ROW-UI-004: 点四象限 row 打开 TaskPomodoroDetail', async ({ page, taskFactory }) => {
    const title = `详情-${Date.now()}`
    await taskFactory.create({ title, quadrant: 2 })

    await page.goto('/tasks') // 默认四象限
    const row = page.locator('.task-row', { hasText: title })
    await expect(row).toBeVisible({ timeout: 10000 })

    await row.locator('.task-title').click()

    await expect(page.locator('.el-dialog__title', { hasText: title })).toBeVisible({ timeout: 10000 })
  })
})
