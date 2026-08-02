import { faker } from '@faker-js/faker'
import type { APIRequestContext } from '@playwright/test'
import type { ScheduleEvent, ScheduleType, CreateScheduleDTO } from '../../../src/types'

/**
 * Generates realistic Schedule data and creates events via the API.
 * Tracks created events for automatic cleanup.
 */
export class ScheduleFactory {
  private createdIds: string[] = []

  constructor(private request: APIRequestContext) {}

  /**
   * Generate schedule creation DTO with optional overrides.
   */
  build(overrides: Partial<CreateScheduleDTO> = {}): CreateScheduleDTO {
    const today = new Date().toISOString().split('T')[0]
    const startHour = faker.number.int({ min: 8, max: 17 })
    const endHour = startHour + 1

    return {
      title: faker.lorem.sentence({ min: 2, max: 5 }),
      start_time: `${today}T${String(startHour).padStart(2, '0')}:00:00+08:00`,
      end_time: `${today}T${String(endHour).padStart(2, '0')}:00:00+08:00`,
      type: 'task' as ScheduleType,
      ...overrides,
    }
  }

  /**
   * Create a schedule event via the API and register it for cleanup.
   */
  async create(overrides: Partial<CreateScheduleDTO> = {}): Promise<ScheduleEvent> {
    const data = this.build(overrides)
    const response = await this.request.post('/api/schedules', { data })
    if (!response.ok()) {
      throw new Error(`Failed to create schedule: ${response.status()} ${await response.text()}`)
    }
    const event: ScheduleEvent = await response.json()
    this.createdIds.push(event.id)
    return event
  }

  /**
   * Delete all schedule events created by this factory.
   */
  async cleanup(): Promise<void> {
    for (const id of this.createdIds) {
      await this.request.delete(`/api/schedules/${id}`).catch(() => {})
    }
    this.createdIds = []
  }
}
