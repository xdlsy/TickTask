<template>
  <div class="app">
    <el-container class="layout">
      <el-aside :width="sidebarWidth" class="sidebar">
        <div class="logo">
          <div class="logo-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/>
              <path d="M12 6v6l4 2"/>
            </svg>
          </div>
          <span v-show="appStore.sidebarOpen" class="logo-text">TickTask</span>
        </div>
        <el-menu
          :default-active="currentRoute"
          @select="handleMenuSelect"
          :collapse="!appStore.sidebarOpen"
          class="sidebar-menu"
        >
          <el-menu-item index="dashboard">
            <el-icon><DataBoard /></el-icon>
            <span>仪表盘</span>
          </el-menu-item>
          <el-menu-item index="timer">
            <el-icon><Timer /></el-icon>
            <span>番茄时钟</span>
          </el-menu-item>
          <el-menu-item index="tasks">
            <el-icon><List /></el-icon>
            <span>任务管理</span>
          </el-menu-item>
          <el-menu-item index="schedule">
            <el-icon><Calendar /></el-icon>
            <span>日程</span>
          </el-menu-item>
          <el-menu-item index="analytics">
            <el-icon><TrendCharts /></el-icon>
            <span>数据分析</span>
          </el-menu-item>
          <el-menu-item index="settings">
            <el-icon><Setting /></el-icon>
            <span>设置</span>
          </el-menu-item>
        </el-menu>
        <div class="sidebar-footer">
          <div class="toggle-btn" @click="appStore.toggleSidebar">
            <el-icon :size="18">
              <Fold v-if="appStore.sidebarOpen" />
              <Expand v-else />
            </el-icon>
          </div>
        </div>
      </el-aside>

      <el-main class="main-content">
        <div class="main-container">
          <router-view />
        </div>
      </el-main>
    </el-container>

    <div class="notifications">
      <el-notification
        v-for="notification in appStore.notifications"
        :key="notification.id"
        :title="notification.type === 'success' ? '成功' : '提示'"
        :type="notification.type"
        :message="notification.message"
        :duration="3000"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { DataBoard, Timer, List, Calendar, TrendCharts, Setting, Fold, Expand } from '@element-plus/icons-vue'
import { useAppStore } from '@/stores/app'
import { useTimerStore } from '@/stores/timer'
import { useAIStore } from '@/stores/ai'
import { wsClient } from '@/utils/websocket'

const router = useRouter()
const route = useRoute()
const appStore = useAppStore()
const timerStore = useTimerStore()
const aiStore = useAIStore()

const currentRoute = computed(() => route.name as string)
const sidebarWidth = computed(() => appStore.sidebarOpen ? '240px' : '72px')

function handleMenuSelect(index: string) {
  router.push(`/${index}`)
}

onMounted(async () => {
  wsClient.connect()
  timerStore.setupWebSocket()
  await aiStore.checkStatus()
})
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  background: #f0f2f5;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

#app {
  height: 100vh;
}

.app {
  height: 100%;
}

.layout {
  height: 100%;
}

.sidebar {
  background: linear-gradient(180deg, #1e293b 0%, #0f172a 100%);
  border-right: none;
  transition: width 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.logo {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 0 16px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.logo-icon {
  width: 32px;
  height: 32px;
  color: #60a5fa;
  flex-shrink: 0;
}

.logo-icon svg {
  width: 100%;
  height: 100%;
}

.logo-text {
  font-size: 18px;
  font-weight: 700;
  color: #fff;
  letter-spacing: -0.5px;
  white-space: nowrap;
}

.sidebar-menu {
  flex: 1;
  border-right: none;
  background: transparent;
  padding: 8px;
}

.sidebar-menu .el-menu-item {
  height: 44px;
  line-height: 44px;
  border-radius: 8px;
  margin-bottom: 4px;
  color: rgba(255, 255, 255, 0.65);
  transition: all 0.2s ease;
}

.sidebar-menu .el-menu-item:hover {
  background: rgba(255, 255, 255, 0.08);
  color: #fff;
}

.sidebar-menu .el-menu-item.is-active {
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
  color: #fff;
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.4);
}

.sidebar-menu .el-menu-item .el-icon {
  font-size: 18px;
}

.sidebar-footer {
  padding: 12px;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
}

.toggle-btn {
  width: 100%;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  color: rgba(255, 255, 255, 0.5);
  cursor: pointer;
  transition: all 0.2s ease;
}

.toggle-btn:hover {
  background: rgba(255, 255, 255, 0.08);
  color: #fff;
}

.main-content {
  padding: 0;
  overflow: hidden;
  background: #f0f2f5;
}

.main-container {
  height: 100%;
  overflow-y: auto;
}

.notifications {
  position: fixed;
  top: 20px;
  right: 20px;
  z-index: 9999;
}
</style>