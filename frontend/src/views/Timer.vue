<template>
  <div class="timer-page">
    <div class="timer-hero">
      <div class="timer-eyebrow">Deep Focus · 25 min</div>
      <div class="timer-header">
        <h2>番茄时钟</h2>
        <p class="subtitle">专注工作，高效生活</p>
      </div>

      <div class="timer-main">
        <TimerDisplay :size="280" />
      </div>

      <TimerControls />
    </div>

    <div class="recent-sessions">
      <div class="section-header">
        <h3>最近记录</h3>
        <span class="session-count">{{ recentSessions.length }} 条记录</span>
      </div>
      <div v-if="recentSessions.length === 0" class="empty-state">
        <div class="empty-icon">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <circle cx="12" cy="12" r="10"/>
            <path d="M12 6v6l4 2"/>
          </svg>
        </div>
        <p>暂无计时记录</p>
        <p class="empty-hint">开始你的第一个番茄吧！</p>
      </div>
      <div v-else class="session-list">
        <div v-for="session in recentSessions" :key="session.id" class="session-item">
          <div class="session-left">
            <div class="session-icon" :class="`icon-${session.type}`">
              <component :is="getSessionIcon(session.type)" :size="18" />
            </div>
            <div class="session-info">
              <div class="session-title">{{ sessionTypeLabel(session.type) }}</div>
              <div class="session-meta">
                <span class="session-time">{{ formatDate(session.start_time) }}</span>
                <span v-if="session.actual_duration" class="session-duration">
                  {{ formatDuration(session.actual_duration) }}
                </span>
                <span v-if="session.interrupt_reason" class="interrupt-reason">
                  打断: {{ interruptReasonLabel(session.interrupt_reason) }}
                </span>
              </div>
            </div>
          </div>
          <div class="session-status" :class="`status-${session.status}`">
            {{ statusLabel(session.status) }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, h } from 'vue'
import TimerDisplay from '@/components/timer/TimerDisplay.vue'
import TimerControls from '@/components/timer/TimerControls.vue'
import { useTimerStore } from '@/stores/timer'
import { formatDuration } from '@/utils/time'
import type { SessionType, SessionStatus } from '@/types'

const timerStore = useTimerStore()

const recentSessions = computed(() => timerStore.recentSessions)

const sessionTypeLabels: Record<SessionType, string> = {
  work: '专注工作',
  short_break: '短休息',
  long_break: '长休息'
}

const statusLabels: Record<SessionStatus, string> = {
  pending: '待开始',
  running: '进行中',
  paused: '已暂停',
  completed: '已完成',
  abandoned: '已放弃'
}

function getSessionIcon(type: SessionType) {
  if (type === 'work') {
    return {
      render() {
        return h('svg', { viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': '2' }, [
          h('circle', { cx: '12', cy: '12', r: '10' }),
          h('path', { d: 'M12 6v6l4 2' })
        ])
      }
    }
  }
  if (type === 'short_break') {
    return {
      render() {
        return h('svg', { viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': '2' }, [
          h('path', { d: 'M17 8h1a4 4 0 1 1 0 8h-1' }),
          h('path', { d: 'M3 8h14v9a4 4 0 0 1-4 4H7a4 4 0 0 1-4-4Z' })
        ])
      }
    }
  }
  return {
    render() {
      return h('svg', { viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': '2' }, [
        h('path', { d: 'M17.5 19H9a7 7 0 1 1 6.71-9h1.79a4.5 4.5 0 1 1 0 9Z' })
      ])
    }
  }
}

function sessionTypeLabel(type: SessionType): string {
  return sessionTypeLabels[type]
}

const interruptReasonLabels: Record<string, string> = {
  meeting: '临时会议',
  call: '紧急电话',
  urgent: '突发急事',
  other: '其他'
}

function statusLabel(status: SessionStatus): string {
  return statusLabels[status]
}

function interruptReasonLabel(reason: string): string {
  return interruptReasonLabels[reason] || reason
}

