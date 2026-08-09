import { test, expect } from '../support/fixtures'

test.describe('@p1 Tasks · 四象限 row 菜单', () => {
  test('ROW-UI-001: row「...」→「编辑」打开 QuadrantView 编辑弹窗', async ({ page, taskFactory }) => {
    const title = `行菜单编辑-${Date.now()}`
    await taskFactory.create({ title, quadrant: 2 })

    await page.goto('/tasks') // 默认四象限
    const row = page.locator('.task-row', { hasText: title })
    await expect(row).toBeVisible({ timeout: 10000 })
    await row.locator('.row-more').click()
    await page.locator('.el-dropdown-menu__item:visible', { hasText: '编辑' }).click()

    await expect(page.locator('.el-dialog__title', { hasText: '编辑任务' })).toBeVisible({ timeout: 10000 })
  })

  test('ROW-UI-002: row 菜单「完成」切换到已完成态', async ({ page, taskFactory }) => {
    const title = `行菜单完成-${Date.now()}`
    await taskFactory.create({ title, quadrant: 2 })

    await page.goto('/tasks')
    const row = page.locator('.task-row', { hasText: title })
    await expect(row).toBeVisible({ timeout: 10000 })
    await row.locator('.row-more').click()
    await page.locator('.el-dropdown-menu__item:visible', { hasText: '完成' }).click()

    await expect(row).toHaveClass(/task-completed/, { timeout: 10000 })
    await expect(page.locator('.el-message--success', { hasText: '已完成' })).toBeVisible({ timeout: 5000 })
  })

  test('ROW-UI-003: row 菜单「删除」移除任务', async ({ page, taskFactory }) => {
    const title = `行菜单删除-${Date.now()}`
    await taskFactory.create({ title, quadrant: 3 })

    await page.goto('/tasks')
    const row = page.locator('.task-row', { hasText: title })
    await expect(row).toBeVisible({ timeout: 10000 })
    await row.locator('.row-more').click()
    await page.locator('.el-dropdown-menu__item:visible', { hasText: '删除' }).click()

    await expect(page.locator('.task-row', { hasText: title })).toHaveCount(0, { timeout: 10000 })
  })
})