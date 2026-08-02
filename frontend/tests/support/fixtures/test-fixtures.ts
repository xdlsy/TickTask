import { test as base, expect } from '@playwright/test'
import { TaskFactory } from '../factories/task.factory'
import { ScheduleFactory } from '../factories/schedule.factory'
import { SessionFactory } from '../factories/session.factory'
import { ApiClient } from '../helpers/api-client'

/**
 * Custom fixtures for TickTask E2E tests.
 * Extends Playwright's base test with typed helpers.
 */
type TickTaskFixtures = {
  taskFactory: TaskFactory
  scheduleFactory: ScheduleFactory
  sessionFactory: SessionFactory
  apiClient: ApiClient
}

export const test = base.extend<TickTaskFixtures>({
  taskFactory: async ({ request }, use) => {
    const factory = new TaskFactory(request)
    await use(factory)
    await factory.cleanup()
  },

  scheduleFactory: async ({ request }, use) => {
    const factory = new ScheduleFactory(request)
    await use(factory)
    await factory.cleanup()
  },

  sessionFactory: async ({ request }, use) => {
    const factory = new SessionFactory(request)
    await use(factory)
    await factory.cleanup()
  },

  apiClient: async ({ request }, use) => {
    const client = new ApiClient(request)
    await use(client)
  },
})

export { expect }
