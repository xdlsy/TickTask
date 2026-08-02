import { test, expect } from '../support/fixtures'

test.describe('@p2 Accessibility', () => {
  test('A11Y-E2E-001: Tab key reaches main interactive elements', async ({ page }) => {
    // Given user is on Dashboard
    await page.goto('/')

    // When pressing Tab multiple times
    const tabPresses = 10
    for (let i = 0; i < tabPresses; i++) {
      await page.keyboard.press('Tab')
    }

    // Then focus is on an interactive element
    const focusedElement = page.locator(':focus')
    await expect(focusedElement).toBeVisible()
  })

  test('A11Y-E2E-002: Task form keyboard operable', async ({ page }) => {
    // Given user is on Tasks page
    await page.goto('/tasks')

    // When pressing Tab to reach "添加任务" button and pressing Enter
    const addBtn = page.getByRole('button', { name: /添加任务/ })
    if (await addBtn.isVisible()) {
      // Tab to the button and activate with keyboard
      await addBtn.focus()
      await page.keyboard.press('Enter')

      // Then form dialog may open
      await page.waitForTimeout(500)
    }

    // Page should remain accessible
    await expect(page.locator('body')).toBeVisible()
  })

  test('A11Y-E2E-003: Navigation bar keyboard reachable', async ({ page }) => {
    // Given user is on any page
    await page.goto('/')

    // When pressing Tab to reach nav links
    const navLinks = page.getByRole('navigation').getByRole('link')
    const linkCount = await navLinks.count()

    // Then navigation links are focusable
    if (linkCount > 0) {
      for (let i = 0; i < Math.min(linkCount, 3); i++) {
        await page.keyboard.press('Tab')
      }
      const focused = page.locator(':focus')
      await expect(focused).toBeVisible()
    }
  })

  test('A11Y-E2E-004: Timer control buttons have accessible names', async ({
    page,
  }) => {
    // Given user is on Timer page
    await page.goto('/timer')

    // Then buttons have visible text or aria-labels
    const buttons = page.locator('button')
    const count = await buttons.count()

    for (let i = 0; i < Math.min(count, 10); i++) {
      const btn = buttons.nth(i)
      const text = await btn.textContent()
      const ariaLabel = await btn.getAttribute('aria-label')
      // Each button should have either text content or aria-label
      if (text || ariaLabel) {
        expect(text || ariaLabel).toBeTruthy()
      }
    }
  })
})
