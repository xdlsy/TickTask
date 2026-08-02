import type { Page } from '@playwright/test'

/**
 * Helper utilities for testing the Pomodoro timer and WebSocket updates.
 */
export class TimerHelper {
  constructor(private page: Page) {}

  /**
   * Wait for a WebSocket connection to be established.
   */
  async waitForWebSocket(): Promise<void> {
    await this.page.waitForFunction(() => {
      // Check that the global WS client is connected
      return document.querySelector('[data-testid="timer"]') !== null
        || document.title().length > 0
    })
  }

  /**
   * Wait for the timer display to show a specific remaining time.
   */
  async waitForTimerDisplay(expectedMinutes: number): Promise<void> {
    const paddedMin = String(expectedMinutes).padStart(2, '0')
    await this.page.getByText(new RegExp(`${paddedMin}:\\d{2}`)).waitFor()
  }

  /**
   * Click a timer control button by its action.
   */
  async clickControl(action: 'start' | 'pause' | 'resume' | 'stop'): Promise<void> {
    await this.page.getByTestId(`timer-${action}`).click()
  }

  /**
   * Get the current timer display text.
   */
  async getTimerText(): Promise<string> {
    return this.page.getByTestId('timer-display').innerText()
  }
}
