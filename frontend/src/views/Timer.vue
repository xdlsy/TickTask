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

function statusLabel(status: SessionStatus): string {
  return statusLabels[status]
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
  max-width: 640px;
  margin: 0 auto;
  padding: 32px;
}

.timer-container {
  background: linear-gradient(135deg, #fff 0%, #f8fafc 100%);
  border-radius: 24px;
  padding: 40px 32px;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.06);
  margin-bottom: 24px;
  border: 1px solid rgba(0, 0, 0, 0.04);
}

.timer-header {
  text-align: center;
  margin-bottom: 32px;
}

.timer-header h2 {
  font-size: 26px;
  font-weight: 700;
  color: #1e293b;
  margin: 0 0 6px 0;
}

.subtitle {
  font-size: 14px;
  color: #94a3b8;
  margin: 0;
}

.timer-main {
  display: flex;
  justify-content: center;
  margin-bottom: 24px;
}

/* 最近会话区域 */
.recent-sessions {
  background: #fff;
  border-radius: 20px;
  padding: 24px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  border: 1px solid rgba(0, 0, 0, 0.04);
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.section-header h3 {
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
  margin: 0;
}

.session-count {
  font-size: 12px;
  color: #94a3b8;
  background: #f1f5f9;
  padding: 4px 10px;
  border-radius: 12px;
}

.empty-state {
  text-align: center;
  padding: 48px 24px;
}

.empty-icon {
  width: 64px;
  height: 64px;
  margin: 0 auto 16px;
  color: #cbd5e1;
}

.empty-icon svg {
  width: 100%;
  height: 100%;
}

.empty-state p {
  color: #64748b;
  margin: 0 0 4px 0;
}

.empty-hint {
  font-size: 13px;
  color: #94a3b8 !important;
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
  background: #f8fafc;
  border-radius: 12px;
  transition: all 0.2s ease;
}

.session-item:hover {
  background: #f1f5f9;
}

.session-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.session-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.session-icon svg {
  width: 18px;
  height: 18px;
}

.icon-work {
  background: linear-gradient(135deg, #fef2f2 0%, #fee2e2 100%);
  color: #ef4444;
}

.icon-short_break {
  background: linear-gradient(135deg, #f0fdf4 0%, #dcfce7 100%);
  color: #22c55e;
}

.icon-long_break {
  background: linear-gradient(135deg, #eff6ff 0%, #dbeafe 100%);
  color: #3b82f6;
}

.session-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.session-title {
  font-size: 14px;
  font-weight: 600;
  color: #1e293b;
}

.session-meta {
  display: flex;
  gap: 8px;
  font-size: 12px;
  color: #94a3b8;
}

.session-duration {
  color: #64748b;
}

.session-duration::before {
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
  background: linear-gradient(135deg, #d1fae5 0%, #a7f3d0 100%);
  color: #059669;
}

.status-abandoned {
  background: linear-gradient(135deg, #fee2e2 0%, #fecaca 100%);
  color: #dc2626;
}

.status-paused {
  background: linear-gradient(135deg, #fef3c7 0%, #fde68a 100%);
  color: #d97706;
}

.status-running {
  background: linear-gradient(135deg, #dbeafe 0%, #bfdbfe 100%);
  color: #2563eb;
}

/* 响应式 */
@media (max-width: 640px) {
  .timer-page {
    padding: 20px;
  }

  .timer-container {
    padding: 28px 20px;
    border-radius: 20px;
  }

  .timer-header h2 {
    font-size: 22px;
  }

  .recent-sessions {
    padding: 20px;
    border-radius: 16px;
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