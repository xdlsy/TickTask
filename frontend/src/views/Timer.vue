<template>
  <div class="timer-page">
    <div class="timer-container">
      <div class="timer-header">
        <h2>番茄时钟</h2>
        <p class="subtitle">专注工作，高效生活</p>
      </div>

      <div class="timer-main">
        <TimerDisplay :size="260" />
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

// 使用 SVG 图标替代表情
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
  max-width: 700px;
  margin: 0 auto;
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.timer-container {
  background: var(--bg-card);
  border-radius: 32px;
  padding: 56px 48px;
  box-shadow: 0 16px 60px rgba(60, 30, 10, 0.1);
  margin-bottom: 32px;
  border: 1px solid var(--border-color);
  position: relative;
  overflow: hidden;
  width: 100%;
}

.timer-container::before {
  content: '';
  position: absolute;
  top: -30%;
  left: -30%;
  width: 160%;
  height: 160%;
  background: radial-gradient(circle at center, rgba(196, 103, 61, 0.04) 0%, transparent 50%);
  pointer-events: none;
  animation: rotate 30s linear infinite;
}

@keyframes rotate {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.timer-header {
  text-align: center;
  margin-bottom: 48px;
  position: relative;
  z-index: 1;
}

.timer-header h2 {
  font-size: 32px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0 0 10px 0;
  letter-spacing: -0.5px;
}

.timer-header h2::before {
  content: '';
  display: inline-block;
  width: 8px;
  height: 8px;
  background: var(--accent-primary);
  border-radius: 50%;
  margin-right: 16px;
  vertical-align: middle;
  box-shadow: 0 0 16px var(--accent-primary);
  animation: pulse 2s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { transform: scale(1); opacity: 1; }
  50% { transform: scale(1.8); opacity: 0.5; }
}

.subtitle {
  font-size: 15px;
  color: var(--text-secondary);
  margin: 0;
}

.timer-main {
  display: flex;
  justify-content: center;
  margin-bottom: 40px;
  position: relative;
  z-index: 1;
}

/* 最近会话区域 */
.recent-sessions {
  background: var(--bg-card);
  border-radius: 24px;
  padding: 32px;
  box-shadow: 0 8px 32px rgba(60, 30, 10, 0.08);
  border: 1px solid var(--border-color);
  position: relative;
  width: 100%;
}

.recent-sessions::before {
  content: '';
  position: absolute;
  top: 0;
  left: 32px;
  right: 32px;
  height: 1px;
  background: linear-gradient(90deg, transparent, rgba(196, 103, 61, 0.2), transparent);
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.section-header h3 {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
  display: flex;
  align-items: center;
  gap: 10px;
}

.section-header h3::before {
  content: '';
  width: 4px;
  height: 20px;
  background: var(--gradient-primary);
  border-radius: 2px;
}

.session-count {
  font-size: 13px;
  color: var(--text-secondary);
  background: rgba(196, 103, 61, 0.05);
  padding: 6px 14px;
  border-radius: 20px;
  font-family: var(--font-mono);
}

.empty-state {
  text-align: center;
  padding: 48px 24px;
  background: rgba(196, 103, 61, 0.02);
  border-radius: var(--radius-md);
  border: 1px dashed var(--border-color);
}

.empty-icon {
  width: 64px;
  height: 64px;
  margin: 0 auto 16px;
  color: var(--text-muted);
}

.empty-icon svg {
  width: 100%;
  height: 100%;
}

.empty-state p {
  color: var(--text-secondary);
  margin: 0 0 4px 0;
}

.empty-hint {
  font-size: 13px;
  color: var(--text-muted) !important;
}

/* 会话列表 */
.session-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.session-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 16px;
  background: rgba(196, 103, 61, 0.02);
  border-radius: var(--radius-md);
  transition: all var(--transition-normal);
  border: 1px solid transparent;
}

.session-item:hover {
  background: rgba(196, 103, 61, 0.04);
  border-color: rgba(196, 103, 61, 0.1);
  transform: translateX(4px);
}

.session-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.session-icon {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
}

.session-icon::after {
  content: '';
  position: absolute;
  inset: -2px;
  border-radius: var(--radius-sm);
  background: inherit;
  filter: blur(8px);
  opacity: 0.4;
  z-index: -1;
}

.session-icon svg {
  width: 18px;
  height: 18px;
  position: relative;
  z-index: 1;
}

.icon-work {
  background: linear-gradient(135deg, #C4554D, #D4786D);
  color: #fff;
  box-shadow: 0 0 12px rgba(196, 85, 77, 0.25);
}

.icon-short_break {
  background: linear-gradient(135deg, #6B8B6F, #8BA88E);
  color: #fff;
  box-shadow: 0 0 12px rgba(107, 139, 111, 0.25);
}

.icon-long_break {
  background: linear-gradient(135deg, var(--accent-primary), var(--accent-secondary));
  color: #fff;
  box-shadow: 0 0 12px rgba(196, 103, 61, 0.2);
}

.session-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.session-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}

.session-meta {
  display: flex;
  gap: 8px;
  font-size: 12px;
  color: var(--text-secondary);
}

.session-duration {
  color: var(--text-muted);
  font-family: var(--font-mono);
}

.session-duration::before {
  content: '·';
  margin-right: 8px;
}

.interrupt-reason {
  color: var(--accent-crimson);
  font-size: 11px;
}

.interrupt-reason::before {
  content: '·';
  margin-right: 8px;
}

/* 状态标签 */
.session-status {
  padding: 5px 12px;
  border-radius: 8px;
  font-size: 11px;
  font-weight: 600;
}

.status-completed {
  background: rgba(107, 139, 111, 0.08);
  color: var(--accent-sage);
  border: 1px solid rgba(107, 139, 111, 0.15);
}

.status-abandoned {
  background: rgba(196, 85, 77, 0.08);
  color: var(--accent-crimson);
  border: 1px solid rgba(196, 85, 77, 0.15);
}

.status-paused {
  background: rgba(196, 149, 61, 0.08);
  color: var(--accent-gold);
  border: 1px solid rgba(196, 149, 61, 0.15);
}

.status-running {
  background: rgba(196, 103, 61, 0.08);
  color: var(--accent-primary);
  border: 1px solid rgba(196, 103, 61, 0.15);
  animation: statusGlow 2s ease-in-out infinite;
}

@keyframes statusGlow {
  0%, 100% { box-shadow: 0 0 8px rgba(196, 103, 61, 0.15); }
  50% { box-shadow: 0 0 16px rgba(196, 103, 61, 0.25); }
}

/* 响应式 */
@media (max-width: 640px) {
  .timer-page {
    padding: 20px;
  }

  .timer-container {
    padding: 32px 20px;
    border-radius: var(--radius-lg);
  }

  .timer-header h2 {
    font-size: 24px;
  }

  .recent-sessions {
    padding: 20px;
    border-radius: var(--radius-md);
  }

  .session-item {
    padding: 12px;
  }

  .session-icon {
    width: 36px;
    height: 36px;
  }
}
</style>