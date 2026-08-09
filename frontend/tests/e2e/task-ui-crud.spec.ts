import { test, expect } from '../support/fixtures'

test.describe('@p0 Tasks · UI 创建/编辑/删除/完成', () => {
  test('TASK-UI-001: 点「添加任务」填表保存,任务出现在四象限视图', async ({ page, apiClient }) => {
    const title = `UI创建-${Date.now()}`
    const cleanup = async () => {
      const tasks = await apiClient.getTasks().catch(() => [])
      const mine = tasks.find((t) => t.title === title)
      if (mine) await apiClient.deleteTask(mine.id).catch(() => {})
    }
    try {
      await page.goto('/tasks')
      await page.getByRole('button', { name: /添加任务/ }).click()

      const dialog = page.locator('.el-dialog', { hasText: '创建任务' })
      await expect(dialog).toBeVisible()
      await dialog.getByPlaceholder('输入任务标题').fill(title)
      await dialog.getByRole('button', { name: '保存' }).click()

      await expect(page.locator('.el-dialog', { hasText: '创建任务' })).toBeHidden({ timeout: 10000 })
      await expect(page.getByText(title).first()).toBeVisible({ timeout: 10000 })
    } finally {
      await cleanup()
    }
  })

  test('TASK-UI-002: 列表视图下拉「编辑」改标题并保存生效', async ({ page, taskFactory }) => {
    const original = `待编辑-${Date.now()}`
    await taskFactory.create({ title: original, quadrant: 2 })
    const changed = `${original}-改`

    await page.goto('/tasks')
    await page.getByRole('button', { name: '列表' }).click()

    const item = page.locator('.task-item', { hasText: original })
    await expect(item).toBeVisible({ timeout: 10000 })
    await item.locator('.action-btn').click()
    await page.locator('.el-dropdown-menu__item:visible', { hasText: '编辑' }).click()

    const dialog = page.locator('.el-dialog', { hasText: '编辑任务' })
    await expect(dialog).toBeVisible()

    await dialog.getByPlaceholder('输入任务标题').fill(changed)
    await dialog.getByRole('button', { name: '保存' }).click()

    await expect(dialog).toBeHidden({ timeout: 10000 })
    await expect(page.getByText(changed).first()).toBeVisible({ timeout: 10000 })
  })

  test('TASK-UI-003: 列表视图下拉「删除」移除任务(无确认弹窗)', async ({ page, taskFactory }) => {
    const title = `待删除-${Date.now()}`
    await taskFactory.create({ title, quadrant: 3 })

    await page.goto('/tasks')
    await page.getByRole('button', { name: '列表' }).click()

    const item = page.locator('.task-item', { hasText: title })
    await expect(item).toBeVisible({ timeout: 10000 })
    await item.locator('.action-btn').click()
    await page.locator('.el-dropdown-menu__item:visible', { hasText: '删除' }).click()

    await expect(page.getByText(title)).toHaveCount(0, { timeout: 10000 })
  })

  test('TASK-UI-004: 列表视图下拉「标记完成」切换到已完成态', async ({ page, taskFactory }) => {
    const title = `待完成-${Date.now()}`
    await taskFactory.create({ title, quadrant: 2 })

    await page.goto('/tasks')
    await page.getByRole('button', { name: '列表' }).click()

    const item = page.locator('.task-item', { hasText: title })
    await expect(item).toBeVisible({ timeout: 10000 })
    await item.locator('.action-btn').click()
    await page.locator('.el-dropdown-menu__item:visible', { hasText: '标记完成' }).click()

    await expect(item).toHaveClass(/completed/, { timeout: 10000 })
  })
})
