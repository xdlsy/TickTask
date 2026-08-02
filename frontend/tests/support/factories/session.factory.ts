import { faker } from '@faker-js/faker'
import type { APIRequestContext } from '@playwright/test'
import type { PomodoroSession, SessionType } from '../../../src/types'

/**
 * Generates realistic PomodoroSession data and creates sessions via the API.
 * Tracks created sessions for automatic cleanup.
 */
export class SessionFactory {
  private createdIds: string[] = []

  constructor(private request: APIRequestContext) {}

  /**
   * Generate session creation data with optional overrides.
   */
  build(overrides: { task_id?: string; type?: SessionType; duration?: number } = {}): {
    task_id?: string
    type: string
    duration: number
  } {
    return {
      type: 'work',
      duration: 25, // Short duration for tests (25 seconds)
      ...overrides,
    }
  }

  /**
   * Create a session via the API and register it for cleanup.
   */
  async create(overrides: { task_id?: string; type?: SessionType; duration?: number } = {}): Promise<PomodoroSession> {
    const data = this.build(overrides)
    const response = await this.request.post('/api/sessions', { data })
    if (!response.ok()) {
      throw new Error(`Failed to create session: ${response.status()} ${await response.text()}`)
    }
    const session: PomodoroSession = await response.json()
    this.createdIds.push(session.id)
    return session
  }

  /**
   * Clean up all sessions created by this factory.
   * Abandons running sessions, completes others.
   */
  async cleanup(): Promise<void> {
    for (const id of this.createdIds) {
      await this.request
        .patch(`/api/sessions/${id}/control`, { data: { action: 'abandon' } })
        .catch(() => {})
    }
    this.createdIds = []
  }
}
