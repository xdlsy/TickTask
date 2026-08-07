<template>
  <div class="app">
    <div class="ambient" aria-hidden="true"></div>

    <header class="topbar">
      <div class="topbar-left">
        <router-link to="/dashboard" class="logo">
          <span class="logo-mark">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="12" cy="12" r="9" />
              <path d="M12 7v5l3 2" />
            </svg>
          </span>
          <span class="logo-text">Tick<em>Task</em></span>
        </router-link>
      </div>

      <nav class="topbar-nav">
        <router-link
          v-for="item in navItems"
          :key="item.path"
          :to="item.path"
          class="nav-item"
          :class="{ active: currentRoute === item.name }"
        >
          <component :is="item.icon" :size="15" />
          <span>{{ item.label }}</span>
        </router-link>
      </nav>

      <div class="topbar-right">
        <span v-if="timerStore.isRunning" class="focus-chip">
          <span class="focus-dot"></span>专注中
        </span>
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
import { DataBoard, Timer, List, Calendar, TrendCharts, Setting, Document } from '@element-plus/icons-vue'
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
  const mm = String(now.getMonth() + 1).padStart(2, '0')
  const dd = String(now.getDate()).padStart(2, '0')
  return `${mm}.${dd} · ${weekdays[now.getDay()]}`
})

