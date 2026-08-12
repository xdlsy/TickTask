import { test, expect } from '../support/fixtures'

test.describe('@p2 Settings · 导出 / 清空', () => {
  test('SET-UI-003: 点「导出全部数据」触发 JSON 下载', async ({ page }) => {
    await page.goto('/settings')
    const [download] = await Promise.all([
      page.waitForEvent('download'),
      page.locator('[data-test="export-btn"]').click(),
    ])
    expect(download.suggestedFilename()).toMatch(/\.json$/)
  })

  test('SET-UI-004: 点「清空全部数据」弹确认 → 取消,数据不清空', async ({ page, apiClient }) => {
    const before = await apiClient.getTasks()

    await page.goto('/settings')
    await page.locator('[data-test="clear-btn"]').click()

    const mb = page.locator('.el-message-box')
    await expect(mb).toBeVisible({ timeout: 5000 })
    // Close the dialog by clicking X or pressing Esc (proper cancel)
    await page.keyboard.press('Escape')
    await expect(mb).toBeHidden({ timeout: 5000 })

    const after = await apiClient.getTasks()
    expect(after.length).toBe(before.length)
  })
})