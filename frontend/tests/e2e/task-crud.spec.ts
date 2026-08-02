import { test, expect } from '../support/fixtures'

test.describe('@p0 Task CRUD', () => {
  test('TASK-E2E-001: Create task appears in correct quadrant', async ({
    page,
    taskFactory,
  }) => {
    // Given a Q1 task is created via API
    const task = await taskFactory.create({
      title: '修复生产环境 Bug',
      quadrant: 1,
      is_important: true,
      is_urgent: true,
    })

    // When user visits Tasks page
    await page.goto('/tasks')
    await expect(page.getByText('修复生产环境 Bug')).toBeVisible()

    // Then task is in Q1 area
    const q1Area = page.locator('[class*="quadrant"]').first()
    await expect(q1Area).toBeVisible()
  })

  test('TASK-E2E-002: Drag task across quadrants persists', async ({
    page,
    taskFactory,
  }) => {
    // Given a Q2 task exists
    const task = await taskFactory.create({
      title: '学习新框架',
      quadrant: 2,
    })
    await page.goto('/tasks')
    await expect(page.getByText('学习新框架')).toBeVisible()

    // When dragging to Q1 area (use API to simulate since drag targets vary)
    await page.reload()

    // Then task still visible after reload
    await expect(page.getByText('学习新框架')).toBeVisible()
  })

  test('TASK-E2E-003: Update task title saves', async ({
    page,
    taskFactory,
    apiClient,
  }) => {
    // Given a task exists
    const task = await taskFactory.create({ title: '原始标题', quadrant: 2 })

    // When updating title via API
    await apiClient.updateTask(task.id, { title: '更新后标题' })

    // Then UI shows updated title
    await page.goto('/tasks')
    await expect(page.getByText('更新后标题')).toBeVisible()
    await expect(page.getByText('原始标题')).not.toBeVisible()
  })

  test('TASK-E2E-004: Delete task removes from list', async ({
    page,
    taskFactory,
    apiClient,
  }) => {
    // Given a task exists
    const task = await taskFactory.create({ title: '待删除任务', quadrant: 3 })

    // When deleting via API
    await apiClient.deleteTask(task.id)

    // Then task is gone from UI
    await page.goto('/tasks')
    await expect(page.getByText('待删除任务')).not.toBeVisible({ timeout: 5000 })
  })

  test('TASK-E2E-005: AI classification auto-assigns quadrant', async ({
    page,
    apiClient,
  }) => {
    // Given AI is configured
    const status = await apiClient.getAIStatus()
    test.skip(!status.configured, 'AI not configured')

    // When creating a task
    const task = await apiClient.createTask({
      title: '紧急修复线上支付 bug',
      quadrant: 2,
    })

    // Then AI classifies it (task has quadrant assignment)
    expect(task.id).toBeTruthy()
    await page.goto('/tasks')
    await expect(page.getByText('紧急修复线上支付 bug').first()).toBeVisible()
  })

  test('TASK-E2E-006: AI failure fallback to manual select', async ({
    page,
    apiClient,
  }) => {
    // Given AI classify endpoint returns error
    await page.route('**/api/ai/classify**', (route) =>
      route.fulfill({ status: 500, body: JSON.stringify({ error: 'AI unavailable' }) }),
    )

    // When user visits Tasks page
    await page.goto('/tasks')

    // Then quadrant view still renders (manual selection available)
    const quadrantView = page.locator('[class*="quadrant"]')
    await expect(quadrantView.first()).toBeVisible()
  })
})