const navItems = [
  { path: '/dashboard', name: 'dashboard', label: '仪表盘', icon: DataBoard },
  { path: '/timer', name: 'timer', label: '番茄钟', icon: Timer },
  { path: '/tasks', name: 'tasks', label: '任务', icon: List },
  { path: '/schedule', name: 'schedule', label: '日程', icon: Calendar },
  { path: '/analytics', name: 'analytics', label: '分析', icon: TrendCharts },
  { path: '/work-log', name: 'work-log', label: '工作日志', icon: Document },
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
@import url('https://fonts.googleapis.com/css2?family=Fraunces:ital,opsz,wght@0,9..144,300..600;1,9..144,300..600&family=Geist:wght@300..700&family=Geist+Mono:wght@400;500;600&display=swap');

/* ════════════════════════════════════════════════════════════
   TICKTASK — "ATELIER NOIR"
   Warm-ink editorial dark. Fraunces × Geist × Geist Mono.
   ════════════════════════════════════════════════════════════ */
:root {
  /* ── Ink — warm, never blue ── */
  --bg-primary: #14120D;
  --bg-secondary: #181610;
  --bg-card: #1E1B14;
  --bg-card-hover: #242019;
  --bg-elevated: #28241C;
  color-scheme: dark;

  /* ── Hairlines (warm translucent bone) ── */
  --border-color: rgba(239, 231, 215, 0.07);
  --border-accent: rgba(239, 231, 215, 0.13);
  --border-strong: rgba(239, 231, 215, 0.20);

  /* ── Bone type ── */
  --text-primary: #F4EFE3;
  --text-secondary: #B3AB99;
  --text-muted: #8A8273;

  /* ── Accent — amber, used with restraint ── */
  --accent-primary: #E6A23C;
  --accent-secondary: #F2B654;
  --accent-tertiary: #C97E2A;
  --accent-crimson: #D86F54;
  --accent-sage: #8FB28C;
  --accent-gold: #D6B45A;
  --accent-sky: #7FA8C0;

  /* translucent accent fills (for icon chips / soft tags) */
  --accent-fill: rgba(230, 162, 60, 0.14);
  --crimson-fill: rgba(216, 111, 84, 0.14);
  --sage-fill: rgba(143, 178, 140, 0.14);
  --gold-fill: rgba(214, 180, 90, 0.14);
  --sky-fill: rgba(127, 168, 192, 0.14);

  /* ── Glow (amber only) ── */
  --glow-primary: 0 0 0 1px rgba(230, 162, 60, 0.24), 0 0 32px rgba(230, 162, 60, 0.10);
  --glow-crimson: 0 0 0 1px rgba(216, 111, 84, 0.22);
  --glow-sage: 0 0 0 1px rgba(143, 178, 140, 0.20);

  /* ── Gradients — subtle, warm ── */
  --gradient-primary: linear-gradient(150deg, #F2B654 0%, #E6A23C 100%);
  --gradient-warm: radial-gradient(120% 100% at 50% 0%, rgba(230, 162, 60, 0.05), transparent 60%), linear-gradient(180deg, #181610 0%, #14120D 100%);
  --gradient-card: linear-gradient(170deg, #1E1B14 0%, #181610 100%);

  /* ── Shadows — layered, soft, warm-black depth ── */
  --shadow-xs: 0 1px 0 rgba(255, 255, 255, 0.025) inset, 0 1px 2px rgba(0, 0, 0, 0.40);
  --shadow-sm: 0 1px 0 rgba(255, 255, 255, 0.03) inset, 0 2px 8px rgba(0, 0, 0, 0.35);
  --shadow-card: 0 1px 0 rgba(255, 255, 255, 0.03) inset, 0 10px 28px rgba(0, 0, 0, 0.38);
  --shadow-card-hover: 0 1px 0 rgba(255, 255, 255, 0.045) inset, 0 18px 48px rgba(0, 0, 0, 0.50);
  --shadow-pop: 0 1px 0 rgba(255, 255, 255, 0.04) inset, 0 24px 64px rgba(0, 0, 0, 0.55);

  /* ── Typography ── */
  --font-display: 'Fraunces', 'Iowan Old Style', 'Georgia', serif;
  --font-body: 'Geist', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
  --font-mono: 'Geist Mono', 'JetBrains Mono', ui-monospace, 'Menlo', monospace;

  /* ── Radii ── */
  --radius-sm: 6px;
  --radius-md: 10px;
  --radius-lg: 14px;
  --radius-xl: 20px;
  --radius-2xl: 26px;

  /* ── Transitions ── */
  --transition-fast: 0.15s cubic-bezier(0.22, 1, 0.36, 1);
  --transition-normal: 0.25s cubic-bezier(0.22, 1, 0.36, 1);
  --transition-slow: 0.35s cubic-bezier(0.22, 1, 0.36, 1);

  /* ── Element Plus brand binding ── */
  --el-color-primary: #E6A23C;
  --el-color-primary-light-3: #C98A30;
  --el-color-primary-light-5: #B07928;
  --el-color-primary-light-7: #6E4B18;
  --el-color-primary-light-8: #4A3210;
  --el-color-primary-light-9: rgba(230, 162, 60, 0.12);
  --el-color-primary-dark-2: #F2B654;
  --el-color-success: #8FB28C;
  --el-color-warning: #D6B45A;
  --el-color-danger: #D86F54;
  --el-color-error: #D86F54;
  --el-color-info: #8A8273;
  --el-bg-color: #1E1B14;
  --el-bg-color-page: #14120D;
  --el-bg-color-overlay: #242019;
  --el-text-color-primary: #F4EFE3;
  --el-text-color-regular: #B3AB99;
  --el-text-color-secondary: #8A8273;
  --el-text-color-placeholder: #6E6859;
  --el-text-color-disabled: #4A4538;
  --el-border-color: rgba(239, 231, 215, 0.12);
  --el-border-color-light: rgba(239, 231, 215, 0.09);
  --el-border-color-lighter: rgba(239, 231, 215, 0.07);
  --el-border-color-extra-light: rgba(239, 231, 215, 0.05);
  --el-border-color-dark: rgba(239, 231, 215, 0.18);
  --el-fill-color: rgba(239, 231, 215, 0.04);
  --el-fill-color-light: rgba(239, 231, 215, 0.03);
  --el-fill-color-lighter: rgba(239, 231, 215, 0.02);
  --el-fill-color-extra-light: rgba(239, 231, 215, 0.015);
  --el-fill-color-dark: rgba(239, 231, 215, 0.06);
  --el-fill-color-darker: rgba(239, 231, 215, 0.08);
  --el-fill-color-blank: #1E1B14;
  --el-mask-color: rgba(10, 9, 6, 0.72);
  --el-box-shadow: 0 1px 0 rgba(255, 255, 255, 0.03) inset, 0 12px 40px rgba(0, 0, 0, 0.45);
  --el-box-shadow-light: 0 1px 0 rgba(255, 255, 255, 0.03) inset, 0 8px 24px rgba(0, 0, 0, 0.40);
  --el-box-shadow-lighter: 0 6px 16px rgba(0, 0, 0, 0.35);
  --el-box-shadow-dark: 0 16px 48px rgba(0, 0, 0, 0.55);
  --el-disabled-bg-color: rgba(239, 231, 215, 0.04);
  --el-disabled-text-color: #5C5749;
  --el-disabled-border-color: rgba(239, 231, 215, 0.08);
  --el-border-radius-base: 8px;
  --el-border-radius-small: 6px;
}

/* ── Reset ── */
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body {
  background: var(--bg-primary);
}

body {
  font-family: var(--font-body);
  color: var(--text-primary);
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  font-feature-settings: 'ss01', 'cv01';
  overflow: hidden;
}

::selection {
  background: rgba(230, 162, 60, 0.22);
  color: var(--text-primary);
}

/* 暖色氛围光 — 固定铺满视口,营造电影感的纵深 */
.ambient {
  position: fixed;
  inset: 0;
  z-index: 0;
  pointer-events: none;
  background:
    radial-gradient(1100px 700px at 80% -10%, rgba(230, 162, 60, 0.09), transparent 60%),
    radial-gradient(900px 600px at 5% 0%, rgba(216, 111, 84, 0.05), transparent 55%);
}

/* 极淡的暖色颗粒 — 叠加混合,营造"印在墨纸上"的质感 */
body::before {
  content: '';
  position: fixed;
  inset: 0;
  z-index: 0;
  pointer-events: none;
  opacity: 0.04;
  mix-blend-mode: overlay;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='220' height='220'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.85' numOctaves='2' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E");
}

#app {
  height: 100vh;
  position: relative;
  z-index: 1;
}

.app {
  height: 100%;
}

/* ── Topbar ── */
.topbar {
  height: 64px;
  background: rgba(16, 14, 9, 0.72);
  backdrop-filter: blur(18px) saturate(1.2);
  -webkit-backdrop-filter: blur(18px) saturate(1.2);
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 36px;
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
  gap: 12px;
  text-decoration: none;
}

.logo-mark {
  width: 30px;
  height: 30px;
  border-radius: 8px;
  display: grid;
  place-items: center;
  background: var(--gradient-primary);
  color: #14120D;
  box-shadow: 0 0 0 1px rgba(230, 162, 60, 0.4), 0 6px 18px rgba(230, 162, 60, 0.22);
  flex-shrink: 0;
}

.logo-mark svg {
  width: 17px;
  height: 17px;
}

.logo-text {
  font-family: var(--font-display);
  font-variation-settings: 'opsz' 144;
  font-size: 21px;
  font-weight: 500;
  letter-spacing: -0.02em;
  color: var(--text-primary);
}

.logo-text em {
  font-style: italic;
  font-weight: 400;
  color: var(--accent-primary);
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
  gap: 7px;
  padding: 7px 13px;
  color: var(--text-secondary);
  text-decoration: none;
  font-size: 13px;
  font-weight: 450;
  border-radius: 8px;
  transition: color var(--transition-fast), background var(--transition-fast);
  letter-spacing: -0.01em;
  white-space: nowrap;
  position: relative;
}

.nav-item :deep(svg),
.nav-item svg {
  opacity: 0.85;
}

.nav-item:hover {
  color: var(--text-primary);
  background: rgba(239, 231, 215, 0.04);
}

.nav-item.active {
  color: var(--text-primary);
}

.nav-item.active::after {
  content: '';
  position: absolute;
  left: 13px;
  right: 13px;
  bottom: -21px;
  height: 2px;
  background: var(--accent-primary);
  border-radius: 2px;
  box-shadow: 0 0 12px rgba(230, 162, 60, 0.6);
}

.topbar-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.focus-chip {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--accent-primary);
  letter-spacing: 0.04em;
}

.focus-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--accent-primary);
  box-shadow: 0 0 10px var(--accent-primary);
  animation: focusPulse 1.8s ease-in-out infinite;
}

