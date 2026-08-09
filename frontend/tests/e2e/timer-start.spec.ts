import { test, expect } from '../support/fixtures'

test.describe('@p0 Timer · 启动', () => {
  test('TMR-UI-001: 点「开始专注」进入运行态(暂停按钮出现)', async ({ page }) => {
    await page.goto('/timer')

    // 等待定时器状态加载;若存在遗留的进行中会话,先放弃以确保从干净态开始
    await expect(page.locator('.timer-time')).toBeVisible({ timeout: 10000 })
    const abandon = page.getByRole('button', { name: /^放弃/ })
    if (await abandon.isVisible()) {
      await abandon.click()
      await page.getByRole('button', { name: /确定|确认|OK|是的/ }).click()
      await expect(page.getByRole('button', { name: /开始专注/ })).toBeVisible({ timeout: 10000 })
    }

    // When:点「开始专注」
    await page.getByRole('button', { name: /开始专注/ }).click()

    // Then:进入运行态 —— 暂停按钮出现
    await expect(page.getByRole('button', { name: /^暂停/ })).toBeVisible({ timeout: 10000 })

    // 清理:放弃本次会话,避免遗留 active session 污染后续用例
    await page.getByRole('button', { name: /^放弃/ }).click()
    await page.getByRole('button', { name: /确定|确认|OK|是的/ }).click()
  })
})
