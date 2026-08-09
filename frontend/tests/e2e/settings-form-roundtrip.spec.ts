import { test, expect } from '../support/fixtures'

test.describe('@p0 Settings · 表单往返', () => {
  test('SET-UI-001: 工作时长改动保存后重载仍生效', async ({ page }) => {
    await page.goto('/settings')

    const workItem = page.locator('.setting-item').filter({ hasText: '工作时长' })
    const workInput = workItem.locator('.el-input-number input')
    const before = (await workInput.inputValue().catch(() => '')) || '25'

    // 改工作时长为 30 并提交
    await workInput.fill('30')
    await workInput.press('Tab')
    await page.getByRole('button', { name: '保存设置' }).first().click()

    // 重载后值应持久
    await page.reload()
    const workInputAfter = page
      .locator('.setting-item')
      .filter({ hasText: '工作时长' })
      .locator('.el-input-number input')
    await expect(workInputAfter).toHaveValue('30', { timeout: 10000 })

    // 恢复原值,避免污染开发库设置
    await workInputAfter.fill(before)
    await workInputAfter.press('Tab')
    await page.getByRole('button', { name: '保存设置' }).first().click()
  })
})