@keyframes focusPulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.35; }
}

.today-date {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-muted);
  font-weight: 400;
  letter-spacing: 0.12em;
}

/* ── Main Content ── */
.main-content {
  padding: 0;
  overflow: hidden;
  background: transparent;
  position: relative;
  height: calc(100vh - 64px);
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
  width: 5px;
}

.main-container::-webkit-scrollbar-track {
  background: transparent;
}

.main-container::-webkit-scrollbar-thumb {
  background: rgba(239, 231, 215, 0.10);
  border-radius: 3px;
}

.main-container::-webkit-scrollbar-thumb:hover {
  background: rgba(239, 231, 215, 0.18);
}

/* ── 卡片:统一的暖墨层叠柔影(全局;页面 scoped 若显式设置 box-shadow 则以其为准) ── */
.card,
.stat-card {
  box-shadow: var(--shadow-card);
}

/* ── Page Transitions ── */
/* 页面根快速淡入并保持 class 存活,以覆盖子区块的错峰窗口;直接子区块按序上浮 */
.page-enter-active {
  animation: pageFadeIn 0.8s cubic-bezier(0.22, 1, 0.36, 1) both;
}

.page-leave-active {
  animation: pageFadeOut 0.15s ease-in both;
}

@keyframes pageFadeIn {
  0% { opacity: 0; }
  18% { opacity: 1; }
  100% { opacity: 1; }
}

