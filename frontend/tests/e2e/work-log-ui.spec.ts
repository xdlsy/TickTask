import { test, expect } from '../support/fixtures'
import type { Page } from '@playwright/test'

const PREFIX = 'E2E-'

async function pickTime(page: Page, dataTest: string, time: string) {
  await page.locator(`[data-test="${dataTest}"]`).click()

  // 等待下拉菜单出现
  await page.waitForSelector('.el-select-dropdown__item:visible', { timeout: 5000 })

  // 等待一下，确保动画完成
  await page.waitForTimeout(200)

  // 直接使用 CSS 选择器找到包含指定文本但不是 disabled 的选项
  const timeOption = page.locator(`.el-select-dropdown__item:not(.is-disabled):visible`).filter({ hasText: time }).first()

  await timeOption.click()
}

async function addItem(page: Page, activity: string) {
  await page.goto('/work-log')

  // 等待页面完全加载，等待 WorkLog 页面的标志性元素出现
  await page.waitForSelector('.work-log-page', { timeout: 10000 })

  // 等待 WorkItemForm 组件渲染完成
  await page.waitForSelector('[data-test="activity-input"]', { timeout: 10000 })

  // data-test 属性直接就在 input 元素上，不需要再找内部的 input
  await page.locator('[data-test="activity-input"]').fill(activity)
  await pickTime(page, 'start-input', '09:00')
  await pickTime(page, 'end-input', '10:00')
  await page.locator('[data-test="submit-btn"]').click()
  await expect(page.getByText(activity).first()).toBeVisible({ timeout: 10000 })
}

async function cleanup(page: Page) {
  await page.goto('/work-log')
  let remaining = page.locator('.el-table__row').filter({ hasText: PREFIX })
  while ((await remaining.count()) > 0) {
    const row = remaining.first()
    await row.locator('[data-test="delete-btn"]').click()
    await page.getByRole('button', { name: /确定删除|确定|OK/ }).click()
    await page.waitForTimeout(300)
    remaining = page.locator('.el-table__row').filter({ hasText: PREFIX })
  }
}

test.describe('@p1 WorkLog · 手动条目流', () => {
  test('WL-UI-001: WorkItemForm 录入条目 → 出现在 TodayPanorama', async ({ page }) => {
    const activity = `${PREFIX}录入-${Date.now()}`
    try {
      await addItem(page, activity)
      await expect(page.locator('.el-table__row').filter({ hasText: activity })).toBeVisible({ timeout: 10000 })
    } finally {
      await cleanup(page)
    }
  })

  test('WL-UI-002: TodayPanorama「编辑」改 activity → 保存生效', async ({ page }) => {
    const original = `${PREFIX}改前-${Date.now()}`
    const changed = `${original}-改`
    try {
      await addItem(page, original)
      const row = page.locator('.el-table__row').filter({ hasText: original })
      await row.locator('[data-test="edit-btn"]').click()

      // data-test 属性直接就在 input 元素上
      await page.locator('[data-test="activity-input"]').fill(changed)
      await page.locator('[data-test="submit-btn"]').click()

      await expect(page.getByText(changed).first()).toBeVisible({ timeout: 10000 })
    } finally {
      await cleanup(page)
    }
  })

  test('WL-UI-003: TodayPanorama「删除」+ 确认 → 条目消失', async ({ page }) => {
    const activity = `${PREFIX}删-${Date.now()}`
    await addItem(page, activity)

    const row = page.locator('.el-table__row').filter({ hasText: activity })
    await row.locator('[data-test="delete-btn"]').click()

    // el-popconfirm 确认按钮是 "Yes"
    await page.locator('.el-popconfirm').getByRole('button', { name: 'Yes' }).click()

    await expect(page.getByText(activity)).toHaveCount(0, { timeout: 10000 })
  })

  test('WL-UI-004: 生成「本周周报」→ ReportDetail 展示', async ({ page }) => {
    const activity = `${PREFIX}周报-${Date.now()}`
    try {
      await addItem(page, activity)

      // 点击生成报告按钮
      await page.locator('.action-btn', { hasText: '生成报告' }).click()

      // 点击"本周周报"选项
      await page.locator('.el-dropdown-menu__item:visible', { hasText: '本周周报' }).click()

      // 处理可能的覆盖确认对话框（如果报告已存在）
      await page.waitForTimeout(1000) // 等待对话框出现
      const messageBoxCount = await page.locator('.el-message-box').count()

      if (messageBoxCount > 0) {
        // 点击"覆盖"按钮
        await page.locator('.el-message-box').getByRole('button', { name: '覆盖' }).click()
      }

      // 等待报告生成完成 - 首先等待 rd-title 元素出现
      await page.waitForSelector('.rd-title', { timeout: 20000 })

      // 检查报告标题是否包含"周报"
      await expect(page.locator('.rd-title').first()).toContainText('周报', { timeout: 5000 })
    } finally {
      await cleanup(page)
    }
  })
})
