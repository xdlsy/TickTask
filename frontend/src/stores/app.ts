import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useAppStore = defineStore('app', () => {
  // State
  const currentView = ref('dashboard') // dashboard, timer, tasks, analytics
  const sidebarOpen = ref(true)
  const notifications = ref<Notification[]>([])

  // Actions
  function setCurrentView(view: string) {
    currentView.value = view
  }

  function toggleSidebar() {
    sidebarOpen.value = !sidebarOpen.value
  }

  function addNotification(message: string, type: 'success' | 'error' | 'info' | 'warning' = 'info') {
    const id = Date.now()
    notifications.value.push({ id, message, type })
    setTimeout(() => removeNotification(id), 3000)
  }

  function removeNotification(id: number) {
    notifications.value = notifications.value.filter(n => n.id !== id)
  }

  return {
    currentView,
    sidebarOpen,
    notifications,
    setCurrentView,
    toggleSidebar,
    addNotification,
    removeNotification
  }
})

interface Notification {
  id: number
  message: string
  type: 'success' | 'error' | 'info' | 'warning'
}