@keyframes pageFadeOut {
  from { opacity: 1; }
  to { opacity: 0; }
}

.page-enter-active > * {
  animation: revealUp 0.55s cubic-bezier(0.22, 1, 0.36, 1) both;
}

.page-enter-active > *:nth-child(1) { animation-delay: 0.04s; }
.page-enter-active > *:nth-child(2) { animation-delay: 0.09s; }
.page-enter-active > *:nth-child(3) { animation-delay: 0.14s; }
.page-enter-active > *:nth-child(4) { animation-delay: 0.19s; }
.page-enter-active > *:nth-child(5) { animation-delay: 0.24s; }
.page-enter-active > *:nth-child(n + 6) { animation-delay: 0.28s; }

@keyframes revealUp {
  from { opacity: 0; transform: translateY(14px); }
  to { opacity: 1; transform: translateY(0); }
}

/* 尊重「减少动态」系统偏好 */
@media (prefers-reduced-motion: reduce) {
  .page-enter-active,
  .page-enter-active > *,
  .page-leave-active,
  .focus-dot {
    animation: none !important;
  }
}

/* ════════════════════════════════════════════════════════════
   Element Plus Overrides — Atelier Noir
   ════════════════════════════════════════════════════════════ */

/* Buttons */
.el-button {
  font-family: var(--font-body);
  font-weight: 500;
  letter-spacing: -0.01em;
}

.el-button--primary {
  --el-button-bg-color: var(--accent-primary);
  --el-button-border-color: var(--accent-primary);
  --el-button-hover-bg-color: var(--accent-secondary);
  --el-button-hover-border-color: var(--accent-secondary);
  --el-button-hover-text-color: #14120D;
  --el-button-active-bg-color: var(--accent-tertiary);
  --el-button-active-border-color: var(--accent-tertiary);
  --el-button-active-text-color: #14120D;
  color: #14120D;
  font-weight: 600;
  box-shadow: var(--shadow-sm);
}

.el-button--primary:hover {
  box-shadow: 0 8px 24px rgba(230, 162, 60, 0.24);
}

.el-button--primary:focus {
  box-shadow: var(--glow-primary);
}

.el-button.is-text {
  color: var(--text-secondary);
}

.el-button.is-text:hover {
  color: var(--accent-primary);
  background: var(--accent-fill);
}

