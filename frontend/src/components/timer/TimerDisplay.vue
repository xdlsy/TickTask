<template>
  <div class="timer-display">
    <div class="timer-circle" :style="{ width: size + 'px', height: size + 'px' }">
      <!-- 背景光晕效果 -->
      <div class="glow" :style="{ background: glowColor }"></div>

      <svg class="progress-ring" :width="size" :height="size">
        <defs>
          <filter id="shadow" x="-20%" y="-20%" width="140%" height="140%">
            <feDropShadow dx="0" dy="0" :stdDeviation="3" :flood-color="color" flood-opacity="0.4"/>
          </filter>
        </defs>
        <circle
          class="progress-ring-bg"
          :cx="size / 2"
          :cy="size / 2"
          :r="radius"
          fill="none"
          stroke="#e5e7eb"
          stroke-width="6"
        />
        <circle
          class="progress-ring-circle"
          :cx="size / 2"
          :cy="size / 2"
          :r="radius"
          fill="none"
          :stroke="color"
          stroke-width="6"
          :stroke-dasharray="circumference"
          :stroke-dashoffset="strokeDashoffset"
          stroke-linecap="round"
          filter="url(#shadow)"
        />
      </svg>
      <div class="timer-content">
        <div v-if="currentTask" class="task-name">{{ currentTask.title }}</div>
        <div class="timer-icon">
          <component :is="timerIcon" :size="28" />
        </div>
        <div class="timer-time">{{ formattedTime }}</div>
        <div class="timer-label">{{ label }}</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, h } from 'vue'
import { useTimerStore } from '@/stores/timer'
import { useTaskStore } from '@/stores/task'
import { formatTime } from '@/utils/time'

const props = withDefaults(
  defineProps<{
    size?: number
  }>(),
  {
    size: 260
  }
)

const timerStore = useTimerStore()
const taskStore = useTaskStore()

const currentTask = computed(() => {
  const taskId = timerStore.currentSession?.task_id
  if (!taskId) return null
  return taskStore.tasks.find(t => t.id === taskId) || null
})

const radius = computed(() => (props.size - 20) / 2)
const circumference = computed(() => 2 * Math.PI * radius.value)
const strokeDashoffset = computed(
  () => circumference.value - (timerStore.percentage / 100) * circumference.value
)

const label = computed(() => {
  if (timerStore.isRunning) return '专注中...'
  if (timerStore.isPaused) return '已暂停'
  return timerStore.currentSession ? '计时器' : '准备开始'
})

const formattedTime = computed(() => formatTime(timerStore.remainingTime))

// 使用 SVG 图标替代表情
const timerIcon = computed(() => {
  const mode = timerStore.currentMode
  if (mode === 'work') {
    return {
      render() {
        return h('svg', { viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': '2' }, [
          h('circle', { cx: '12', cy: '12', r: '10' }),
          h('path', { d: 'M12 6v6l4 2' })
        ])
      }
    }
  }
  if (mode === 'short_break') {
    return {
      render() {
        return h('svg', { viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': '2' }, [
          h('path', { d: 'M17 8h1a4 4 0 1 1 0 8h-1' }),
          h('path', { d: 'M3 8h14v9a4 4 0 0 1-4 4H7a4 4 0 0 1-4-4Z' }),
          h('line', { x1: '6', y1: '2', x2: '6', y2: '4' }),
          h('line', { x1: '10', y1: '2', x2: '10', y2: '4' }),
          h('line', { x1: '14', y1: '2', x2: '14', y2: '4' })
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
})

const color = computed(() => {
  const mode = timerStore.currentMode
  if (mode === 'work') return '#ef4444'
  if (mode === 'short_break') return '#22c55e'
  return '#3b82f6'
})

const glowColor = computed(() => {
  const mode = timerStore.currentMode
  if (mode === 'work') return 'radial-gradient(circle, rgba(239, 68, 68, 0.1) 0%, transparent 70%)'
  if (mode === 'short_break') return 'radial-gradient(circle, rgba(34, 197, 94, 0.1) 0%, transparent 70%)'
  return 'radial-gradient(circle, rgba(59, 130, 246, 0.1) 0%, transparent 70%)'
})
</script>

<style scoped>
.timer-display {
  display: flex;
  justify-content: center;
  align-items: center;
}

.timer-circle {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
}

.glow {
  position: absolute;
  width: 120%;
  height: 120%;
  border-radius: 50%;
  pointer-events: none;
}

.progress-ring {
  transform: rotate(-90deg);
  position: relative;
  z-index: 1;
}

.progress-ring-circle {
  transition: stroke-dashoffset 0.5s ease-out;
}

.timer-content {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  text-align: center;
  z-index: 2;
}

.timer-icon {
  width: 32px;
  height: 32px;
  margin: 0 auto 4px;
  color: #64748b;
}

.timer-icon svg {
  width: 100%;
  height: 100%;
}

.task-name {
  font-size: 14px;
  color: #6b7280;
  margin-bottom: 8px;
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.timer-time {
  font-size: 44px;
  font-weight: 700;
  color: #1f2937;
  font-variant-numeric: tabular-nums;
  line-height: 1;
  letter-spacing: -1px;
}

.timer-label {
  font-size: 13px;
  color: #6b7280;
  margin-top: 6px;
  font-weight: 500;
}

/* 响应式 */
@media (max-width: 480px) {
  .timer-icon {
    width: 28px;
    height: 28px;
  }

  .timer-time {
    font-size: 36px;
  }

  .timer-label {
    font-size: 12px;
  }
}
</style>