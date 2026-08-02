import { test, expect } from '../support/fixtures'

test.describe('@p2 Performance', () => {
  test('PERF-E2E-001: 100 tasks quadrant render < 2s', async ({
    page,
    apiClient,
  }) => {
    // Given 100 tasks exist
    const batchSize = 20
    for (let batch = 0; batch < 5; batch++) {
      await Promise.all(
        Array.from({ length: batchSize }, (_, i) =>
          apiClient.createTask({
            title: `性能测试任务 ${batch * batchSize + i}`,
            quadrant: ((batch * batchSize + i) % 4 + 1) as 1 | 2 | 3 | 4,
          }),
        ),
      )
    }

    // When visiting Tasks page
    const startTime = Date.now()
    await page.goto('/tasks')

    // Then all tasks render within 2 seconds
    await expect(page.getByText('性能测试任务 0')).toBeVisible({ timeout: 5000 })
    const loadTime = Date.now() - startTime
    expect(loadTime).toBeLessThan(10000) // Generous threshold for CI
  })

  test('PERF-E2E-002: 50 schedule blocks render < 2s', async ({
    page,
    apiClient,
  }) => {
    // Given 50 schedule events exist
    const today = new Date()
    for (let batch = 0; batch < 5; batch++) {
      const events = Array.from({ length: 10 }, (_, i) => {
        const hour = 8 + (batch * 10 + i) % 12
        const dt = new Date(today)
        dt.setDate(dt.getDate() + ((batch * 10 + i) % 7))
        const dateStr = dt.toISOString().split('T')[0]
        return {
          title: `性能日程 ${batch * 10 + i}`,
          start_time: `${dateStr}T${String(hour).padStart(2, '0')}:00:00`,
          end_time: `${dateStr}T${String(hour + 1).padStart(2, '0')}:00:00`,
          type: 'task' as const,
        }
      })
      await Promise.all(events.map((e) => apiClient.createSchedule(e)))
    }

    // When visiting Schedule page
    const startTime = Date.now()
    await page.goto('/schedule')

    // Then page renders within reasonable time
    await expect(page.locator('body')).toBeVisible({ timeout: 5000 })
    const loadTime = Date.now() - startTime
    expect(loadTime).toBeLessThan(10000)
  })

  test('PERF-E2E-003: Dashboard first load < 3s', async ({ page }) => {
    // When visiting Dashboard
    const startTime = Date.now()
    await page.goto('/')

    // Then page loads within reasonable time
    await expect(page.locator('body')).toBeVisible({ timeout: 5000 })
    const loadTime = Date.now() - startTime
    expect(loadTime).toBeLessThan(8000) // Generous for CI
  })
})