.el-button.is-link {
  font-family: var(--font-mono);
  font-size: 11px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.el-button.is-link:hover {
  color: var(--accent-secondary);
}

.el-button--default {
  background: var(--bg-card);
  border-color: var(--border-accent);
  color: var(--text-secondary);
}

.el-button--default:hover {
  background: var(--bg-card-hover);
  border-color: var(--border-strong);
  color: var(--text-primary);
}

/* Inputs */
.el-input__wrapper,
.el-textarea__inner {
  background: var(--bg-secondary) !important;
  box-shadow: 0 0 0 1px var(--border-accent) inset !important;
  border-radius: var(--radius-sm) !important;
  transition: box-shadow var(--transition-fast) !important;
}

.el-input__wrapper:hover,
.el-textarea__inner:hover {
  box-shadow: 0 0 0 1px var(--border-strong) inset !important;
}

.el-input__wrapper.is-focus,
.el-textarea__inner:focus {
  box-shadow: 0 0 0 1px var(--accent-primary) inset, 0 0 0 3px rgba(230, 162, 60, 0.12) !important;
}

.el-input__inner,
.el-textarea__inner {
  color: var(--text-primary) !important;
  font-family: var(--font-body);
}

.el-input__inner::placeholder,
.el-textarea__inner::placeholder {
  color: var(--text-muted) !important;
}

/* Select */
.el-select__wrapper {
  background: var(--bg-secondary) !important;
  box-shadow: 0 0 0 1px var(--border-accent) inset !important;
  border-radius: var(--radius-sm) !important;
}

.el-select__wrapper.is-hovering {
  box-shadow: 0 0 0 1px var(--border-strong) inset !important;
}

.el-select__wrapper.is-focused {
  box-shadow: 0 0 0 1px var(--accent-primary) inset, 0 0 0 3px rgba(230, 162, 60, 0.12) !important;
}

.el-select-dropdown,
.el-dropdown-menu,
.el-select__popper.el-popper {
  background: var(--bg-elevated) !important;
  border: 1px solid var(--border-accent) !important;
  border-radius: var(--radius-md) !important;
  box-shadow: var(--shadow-pop) !important;
}

.el-select-dropdown__item {
  color: var(--text-secondary) !important;
  font-family: var(--font-body);
  font-size: 13px !important;
  border-radius: var(--radius-sm) !important;
}

.el-select-dropdown__item.is-hovering,
.el-select-dropdown__item:hover {
  background: rgba(239, 231, 215, 0.05) !important;
  color: var(--text-primary) !important;
}

.el-select-dropdown__item.is-selected {
  color: var(--accent-primary) !important;
  background: var(--accent-fill) !important;
  font-weight: 600 !important;
}

.el-popper.is-light {
  background: var(--bg-elevated) !important;
  border: 1px solid var(--border-accent) !important;
}

.el-popper.is-light .el-popper__arrow::before {
  background: var(--bg-elevated) !important;
  border-color: var(--border-accent) !important;
}

/* Dialog */
.el-dialog,
.el-drawer {
  background: var(--bg-card) !important;
  border-radius: var(--radius-xl) !important;
  border: 1px solid var(--border-accent) !important;
  font-family: var(--font-body);
  box-shadow: var(--shadow-pop) !important;
}

.el-dialog__header,
.el-drawer__header {
  font-family: var(--font-display);
}

.el-dialog__title,
.el-drawer__title {
  color: var(--text-primary) !important;
  font-weight: 500 !important;
  letter-spacing: -0.02em;
}

.el-dialog__headerbtn .el-dialog__close,
.el-drawer__close-btn {
  color: var(--text-muted) !important;
}

.el-dialog__headerbtn:hover .el-dialog__close {
  color: var(--accent-primary) !important;
}

.el-overlay {
  background: var(--el-mask-color) !important;
  backdrop-filter: blur(2px);
}

/* Form */
.el-form-item__label {
  color: var(--text-secondary) !important;
  font-family: var(--font-body);
  font-weight: 500 !important;
  font-size: 13px !important;
}

.el-form-item__error {
  color: var(--accent-crimson) !important;
}

/* Tags */
.el-tag {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.1em;
  border-radius: 999px !important;
  border: 1px solid var(--border-accent) !important;
  background: rgba(239, 231, 215, 0.04) !important;
  color: var(--text-secondary) !important;
  font-weight: 500;
}

.el-tag--primary {
  background: var(--accent-fill) !important;
  color: var(--accent-primary) !important;
  border-color: rgba(230, 162, 60, 0.3) !important;
}

.el-tag--success {
  background: var(--sage-fill) !important;
  color: var(--accent-sage) !important;
  border-color: rgba(143, 178, 140, 0.3) !important;
}

.el-tag--warning {
  background: var(--gold-fill) !important;
  color: var(--accent-gold) !important;
  border-color: rgba(214, 180, 90, 0.3) !important;
}

.el-tag--danger {
  background: var(--crimson-fill) !important;
  color: var(--accent-crimson) !important;
  border-color: rgba(216, 111, 84, 0.3) !important;
}

/* Switch */
.el-switch.is-checked .el-switch__core {
  background-color: var(--accent-primary) !important;
  border-color: var(--accent-primary) !important;
}

.el-switch__core {
  border-color: var(--border-accent) !important;
  background-color: var(--bg-elevated) !important;
}

/* Radio button group */
.el-radio-button__original-radio:checked + .el-radio-button__inner {
  background-color: var(--accent-primary) !important;
  border-color: var(--accent-primary) !important;
  color: #14120D !important;
  box-shadow: none !important;
  font-weight: 600;
}

.el-radio-button__inner {
  background: var(--bg-secondary) !important;
  border-color: var(--border-accent) !important;
  color: var(--text-secondary) !important;
}

/* Radio / Checkbox */
.el-radio__label,
.el-checkbox__label {
  color: var(--text-secondary) !important;
}

.el-radio__input.is-checked .el-radio__inner,
.el-checkbox__input.is-checked .el-checkbox__inner {
  background-color: var(--accent-primary) !important;
  border-color: var(--accent-primary) !important;
}

.el-radio__input.is-checked + .el-radio__label,
.el-checkbox__input.is-checked + .el-checkbox__label {
  color: var(--accent-primary) !important;
}

.el-radio__inner,
.el-checkbox__inner {
  background: var(--bg-secondary) !important;
  border-color: var(--border-accent) !important;
}

/* Input number */
.el-input-number__decrease:hover,
.el-input-number__increase:hover {
  color: var(--accent-primary) !important;
}

.el-input-number__decrease,
.el-input-number__increase {
  background: transparent !important;
  color: var(--text-muted) !important;
  border-color: var(--border-color) !important;
}

/* Dropdown menu */
.el-dropdown-menu {
  background: var(--bg-elevated) !important;
  border: 1px solid var(--border-accent) !important;
  border-radius: var(--radius-md) !important;
  padding: 4px !important;
  box-shadow: var(--shadow-pop) !important;
}

.el-dropdown-menu__item {
  color: var(--text-secondary) !important;
  border-radius: var(--radius-sm) !important;
  font-family: var(--font-body);
  font-size: 13px !important;
}

.el-dropdown-menu__item:not(.is-disabled):hover,
.el-dropdown-menu__item:focus {
  background: rgba(239, 231, 215, 0.05) !important;
  color: var(--text-primary) !important;
}

/* Message / Notification */
.el-message {
  font-family: var(--font-body);
  background: var(--bg-elevated) !important;
  border: 1px solid var(--border-accent) !important;
  border-radius: var(--radius-md) !important;
  box-shadow: var(--shadow-pop) !important;
  color: var(--text-primary) !important;
}

.el-message--success { border-color: rgba(143, 178, 140, 0.35) !important; }
.el-message--warning { border-color: rgba(214, 180, 90, 0.35) !important; }
.el-message--error { border-color: rgba(216, 111, 84, 0.35) !important; }

.el-message__content {
  color: var(--text-primary) !important;
}

/* Card / Table */
.el-card {
  background: var(--bg-card) !important;
  border-color: var(--border-accent) !important;
  border-radius: var(--radius-xl) !important;
}

.el-table {
  background: transparent !important;
  color: var(--text-secondary) !important;
  --el-table-bg-color: transparent;
  --el-table-tr-bg-color: transparent;
  --el-table-header-bg-color: transparent;
  --el-table-border-color: var(--border-color);
  --el-table-row-hover-bg-color: rgba(239, 231, 215, 0.03);
  --el-table-text-color: var(--text-secondary);
  --el-table-header-text-color: var(--text-muted);
}

.el-table th.el-table__cell {
  background: transparent !important;
  font-family: var(--font-mono) !important;
  font-size: 10.5px !important;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  font-weight: 500 !important;
}

.el-table td.el-table__cell,
.el-table th.el-table__cell {
  border-bottom-color: var(--border-color) !important;
}

.el-table__empty-block {
  background: transparent;
}

/* Date picker / time select */
.el-time-select .el-select__wrapper {
  padding: 4px 12px !important;
  min-height: 32px !important;
}

.el-time-panel,
.el-date-picker,
.el-picker__popper {
  background: var(--bg-elevated) !important;
  border: 1px solid var(--border-accent) !important;
}

/* Tooltip */
.el-popper.is-dark {
  background: var(--bg-elevated) !important;
  border: 1px solid var(--border-accent) !important;
  color: var(--text-primary) !important;
  font-size: 12px;
}

/* Divider */
.el-divider {
  border-color: var(--border-color);
}

.el-divider__text {
  background: var(--bg-card) !important;
  color: var(--text-muted) !important;
}

/* Alert */
.el-alert {
  border-radius: var(--radius-md) !important;
}

.el-alert--info {
  background: rgba(239, 231, 215, 0.04) !important;
}

/* Tabs */
.el-tabs__item {
  color: var(--text-muted) !important;
  font-family: var(--font-body);
  font-weight: 500;
}

.el-tabs__item.is-active {
  color: var(--text-primary) !important;
}

.el-tabs__active-bar {
  background-color: var(--accent-primary) !important;
}

.el-tabs__nav-wrap::after {
  background-color: var(--border-color) !important;
}

/* Empty */
.el-empty__description p {
  color: var(--text-muted) !important;
}
</style>
