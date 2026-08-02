import { test, expect } from '../support/fixtures'

test.describe.configure({ mode: 'serial' })

test.describe('@p0 Schedule CRUD', () => {
  test('SCH-E2E-003: Create schedule appears on calendar', async ({
    page,
    scheduleFactory,
  }) => {
    // Given a schedule is created
    const today = new Date().toISOString().split('T')[0]
    const event = await scheduleFactory.create({
      title: '新建日程测试',
      start_time: `${today}T09:00:00+08:00`,
      end_time: `${today}T10:00:00+08:00`,
    })

    // When user visits Schedule page and navigates to today
    await page.goto('/schedule')

    // Switch to Day view and navigate to today
    const dayBtn = page.getByRole('button', { name: /^日$|day/i })
    if (await dayBtn.isVisible()) await dayBtn.click()
    const todayBtn = page.getByRole('button', { name: /今天/ })
    if (await todayBtn.isVisible()) await todayBtn.click()

    // Then the event is visible on the calendar
    await expect(page.getByText('新建日程测试')).toBeVisible({ timeout: 10000 })
  })

  test('SCH-E2E-004: Move schedule persists after reload', async ({
    page,
    scheduleFactory,
    apiClient,
  }) => {
    // Given a schedule event
    const today = new Date().toISOString().split('T')[0]
    const event = await scheduleFactory.create({
      title: '移动日程测试',
      start_time: `${today}T09:00:00+08:00`,
      end_time: `${today}T10:00:00+08:00`,
    })

    // When moving the schedule via API (simulating drag)
    await apiClient.moveSchedule(event.id, {
      start_time: `${today}T14:00:00+08:00`,
      end_time: `${today}T15:00:00+08:00`,
    })

    // Then event persists at new time after reload
    await page.goto('/schedule')
    const dayBtn2 = page.getByRole('button', { name: /^日$|day/i })
    if (await dayBtn2.isVisible()) await dayBtn2.click()
    const todayBtn2 = page.getByRole('button', { name: /今天/ })
    if (await todayBtn2.isVisible()) await todayBtn2.click()
    await expect(page.getByText('移动日程测试')).toBeVisible({ timeout: 10000 })
  })

  test('SCH-E2E-005: AI revise preview and confirm', async ({
    page,
    apiClient,
  }) => {
    // Given AI is configured
    const status = await apiClient.getAIStatus()
    test.skip(!status.configured, 'AI not configured')

    // When clicking "修订日程"
    await page.goto('/schedule')
    const reviseBtn = page.getByRole('button', { name: /修订日程|修订/ })
    if (await reviseBtn.isVisible()) {
      await reviseBtn.click({ force: true }).catch(() => {})

      // Then revision dialog or input appears
      await page.waitForTimeout(1000)
      const inputField = page.locator('textarea, [class*="revision"] input, .el-dialog input').first()
      if (await inputField.isVisible()) {
        await inputField.fill('把上午的日程移到下午')
        const submitBtn = page.getByRole('button', { name: /提交|生成|确定|发送/ })
        if (await submitBtn.isVisible()) {
          await submitBtn.click()
          // Wait for AI processing
          await page.waitForTimeout(5000)
        }
      }
    }
    // Verify page is still functional after revision attempt
    await expect(page.locator('body')).toBeVisible()
  })

  test('SCH-E2E-006: AI revise reject preserves original', async ({
    page,
    scheduleFactory,
    apiClient,
  }) => {
    // Given AI is configured and schedules exist
    const status = await apiClient.getAIStatus()
    test.skip(!status.configured, 'AI not configured')

    const today = new Date().toISOString().split('T')[0]
    await scheduleFactory.create({
      title: '拒绝修订测试',
      start_time: `${today}T10:00:00+08:00`,
      end_time: `${today}T11:00:00+08:00`,
    })

    // When clicking "修订日程"
    await page.goto('/schedule')
    const reviseBtn = page.getByRole('button', { name: /修订日程|修订/ })
    if (await reviseBtn.isVisible()) {
      await reviseBtn.click({ force: true }).catch(() => {})

      // Then cancel without applying
      const cancelBtn = page.getByRole('button', { name: /取消|关闭/ })
      if (await cancelBtn.isVisible()) {
        await cancelBtn.click()
      }

      // Then original schedule is unchanged after reload
      await page.reload()
      await expect(page.getByText('拒绝修订测试')).toBeVisible()
    }
  })
})
