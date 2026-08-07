<template>
  <div class="timer-display">
    <div class="timer-circle" :style="{ width: size + 'px', height: size + 'px' }">
      <svg class="progress-ring" :width="size" :height="size">
        <circle
          class="progress-ring-bg"
          :cx="size / 2"
          :cy="size / 2"
          :r="radius"
          fill="none"
          stroke="rgba(239, 231, 215, 0.05)"
          stroke-width="14"
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
        />
      </svg>
      <div class="timer-content">
        <div v-if="currentTask" class="task-name">{{ currentTask.title }}</div>
        <div class="timer-icon" v-html="timerSvgIcon"></div>
        <div class="timer-time">{{ formattedTime }}</div>
        <div class="timer-label" :class="{ live: timerStore.isRunning }">{{ label }}</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
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

const timerSvgIcon = computed(() => {
  const mode = timerStore.currentMode
  if (mode === 'work') {
    return `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" width="28" height="28">
      <circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/>
    </svg>`
  }
  if (mode === 'short_break') {
    return `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" width="28" height="28">
      <path d="M17 8h1a4 4 0 1 1 0 8h-1"/><path d="M3 8h14v9a4 4 0 0 1-4 4H7a4 4 0 0 1-4-4Z"/>
      <line x1="6" y1="2" x2="6" y2="4"/><line x1="10" y1="2" x2="10" y2="4"/><line x1="14" y1="2" x2="14" y2="4"/>
    </svg>`
  }
  return `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" width="28" height="28">
    <path d="M17.5 19H9a7 7 0 1 1 6.71-9h1.79a4.5 4.5 0 1 1 0 9Z"/>
  </svg>`
})

const color = computed(() => {
  const mode = timerStore.currentMode
  if (mode === 'work') return 'var(--accent-primary)'
  if (mode === 'short_break') return 'var(--accent-sage)'
  return 'var(--accent-gold)'
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

.progress-ring {
  transform: rotate(-90deg);
  filter: drop-shadow(0 0 24px rgba(230, 162, 60, 0.16));
}

.progress-ring-circle {
  transition: stroke-dashoffset 0.5s cubic-bezier(0.22, 1, 0.36, 1);
}

.timer-content {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.timer-icon {
  width: 32px;
  height: 32px;
  margin: 0 auto 10px;
  color: var(--text-muted);
  display: flex;
  align-items: center;
  justify-content: center;
}

.task-name {
  font-size: 12.5px;
  color: var(--text-secondary);
  margin-bottom: 14px;
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  letter-spacing: 0.01em;
}

.timer-time {
  font-size: 68px;
  font-weight: 360;
  color: var(--text-primary);
  font-variant-numeric: tabular-nums;
  line-height: 1;
  letter-spacing: -0.04em;
  font-family: var(--font-display);
  font-variation-settings: 'opsz' 144;
}

.timer-label {
  font-size: 10.5px;
  color: var(--text-muted);
  margin-top: 16px;
  font-weight: 500;
  letter-spacing: 0.3em;
  text-transform: uppercase;
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.timer-label.live {
  color: var(--accent-primary);
}

.timer-label.live::before {
  content: '';
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--accent-primary);
  box-shadow: 0 0 10px var(--accent-primary);
  animation: timerPulse 1.8s ease-in-out infinite;
}

@keyframes timerPulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.35; }
}

@media (max-width: 480px) {
  .timer-time {
    font-size: 48px;
  }

  .timer-label {
    font-size: 10px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .timer-label.live::before {
    animation: none;
  }
}
</style>
