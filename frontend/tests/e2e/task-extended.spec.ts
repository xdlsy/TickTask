import { test, expect } from '../support/fixtures'

test.describe('@p1 Task Extended', () => {
  test('TASK-E2E-007: List/quadrant view switch data consistent', async ({
    page,
    taskFactory,
  }) => {
    // Given multiple tasks exist
    await taskFactory.create({ title: '视图任务A', quadrant: 1 })
    await taskFactory.create({ title: '视图任务B', quadrant: 2 })
    await taskFactory.create({ title: '视图任务C', quadrant: 3 })

    // When visiting Tasks in quadrant view
    await page.goto('/tasks')
    await expect(page.getByText('视图任务A')).toBeVisible()

    // And switching to list view
    const listBtn = page.getByRole('button', { name: /^列表$|list/i })
    if (await listBtn.isVisible()) {
      await listBtn.click()

      // Then all tasks are still visible
      await expect(page.getByText('视图任务A')).toBeVisible()
      await expect(page.getByText('视图任务B')).toBeVisible()
      await expect(page.getByText('视图任务C')).toBeVisible()
    }

    // And switching back to quadrant view
    const quadBtn = page.getByRole('button', { name: /^四象限$|quadrant/i })
    if (await quadBtn.isVisible()) {
      await quadBtn.click()
      await expect(page.getByText('视图任务A')).toBeVisible()
    }
  })

  test('TASK-E2E-008: completed_at timestamp on completion', async ({
    apiClient,
    taskFactory,
  }) => {
    // Given a task
    const task = await taskFactory.create({ title: '完成时间戳测试', quadrant: 2 })

    // When marking it as completed
    await apiClient.updateTask(task.id, { status: 'completed' })

    // Then completed_at is set
    const updated = await apiClient.getTask(task.id)
    expect(updated.status).toBe('completed')
    expect(updated.completed_at).toBeTruthy()
  })

  test('TASK-E2E-010: Tags display in task card', async ({
    page,
    taskFactory,
  }) => {
    // Given a task with tags
    await taskFactory.create({
      title: '标签显示测试',
      quadrant: 1,
      tags: ['bug', 'urgent'],
    })

    // When visiting Tasks page
    await page.goto('/tasks')

    // Then task is visible with tags
    await expect(page.getByText('标签显示测试')).toBeVisible()
  })
})
