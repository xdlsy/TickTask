import { test, expect } from '../support/fixtures'

test.describe('@p0 Backend API', () => {
  test('API-E2E-001: Task CRUD lifecycle', async ({ apiClient }) => {
    // Create
    const created = await apiClient.createTask({
      title: 'CRUD Test Task',
      description: 'Testing full lifecycle',
      quadrant: 2,
      is_important: true,
      is_urgent: false,
    })
    expect(created.id).toBeTruthy()
    expect(created.title).toBe('CRUD Test Task')
    expect(created.status).toBe('todo')

    // Read
    const read = await apiClient.getTask(created.id)
    expect(read.title).toBe('CRUD Test Task')

    // Update
    await apiClient.updateTask(created.id, {
      title: 'Updated CRUD Task',
      status: 'in_progress',
    })
    const updated = await apiClient.getTask(created.id)
    expect(updated.title).toBe('Updated CRUD Task')
    expect(updated.status).toBe('in_progress')

    // Delete
    await apiClient.deleteTask(created.id)
    // After deletion, getTask returns 404 with error JSON
    const deletedRes = await apiClient.request.get(`/api/tasks/${created.id}`)
    expect(deletedRes.status()).toBe(404)
  })

  test('API-E2E-002: Schedule CRUD + Move', async ({ apiClient }) => {
    const today = new Date().toISOString().split('T')[0]

    // Create
    const created = await apiClient.createSchedule({
      title: 'CRUD Test Schedule',
      start_time: `${today}T09:00:00+08:00`,
      end_time: `${today}T10:00:00+08:00`,
      type: 'task',
    })
    expect(created.id).toBeTruthy()

    // Read (via list)
    const events = await apiClient.getSchedules()
    expect(events.some((e) => e.id === created.id)).toBeTruthy()

    // Move
    await apiClient.moveSchedule(created.id, {
      start_time: `${today}T14:00:00+08:00`,
      end_time: `${today}T15:00:00+08:00`,
    })

    // Update
    await apiClient.updateSchedule(created.id, {
      title: 'Updated Schedule',
    })

    // Delete
    await apiClient.deleteSchedule(created.id)
    const afterDelete = await apiClient.getSchedules()
    expect(afterDelete.some((e) => e.id === created.id)).toBeFalsy()
  })

  test('API-E2E-003: Session state machine', async ({ apiClient }) => {
    // Clean up any leftover active session
    const existing = await apiClient.getActiveSession()
    if (existing) {
      await apiClient.controlSession(existing.id, 'abandon').catch(() => {})
    }

    // Create session
    const session = await apiClient.createSession({
      type: 'work',
      duration: 25,
    })
    expect(session.id).toBeTruthy()
    expect(session.status).toBe('running')

    // Pause
    await apiClient.controlSession(session.id, 'pause')
    const paused = await apiClient.getActiveSession()
    expect(paused).toBeTruthy()

    // Resume
    await apiClient.controlSession(session.id, 'resume')

    // Complete
    await apiClient.controlSession(session.id, 'complete')

    // After completion, active session should be null (no running session)
    const afterComplete = await apiClient.getActiveSession()
    // May still return the completed session or null depending on backend behavior
    if (afterComplete) {
      expect(afterComplete.status).not.toBe('running')
    }
  })

  test('API-E2E-004: Analytics aggregation', async ({
    apiClient,
    taskFactory,
    sessionFactory,
  }) => {
    // Given a completed session
    const task = await taskFactory.create({ title: 'Analytics Task', quadrant: 1 })
    const session = await sessionFactory.create({
      task_id: task.id,
      type: 'work',
      duration: 25,
    })
    await apiClient.controlSession(session.id, 'complete')

    // When fetching analytics summary
    const summary = await apiClient.getAnalyticsSummary()

    // Then summary has valid data structure
    expect(summary).toBeTruthy()
    expect(typeof summary.completed_pomodoros).toBe('number')
    expect(typeof summary.completed_tasks).toBe('number')
  })
})
