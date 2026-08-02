import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAppStore } from '@/stores/app'

describe('App Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('initial state', () => {
    it('should have default view as dashboard', () => {
      const store = useAppStore()
      expect(store.currentView).toBe('dashboard')
    })

    it('should have sidebar open by default', () => {
      const store = useAppStore()
      expect(store.sidebarOpen).toBe(true)
    })

    it('should have empty notifications', () => {
      const store = useAppStore()
      expect(store.notifications).toEqual([])
    })
  })

  describe('setCurrentView', () => {
    it('should change current view', () => {
      const store = useAppStore()
      store.setCurrentView('timer')
      expect(store.currentView).toBe('timer')

      store.setCurrentView('tasks')
      expect(store.currentView).toBe('tasks')

      store.setCurrentView('analytics')
      expect(store.currentView).toBe('analytics')
    })
  })

  describe('toggleSidebar', () => {
    it('should toggle sidebar state', () => {
      const store = useAppStore()
      expect(store.sidebarOpen).toBe(true)

      store.toggleSidebar()
      expect(store.sidebarOpen).toBe(false)

      store.toggleSidebar()
      expect(store.sidebarOpen).toBe(true)
    })
  })

  describe('addNotification', () => {
    it('should add notification with default type', () => {
      const store = useAppStore()
      store.addNotification('Test message')

      expect(store.notifications).toHaveLength(1)
      expect(store.notifications[0].message).toBe('Test message')
      expect(store.notifications[0].type).toBe('info')
    })

    it('should add notification with specified type', () => {
      const store = useAppStore()
      store.addNotification('Error!', 'error')

      expect(store.notifications[0].type).toBe('error')
    })

    it('should auto-remove notification after 3 seconds', () => {
      const store = useAppStore()
      store.addNotification('Temporary')

      expect(store.notifications).toHaveLength(1)

      vi.advanceTimersByTime(3000)

      expect(store.notifications).toHaveLength(0)
    })

    it('should add multiple notifications with unique IDs', () => {
      const store = useAppStore()
      store.addNotification('First')
      vi.advanceTimersByTime(1) // advance to get different Date.now()
      store.addNotification('Second')

      expect(store.notifications).toHaveLength(2)
      expect(store.notifications[0].id).not.toBe(store.notifications[1].id)
    })
  })

  describe('removeNotification', () => {
    it('should remove notification by ID', () => {
      const store = useAppStore()
      const now = Date.now()
      let counter = 0
      vi.spyOn(Date, 'now').mockImplementation(() => now + counter++)
      store.addNotification('Keep')
      store.addNotification('Remove')
      const toRemove = store.notifications[1]

      store.removeNotification(toRemove.id)

      expect(store.notifications).toHaveLength(1)
      expect(store.notifications[0].message).toBe('Keep')
    })

    it('should do nothing if ID not found', () => {
      const store = useAppStore()
      store.addNotification('Only')

      store.removeNotification(99999)

      expect(store.notifications).toHaveLength(1)
    })
  })
})
