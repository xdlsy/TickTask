import { test, expect } from '../support/fixtures'
import type { Page } from '@playwright/test'

async function selectOption(page: Page, selectNth: number, label: string) {
  await page.locator('.el-select').nth(selectNth).click()
  await page.locator('.el-select-dropdown__item:visible', { hasText: label }).click()
}

test.describe('@p2 Tasks · 列表筛选与排序', () => {
  test('LIST-UI-001: 状态筛选「已完成」只显示已完成任务', async ({ page, taskFactory, apiClient }) => {
    const todo = `筛A-${Date.now()}`
    const done = `筛B-${Date.now()}`
    await taskFactory.create({ title: todo, quadrant: 2, status: 'todo' })
    const d = await taskFactory.create({ title: done, quadrant: 2 })
    await apiClient.updateTask(d.id, { status: 'completed' }).catch(() => {})

    await page.goto('/tasks')
    await page.getByRole('button', { name: '列表' }).click()

    // 状态筛选是第 0 个 el-select
    await selectOption(page, 0, '已完成')
    await expect(page.locator('.task-item', { hasText: done })).toBeVisible({ timeout: 10000 })
    await expect(page.locator('.task-item', { hasText: todo })).toHaveCount(0)
  })

  test('LIST-UI-002: 切换排序不报错且列表仍渲染', async ({ page, taskFactory }) => {
    await taskFactory.create({ title: `排-${Date.now()}`, quadrant: 1 })
    await taskFactory.create({ title: `排-${Date.now()}-2`, quadrant: 3 })

    await page.goto('/tasks')
    await page.getByRole('button', { name: '列表' }).click()
    await expect(page.locator('.task-item').first()).toBeVisible({ timeout: 10000 })

    // 排序是第 2 个 el-select(状态/象限/排序)
    await selectOption(page, 2, '优先级')
    await expect(page.locator('.task-item').first()).toBeVisible({ timeout: 10000 })
  })
})
