<template>
  <div class="app">
    <header class="topbar">
      <div class="topbar-left">
        <div class="logo">
          <div class="logo-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <circle cx="12" cy="12" r="10"/>
              <path d="M12 6v6l4 2"/>
            </svg>
          </div>
          <span class="logo-text">TickTask</span>
        </div>
      </div>

      <nav class="topbar-nav">
        <router-link
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          class="nav-item"
          :class="{ active: currentRoute === item.name }"
        >
          <component :is="item.icon" :size="16" />
          <span>{{ item.label }}</span>
        </router-link>
      </nav>

      <div class="topbar-right">
        <div class="today-date">{{ todayDate }}</div>
      </div>
    </header>

    <main class="main-content">
      <div class="main-container">
        <router-view v-slot="{ Component }">
          <transition name="page" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { DataBoard, Timer, List, Calendar, TrendCharts, Setting } from '@element-plus/icons-vue'
import { useTimerStore } from '@/stores/timer'
import { useAIStore } from '@/stores/ai'
import { wsClient } from '@/utils/websocket'

const route = useRoute()
const timerStore = useTimerStore()
const aiStore = useAIStore()

const currentRoute = computed(() => route.name as string)

const todayDate = computed(() => {
  const now = new Date()
  const weekdays = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
  return `${now.getMonth() + 1}月${now.getDate()}日 ${weekdays[now.getDay()]}`
})

const navItems = [
  { path: '/dashboard', name: 'dashboard', label: '仪表盘', icon: DataBoard },
  { path: '/timer', name: 'timer', label: '番茄钟', icon: Timer },
  { path: '/tasks', name: 'tasks', label: '任务', icon: List },
  { path: '/schedule', name: 'schedule', label: '日程', icon: Calendar },
  { path: '/analytics', name: 'analytics', label: '分析', icon: TrendCharts },
  { path: '/settings', name: 'settings', label: '设置', icon: Setting },
]

onMounted(async () => {
  wsClient.connect()
  timerStore.setupWebSocket()
  await aiStore.checkStatus()
})
</script>

<style>
/* ── Typography ── */
@import url('https://fonts.googleapis.com/css2?family=Playfair+Display:ital,wght@0,500;0,600;0,700;1,500&family=DM+Sans:opsz,wght@9..40,400;9..40,500;9..40,600&family=JetBrains+Mono:wght@400;500&display=swap');

