import { test, expect } from '../support/fixtures'

test.describe('@p2 Navigation', () => {
  test('NAV-E2E-001: Browser forward/back navigation', async ({ page }) => {
    // Given user navigates through pages
    await page.goto('/')
    await page.getByRole('link', { name: /任务/ }).click()
    await expect(page).toHaveURL(/\/tasks/)
    await page.getByRole('link', { name: /日程/ }).click()
    await expect(page).toHaveURL(/\/schedule/)

    // When pressing back
    await page.goBack()
    await expect(page).toHaveURL(/\/tasks/)

    // When pressing forward
    await page.goForward()
    await expect(page).toHaveURL(/\/schedule/)
  })

  test('NAV-E2E-002: Refresh any page restores correctly', async ({ page }) => {
    const routes = ['/dashboard', '/tasks', '/timer', '/schedule', '/analytics', '/settings']

    for (const route of routes) {
      // When visiting a page and refreshing
      await page.goto(route)
      await page.reload()

      // Then page still loads correctly
      await expect(page.locator('body')).toBeVisible()
      await expect(page).toHaveURL(new RegExp(route.replace('/', '/')))
    }
  })

  test('NAV-E2E-003: Deep link direct access works', async ({ page }) => {
    const routes = ['/timer', '/tasks', '/schedule', '/analytics', '/settings']

    for (const route of routes) {
      // When directly navigating to a deep link
      await page.goto(route)

      // Then page renders without needing to navigate from root
      await expect(page.locator('body')).toBeVisible()
      await expect(page).toHaveURL(new RegExp(route.replace('/', '/')))

      // Navigation bar should be present
      await expect(page.getByRole('navigation')).toBeVisible()
    }
  })
})
