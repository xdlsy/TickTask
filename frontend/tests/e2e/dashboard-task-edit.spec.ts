import { test, expect } from '../support/fixtures'

/**
 * 回归守门:在「仪表盘」点击任务的「编辑」必须弹出编辑任务弹窗。
 *
 * 该入口此前在 E2E 套件中从未被覆盖,曾出现「点击编辑无任何反应」的致命回归。
 * 本用例全程使用真实浏览器点击(创建任务 → 仪表盘点编辑),不调用业务接口做断言,
 * 只把「编辑弹窗是否出现」作为唯一判定,确保 UI 行为契约被守住。
 */
test.describe('@p0 Dashboard · 任务编辑入口', () => {
  test('DASH-EDIT-E2E-001: 仪表盘点「编辑」应弹出编辑任务弹窗', async ({ page, apiClient }) => {
    const uniqueTitle = `E2E-编辑拦截-${Date.now()}`

    // UI 创建的任务不会进 taskFactory 的清理清单,用 apiClient 兜底回收,避免污染开发库
    const cleanup = async () => {
      const tasks = await apiClient.getTasks().catch(() => [])
      const mine = tasks.find((t) => t.title === uniqueTitle)
      if (mine) await apiClient.deleteTask(mine.id).catch(() => {})
    }

    try {
      // Given:在前台任务页用真实点击创建一个任务(打开创建弹窗 → 填标题 → 保存)
      await page.goto('/tasks')
      await page.getByRole('button', { name: /添加任务/ }).click()

      const createDialog = page.locator('.el-dialog', { hasText: '创建任务' })
      await expect(createDialog).toBeVisible()
      await createDialog.getByPlaceholder('输入任务标题').fill(uniqueTitle)
      await createDialog.getByRole('button', { name: '保存' }).click()
      // 创建成功后,任务出现在任务页四象限视图中
      await expect(page.getByText(uniqueTitle).first()).toBeVisible({ timeout: 10000 })

      // When:进入仪表盘,在「最近任务」里展开该任务的下拉菜单并点击「编辑」
      await page.goto('/')
      await expect(page.getByText('最近任务')).toBeVisible()
      const card = page.locator('.task-card', { hasText: uniqueTitle })
      await expect(card).toBeVisible({ timeout: 10000 })
      await card.locator('.more-icon').click()
      // 仪表盘上每张 TaskCard 的 el-dropdown 菜单都会被 Element Plus 渲染进 DOM(其余隐藏),
      // 故必须用 :visible 限定到当前展开的那一个,避免 strict mode 命中多个。
      await page.locator('.el-dropdown-menu__item:visible', { hasText: '编辑' }).click()

      // Then:弹出「编辑任务」弹窗(存在回归时此处会超时失败 → 拦截成功)
      await expect(page.locator('.el-dialog__title', { hasText: '编辑任务' })).toBeVisible({
        timeout: 10000,
      })
    } finally {
      await cleanup()
    }
  })

  test('DASH-EDIT-E2E-002: 仪表盘点「完成」将任务标记为已完成', async ({ page, taskFactory }) => {
    const title = `看板完成-${Date.now()}`
    const task = await taskFactory.create({ title, quadrant: 2 })

    await page.goto('/')
    const card = page.locator('.task-card', { hasText: title })
    await expect(card).toBeVisible({ timeout: 10000 })
    await card.locator('.more-icon').click()
    await page.locator('.el-dropdown-menu__item:visible', { hasText: '完成' }).click()

    // 卡片转为 completed 态(透明/删除线)
    await expect(card).toHaveClass(/task-completed/, { timeout: 10000 })
    await expect(page.locator('.el-message--success', { hasText: '已完成' })).toBeVisible({ timeout: 5000 })
    expect(task.id).toBeTruthy()
  })

  test('DASH-EDIT-E2E-003: 仪表盘点「删除」移除任务卡片', async ({ page, taskFactory }) => {
    const title = `看板删除-${Date.now()}`
    await taskFactory.create({ title, quadrant: 3 })

    await page.goto('/')
    const card = page.locator('.task-card', { hasText: title })
    await expect(card).toBeVisible({ timeout: 10000 })
    await card.locator('.more-icon').click()
    await page.locator('.el-dropdown-menu__item:visible', { hasText: '删除' }).click()

    await expect(page.locator('.task-card', { hasText: title })).toHaveCount(0, { timeout: 10000 })
  })
})
