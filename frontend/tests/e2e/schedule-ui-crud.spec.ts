import { test, expect } from '../support/fixtures'

test.describe('@p0 Schedule · UI 增删改', () => {
  test('SCH-UI-001: 点空槽打开「新建日程」表单,保存后日程出现', async ({ page, apiClient }) => {
    const title = `日程-${Date.now()}`
    const cleanup = async () => {
      const events = await apiClient.getSchedules().catch(() => [])
      const mine = events.find((e) => e.title === title)
      if (mine) await apiClient.deleteSchedule(mine.id).catch(() => {})
    }
    try {
      await page.goto('/schedule')
      await page.getByRole('button', { name: '日', exact: true }).click()
      await page.getByRole('button', { name: '今天', exact: true }).click()

      // 点第一个空槽打开新建表单
      await page.locator('.hour-slot').first().click()
      const dialog = page.locator('.el-dialog', { hasText: '新建日程' })
      await expect(dialog).toBeVisible()
      await dialog.getByPlaceholder('请输入日程标题').fill(title)
      await dialog.getByRole('button', { name: '保存' }).click()

      await expect(page.getByText(title).first()).toBeVisible({ timeout: 10000 })
    } finally {
      await cleanup()
    }
  })

  test('SCH-UI-002: 点事件块打开「编辑日程」,改标题保存生效', async ({ page, scheduleFactory }) => {
    const original = `原日程-${Date.now()}`
    const ev = await scheduleFactory.create({ title: original })
    const changed = `${original}-改`

    await page.goto('/schedule')
    await page.getByRole('button', { name: '日', exact: true }).click()
    await page.getByRole('button', { name: '今天', exact: true }).click()

    const block = page.locator('.event-block', { hasText: original }).first()
    await expect(block).toBeVisible({ timeout: 10000 })
    await block.click()

    const dialog = page.locator('.el-dialog', { hasText: '编辑日程' })
    await expect(dialog).toBeVisible()
    await dialog.getByPlaceholder('请输入日程标题').fill(changed)
    await dialog.getByRole('button', { name: '保存' }).click()

    await expect(page.getByText(changed).first()).toBeVisible({ timeout: 10000 })
    expect(ev.id).toBeTruthy()
  })

  test('SCH-UI-003: 编辑对话框内点「删除」移除日程', async ({ page, scheduleFactory }) => {
    const title = `待删日程-${Date.now()}`
    await scheduleFactory.create({ title })

    await page.goto('/schedule')
    await page.getByRole('button', { name: '日', exact: true }).click()
    await page.getByRole('button', { name: '今天', exact: true }).click()

    const block = page.locator('.event-block', { hasText: title }).first()
    await expect(block).toBeVisible({ timeout: 10000 })
    await block.click()

    const dialog = page.locator('.el-dialog', { hasText: '编辑日程' })
    await expect(dialog).toBeVisible()
    await dialog.getByRole('button', { name: '删除' }).click()

    await expect(page.getByText(title)).toHaveCount(0, { timeout: 10000 })
  })
})
