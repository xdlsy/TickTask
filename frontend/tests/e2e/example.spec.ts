import { test, expect } from '../support/fixtures'

test.describe('@smoke App Smoke Tests', () => {
  test('loads the dashboard page successfully', async ({ page }) => {
    // Given the app is running
    await page.goto('/')

    // When the page loads
    // Then the title is visible and the navigation works
    await expect(page).toHaveTitle(/TickTask/)
    await expect(page.getByRole('navigation')).toBeVisible()
  })

  test('navigates to Tasks page', async ({ page }) => {
    // Given the user is on the dashboard
    await page.goto('/')

    // When they click the 任务 nav link
    await page.getByRole('link', { name: '任务' }).click()

    // Then the Tasks page is displayed
    await expect(page).toHaveURL(/\/tasks/)
  })

  test('creates a task via the API and verifies it in the UI', async ({ page, taskFactory }) => {
    // Given a new task is created via the API
    const task = await taskFactory.create({
      title: 'E2E Test Task',
      description: 'Created by Playwright smoke test',
      quadrant: 1,
    })

    // When the user navigates to the Tasks page
    await page.goto('/tasks')

    // Then the task appears in the task list
    await expect(page.getByText(task.title)).toBeVisible()
  })

  test('navigates to Schedule page', async ({ page }) => {
    // Given the user is on the dashboard
    await page.goto('/')

    // When they click the 日程 nav link
    await page.getByRole('link', { name: '日程' }).click()

    // Then the Schedule calendar is displayed
    await expect(page).toHaveURL(/\/schedule/)
  })

  test('navigates to Timer page', async ({ page }) => {
    // Given the user is on the dashboard
    await page.goto('/')

    // When they click the 番茄钟 nav link
    await page.getByRole('link', { name: '番茄钟' }).click()

    // Then the Timer page is displayed
    await expect(page).toHaveURL(/\/timer/)
  })
})