/* ── Design Tokens ── */
:root {
  /* Background — warm paper */
  --bg-primary: #FAF9F6;
  --bg-secondary: #F5F2ED;
  --bg-card: #FFFEFC;
  --bg-card-hover: #FDFBF8;
  --bg-elevated: #FFFFFE;

  /* Border — nearly invisible */
  --border-color: rgba(0, 0, 0, 0.06);
  --border-accent: rgba(0, 0, 0, 0.10);

  /* Text */
  --text-primary: #1C1B1A;
  --text-secondary: #6E6A65;
  --text-muted: #9C9893;

  /* Accent — single muted burnt umber, used sparingly */
  --accent-primary: #B8452C;
  --accent-secondary: #C9604A;
  --accent-tertiary: #D98A75;
  --accent-crimson: #B8452C;
  --accent-sage: #6B8B6F;
  --accent-gold: #B8954D;

  /* Glow — barely there */
  --glow-primary: 0 0 0 1px rgba(184, 69, 44, 0.12);
  --glow-crimson: 0 0 0 1px rgba(184, 69, 44, 0.10);
  --glow-sage: 0 0 0 1px rgba(107, 139, 111, 0.10);

  /* Gradients — subtle */
  --gradient-primary: linear-gradient(135deg, #B8452C 0%, #C9604A 100%);
  --gradient-warm: linear-gradient(180deg, #FAF9F6 0%, #F5F2ED 100%);
  --gradient-card: linear-gradient(160deg, #FFFEFC 0%, #FDFBF8 100%);

  /* Typography */
  --font-display: 'Playfair Display', 'Georgia', 'Times New Roman', serif;
  --font-body: 'DM Sans', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  --font-mono: 'JetBrains Mono', 'Fira Code', 'Menlo', monospace;

  /* Radii — softer */
  --radius-sm: 4px;
  --radius-md: 8px;
  --radius-lg: 12px;
  --radius-xl: 16px;
  --radius-2xl: 20px;

  /* Transitions */
  --transition-fast: 0.15s ease;
  --transition-normal: 0.25s ease;
  --transition-slow: 0.35s ease;
}

/* ── Reset ── */
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: var(--font-body);
  background: var(--bg-primary);
  color: var(--text-primary);
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  overflow: hidden;
}

#app {
  height: 100vh;
}

.app {
  height: 100%;
}

/* ── Topbar ── */
.topbar {
  height: 60px;
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 40px;
  position: relative;
  z-index: 100;
}

.topbar-left {
  display: flex;
  align-items: center;
}

/* Logo */
.logo {
  display: flex;
  align-items: center;
  gap: 10px;
}

.logo-icon {
  width: 30px;
  height: 30px;
  color: var(--accent-primary);
}

.logo-icon svg {
  width: 100%;
  height: 100%;
}

.logo-text {
  font-family: var(--font-display);
  font-size: 18px;
  font-weight: 600;
  letter-spacing: -0.3px;
  color: var(--text-primary);
}

/* Navigation */
.topbar-nav {
  display: flex;
  align-items: center;
  gap: 2px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  color: var(--text-secondary);
  text-decoration: none;
  font-size: 13px;
  font-weight: 500;
  border-radius: 6px;
  transition: all var(--transition-fast);
  letter-spacing: -0.1px;
  white-space: nowrap;
}

.nav-item:hover {
  color: var(--text-primary);
  background: rgba(0, 0, 0, 0.04);
}

.nav-item.active {
  color: var(--accent-primary);
  background: rgba(184, 69, 44, 0.06);
  font-weight: 600;
}

.topbar-right {
  display: flex;
  align-items: center;
}

.today-date {
  font-size: 12px;
  color: var(--text-muted);
  font-weight: 500;
  letter-spacing: 0.2px;
}

/* ── Main Content ── */
.main-content {
  padding: 0;
  overflow: hidden;
  background: var(--bg-primary);
  position: relative;
  height: calc(100vh - 60px);
}

.main-container {
  height: 100%;
  overflow-y: auto;
  position: relative;
  z-index: 1;
  padding: 40px;
}

/* Scrollbar */
.main-container::-webkit-scrollbar {
  width: 4px;
}

.main-container::-webkit-scrollbar-track {
  background: transparent;
}

.main-container::-webkit-scrollbar-thumb {
  background: rgba(0, 0, 0, 0.10);
  border-radius: 2px;
}

.main-container::-webkit-scrollbar-thumb:hover {
  background: rgba(0, 0, 0, 0.18);
}

/* ── Page Transitions ── */
.page-enter-active {
  animation: pageIn 0.35s ease-out;
}

.page-leave-active {
  animation: pageOut 0.2s ease-in;
}

@keyframes pageIn {
  from {
    opacity: 0;
    transform: translateY(12px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes pageOut {
  from {
    opacity: 1;
    transform: translateY(0);
  }
  to {
    opacity: 0;
    transform: translateY(-6px);
  }
}

/* ── Element Plus Overrides ── */
.el-button--primary {
  --el-button-bg-color: var(--accent-primary);
  --el-button-border-color: var(--accent-primary);
  --el-button-hover-bg-color: var(--accent-secondary);
  --el-button-hover-border-color: var(--accent-secondary);
  --el-button-active-bg-color: #9E3B24;
  --el-button-active-border-color: #9E3B24;
  font-family: var(--font-body);
  font-weight: 500;
  letter-spacing: -0.1px;
}

.el-button--primary:focus {
  box-shadow: var(--glow-primary);
}

.el-input__wrapper {
  background: var(--bg-card) !important;
  border: 1px solid var(--border-color) !important;
  box-shadow: none !important;
  border-radius: var(--radius-sm) !important;
  transition: border-color var(--transition-fast) !important;
}

.el-input__wrapper:hover {
  border-color: var(--border-accent) !important;
}

.el-input__wrapper.is-focus {
  border-color: var(--accent-primary) !important;
  box-shadow: 0 0 0 2px rgba(184, 69, 44, 0.10) !important;
}

.el-input__inner {
  color: var(--text-primary) !important;
  font-family: var(--font-body);
}

.el-input__inner::placeholder {
  color: var(--text-muted) !important;
}

.el-select__wrapper {
  background: var(--bg-card) !important;
  border: 1px solid var(--border-color) !important;
  box-shadow: none !important;
}

.el-select-dropdown {
  background: var(--bg-card) !important;
  border: 1px solid var(--border-color) !important;
  border-radius: var(--radius-md) !important;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.06) !important;
}

.el-select-dropdown__item {
  color: var(--text-secondary) !important;
  font-family: var(--font-body);
  font-size: 13px !important;
}

.el-select-dropdown__item:hover {
  background: rgba(0, 0, 0, 0.04) !important;
  color: var(--text-primary) !important;
}

.el-select-dropdown__item.is-selected {
  color: var(--accent-primary) !important;
  background: rgba(184, 69, 44, 0.06) !important;
  font-weight: 600 !important;
}

.el-popper {
  background: var(--bg-card) !important;
  border: 1px solid var(--border-color) !important;
}

.el-popper__arrow::before {
  background: var(--bg-card) !important;
  border-color: var(--border-color) !important;
}

.el-dialog {
  background: var(--bg-card) !important;
  border-radius: var(--radius-xl) !important;
  border: 1px solid var(--border-color) !important;
  font-family: var(--font-body);
  box-shadow: 0 24px 80px rgba(0, 0, 0, 0.08) !important;
}

.el-dialog__header {
  font-family: var(--font-display);
}

.el-dialog__title {
  color: var(--text-primary) !important;
  font-weight: 600 !important;
}

.el-form-item__label {
  color: var(--text-secondary) !important;
  font-family: var(--font-body);
  font-weight: 500 !important;
  font-size: 13px !important;
}

.el-tag {
  font-family: var(--font-body);
  border-radius: var(--radius-sm) !important;
}

.el-switch.is-checked .el-switch__core {
  background-color: var(--accent-sage) !important;
  border-color: var(--accent-sage) !important;
}

.el-radio-button__original-radio:checked + .el-radio-button__inner {
  background-color: var(--accent-primary) !important;
  border-color: var(--accent-primary) !important;
  box-shadow: none !important;
}

.el-input-number__decrease:hover,
.el-input-number__increase:hover {
  color: var(--accent-primary) !important;
}

.el-dropdown-menu {
  background: var(--bg-card) !important;
  border: 1px solid var(--border-color) !important;
  border-radius: var(--radius-md) !important;
  padding: 4px !important;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.06) !important;
}

.el-dropdown-menu__item {
  color: var(--text-secondary) !important;
  border-radius: var(--radius-sm) !important;
  font-family: var(--font-body);
  font-size: 13px !important;
}

.el-dropdown-menu__item:hover {
  background: rgba(0, 0, 0, 0.04) !important;
  color: var(--text-primary) !important;
}

.el-message {
  font-family: var(--font-body);
  border-radius: var(--radius-md) !important;
}

.el-time-select .el-select__wrapper {
  padding: 4px 12px !important;
  min-height: 32px !important;
}
</style>
