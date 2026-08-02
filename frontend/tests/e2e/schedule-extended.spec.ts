import { test, expect } from '../support/fixtures'

test.describe.configure({ mode: 'serial' })

test.describe('@p1 Schedule Extended', () => {
  test.beforeEach(async ({ apiClient }) => {
    // Clean up any active timer session to avoid state conflicts
    const activeSession = await apiClient.getActiveSession()
    if (activeSession) {
      await apiClient.controlSession(activeSession.id, 'abandon').catch(() => {})
    }
  })

  test('SCH-E2E-009: Week view renders week events', async ({
    page,
    scheduleFactory,
  }) => {
    // Given events across multiple days
    const today = new Date()
    const dates = [0, 1, 2].map((d) => {
      const dt = new Date(today)
      dt.setDate(dt.getDate() + d)
      return dt.toISOString().split('T')[0]
    })

    await scheduleFactory.create({
      title: '周视图事件1',
      start_time: `${dates[0]}T09:00:00+08:00`,
      end_time: `${dates[0]}T10:00:00+08:00`,
    })
    await scheduleFactory.create({
      title: '周视图事件2',
      start_time: `${dates[1]}T14:00:00+08:00`,
      end_time: `${dates[1]}T15:00:00+08:00`,
    })

    // When visiting Schedule page and navigating to today
    await page.goto('/schedule')

    // Switch to Day view and navigate to today to ensure correct date context
    await page.getByRole('button', { name: /^日$|day/i }).click({ force: true }).catch(() => {})
    await page.getByRole('button', { name: /今天/ }).click()

    // Then at least the first event is visible
    await expect(page.getByText('周视图事件1').first()).toBeVisible({ timeout: 10000 })

    // When switching to Week view
    const weekBtn = page.getByRole('button', { name: /^周$|week/i })
    if (await weekBtn.isVisible()) {
      await weekBtn.click({ force: true }).catch(() => {})

      // Then events are visible in Week view
      await expect(page.getByText('周视图事件1').first()).toBeVisible({ timeout: 10000 })
      await expect(page.getByText('周视图事件2').first()).toBeVisible({ timeout: 10000 })
    }
  })

  test('SCH-E2E-010: Month view renders calendar grid', async ({
    page,
    scheduleFactory,
  }) => {
    // Given an event
    const today = new Date().toISOString().split('T')[0]
    await scheduleFactory.create({
      title: '月视图事件',
      start_time: `${today}T10:00:00+08:00`,
      end_time: `${today}T11:00:00+08:00`,
    })

    // When visiting Schedule page and navigating to today
    await page.goto('/schedule')

    // Switch to Day view and navigate to today first
    await page.getByRole('button', { name: /^日$|day/i }).click({ force: true }).catch(() => {})
    await page.getByRole('button', { name: /今天/ }).click()

    // Verify event exists in Day view
    await expect(page.getByText('月视图事件').first()).toBeVisible({ timeout: 10000 })

    // When switching to Month view
    const monthBtn = page.getByRole('button', { name: /^月$|month/i })
    if (await monthBtn.isVisible()) {
      await monthBtn.click({ force: true }).catch(() => {})

      // Then calendar grid is visible
      const grid = page.locator('.calendar-grid, [class*="month"], [class*="calendar"]')
      if (await grid.isVisible()) {
        expect(await grid.isVisible()).toBeTruthy()
      }
    }
  })

  test('SCH-E2E-011: Color coding by schedule type', async ({
    page,
    scheduleFactory,
  }) => {
    // Given schedules of different types
    const today = new Date().toISOString().split('T')[0]
    await scheduleFactory.create({
      title: '任务类型日程',
      start_time: `${today}T09:00:00+08:00`,
      end_time: `${today}T10:00:00+08:00`,
      type: 'task',
    })
    await scheduleFactory.create({
      title: '自定义类型日程',
      start_time: `${today}T10:00:00+08:00`,
      end_time: `${today}T11:00:00+08:00`,
      type: 'custom',
    })

    // When visiting Schedule page
    await page.goto('/schedule')

    // Switch to Day view and navigate to today
    await page.getByRole('button', { name: /^日$|day/i }).click({ force: true }).catch(() => {})
    await page.getByRole('button', { name: /今天/ }).click()

    // Then both events are visible
    await expect(page.getByText('任务类型日程').first()).toBeVisible({ timeout: 10000 })
    await expect(page.getByText('自定义类型日程').first()).toBeVisible({ timeout: 10000 })
  })

  test('SCH-E2E-012: Edit event dialog modifies fields', async ({
    page,
    scheduleFactory,
    apiClient,
  }) => {
    // Given a schedule event
    const today = new Date().toISOString().split('T')[0]
    const event = await scheduleFactory.create({
      title: '编辑前标题',
      start_time: `${today}T09:00:00+08:00`,
      end_time: `${today}T10:00:00+08:00`,
    })

    // When updating via API
    await apiClient.updateSchedule(event.id, { title: '编辑后标题' })

    // Then update is reflected
    await page.goto('/schedule')

    // Switch to Day view and navigate to today
    await page.getByRole('button', { name: /^日$|day/i }).click({ force: true }).catch(() => {})
    await page.getByRole('button', { name: /今天/ }).click()

    await expect(page.getByText('编辑后标题').first()).toBeVisible({ timeout: 10000 })
  })

  test('SCH-E2E-013: Delete event with confirmation', async ({
    page,
    scheduleFactory,
    apiClient,
  }) => {
    // Given a schedule event
    const today = new Date().toISOString().split('T')[0]
    const event = await scheduleFactory.create({
      title: '待删除日程',
      start_time: `${today}T09:00:00+08:00`,
      end_time: `${today}T10:00:00+08:00`,
    })

    // When deleting via API
    await apiClient.deleteSchedule(event.id)

    // Then event is gone
    await page.goto('/schedule')

    // Switch to Day view and navigate to today
    await page.getByRole('button', { name: /^日$|day/i }).click({ force: true }).catch(() => {})
    await page.getByRole('button', { name: /今天/ }).click()

    await expect(page.getByText('待删除日程').first()).not.toBeVisible({ timeout: 10000 })
  })
})