function formatDate(dateStr: string): string {
  const date = new Date(dateStr)
  const now = new Date()
  const isToday = date.toDateString() === now.toDateString()

  if (isToday) {
    return '今天 ' + date.toLocaleTimeString('zh-CN', {
      hour: '2-digit',
      minute: '2-digit'
    })
  }

  return date.toLocaleDateString('zh-CN', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

onMounted(() => {
  timerStore.fetchActiveSession()
  timerStore.fetchRecentSessions()
})
</script>

<style scoped>
.timer-page {
  max-width: 720px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.timer-hero {
  position: relative;
  background: var(--gradient-warm);
  border-radius: var(--radius-2xl);
  padding: 48px 48px 40px;
  margin-bottom: 18px;
  border: 1px solid var(--border-color);
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  overflow: hidden;
}

.timer-hero::before {
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: radial-gradient(50% 45% at 50% 42%, rgba(230, 162, 60, 0.09), transparent 70%);
}

.timer-eyebrow {
  font-family: var(--font-mono);
  font-size: 10.5px;
  letter-spacing: 0.3em;
  text-transform: uppercase;
  color: var(--accent-primary);
  margin-bottom: 6px;
}

.timer-header {
  text-align: center;
  margin-bottom: 36px;
}

.timer-header h2 {
  font-family: var(--font-display);
  font-variation-settings: 'opsz' 144;
  font-size: 30px;
  font-weight: 400;
  color: var(--text-primary);
  margin: 0 0 8px 0;
  letter-spacing: -0.03em;
}

.subtitle {
  font-size: 14px;
  color: var(--text-muted);
  margin: 0;
  font-weight: 400;
}

.timer-main {
  display: flex;
  justify-content: center;
  margin-bottom: 36px;
  position: relative;
  z-index: 1;
}

.recent-sessions {
  background: var(--gradient-card);
  border-radius: var(--radius-xl);
  padding: 28px;
  border: 1px solid var(--border-color);
  width: 100%;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 18px;
}

.section-header h3 {
  font-family: var(--font-display);
  font-variation-settings: 'opsz' 60;
  font-size: 18px;
  font-weight: 420;
  color: var(--text-primary);
  margin: 0;
  letter-spacing: -0.02em;
}

.session-count {
  font-size: 11px;
  color: var(--text-muted);
  font-family: var(--font-mono);
  letter-spacing: 0.04em;
}

.empty-state {
  text-align: center;
  padding: 48px 24px;
  border-radius: var(--radius-md);
  border: 1px dashed var(--border-accent);
}

.empty-icon {
  width: 48px;
  height: 48px;
  margin: 0 auto 16px;
  color: var(--text-muted);
  opacity: 0.4;
}

.empty-icon svg {
  width: 100%;
  height: 100%;
}

.empty-state p {
  color: var(--text-secondary);
  margin: 0 0 4px 0;
  font-size: 14px;
}

.empty-hint {
  font-size: 13px;
  color: var(--text-muted) !important;
}

.session-list {
  display: flex;
  flex-direction: column;
}

.session-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 4px;
  border-bottom: 1px solid var(--border-color);
  transition: padding-left var(--transition-fast);
}

.session-item:last-child {
  border-bottom: none;
}

.session-item:hover {
  padding-left: 8px;
}

.session-left {
  display: flex;
  align-items: center;
  gap: 13px;
}

.session-icon {
  width: 36px;
  height: 36px;
  border-radius: 9px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.session-icon svg {
  width: 17px;
  height: 17px;
}

.icon-work {
  background: var(--accent-fill);
  color: var(--accent-primary);
}

.icon-short_break {
  background: var(--sage-fill);
  color: var(--accent-sage);
}

.icon-long_break {
  background: var(--gold-fill);
  color: var(--accent-gold);
}

.session-info {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.session-title {
  font-size: 13.5px;
  font-weight: 500;
  color: var(--text-primary);
}

.session-meta {
  display: flex;
  gap: 8px;
  font-size: 11.5px;
  color: var(--text-muted);
  font-family: var(--font-mono);
  letter-spacing: 0.02em;
}

.session-duration::before {
  content: '\00B7';
  margin-right: 8px;
}

.interrupt-reason {
  color: var(--accent-crimson);
  font-size: 11px;
}

.interrupt-reason::before {
  content: '\00B7';
  margin-right: 8px;
  color: var(--text-muted);
}

.session-status {
  padding: 4px 10px;
  border-radius: 999px;
  font-size: 10.5px;
  font-weight: 500;
  font-family: var(--font-mono);
  letter-spacing: 0.06em;
  flex-shrink: 0;
}

.status-completed {
  background: var(--sage-fill);
  color: var(--accent-sage);
}

.status-abandoned {
  background: var(--crimson-fill);
  color: var(--accent-crimson);
}

.status-paused {
  background: var(--gold-fill);
  color: var(--accent-gold);
}

.status-running {
  background: var(--accent-fill);
  color: var(--accent-primary);
}

.status-pending {
  background: rgba(239, 231, 215, 0.05);
  color: var(--text-muted);
}

@media (max-width: 640px) {
  .timer-hero {
    padding: 32px 20px;
    border-radius: var(--radius-lg);
  }

  .timer-header h2 {
    font-size: 24px;
  }

  .recent-sessions {
    padding: 20px;
    border-radius: var(--radius-lg);
  }
}
</style>
