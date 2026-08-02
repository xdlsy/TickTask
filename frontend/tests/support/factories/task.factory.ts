import { faker } from '@faker-js/faker'
import type { APIRequestContext } from '@playwright/test'
import type { Task, Quadrant, TaskStatus } from '../../../src/types'

/**
 * Generates realistic Task data and creates tasks via the API.
 * Tracks created tasks for automatic cleanup.
 */
export class TaskFactory {
  private createdIds: string[] = []

  constructor(private request: APIRequestContext) {}

  /**
   * Generate task fields with optional overrides.
   */
  build(overrides: Partial<Task> = {}): Omit<Task, 'id' | 'created_at' | 'updated_at' | 'completed_at'> {
    return {
      title: faker.lorem.sentence({ min: 2, max: 6 }),
      description: faker.lorem.paragraph(),
      quadrant: 2 as Quadrant,
      is_important: true,
      is_urgent: false,
      status: 'todo' as TaskStatus,
      estimated_time: faker.number.int({ min: 15, max: 120 }),
      deadline: faker.date.future().toISOString(),
      start_date: null,
      due_date: null,
      is_recurring: false,
      recurrence_pattern: '',
      recurrence_day: 0,
      preferred_start_time: '',
      preferred_end_time: '',
      tags: [faker.lorem.word()],
      order: 0,
      ...overrides,
    }
  }

  /**
   * Create a task via the API and register it for cleanup.
   */
  async create(overrides: Partial<Task> = {}): Promise<Task> {
    const data = this.build(overrides)
    const response = await this.request.post('/api/tasks', { data })
    if (!response.ok()) {
      throw new Error(`Failed to create task: ${response.status()} ${await response.text()}`)
    }
    const task: Task = await response.json()
    this.createdIds.push(task.id)
    return task
  }

  /**
   * Delete all tasks created by this factory.
   */
  async cleanup(): Promise<void> {
    for (const id of this.createdIds) {
      await this.request.delete(`/api/tasks/${id}`).catch(() => {})
    }
    this.createdIds = []
  }
}
