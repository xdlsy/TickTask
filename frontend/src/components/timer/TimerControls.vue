<template>
  <div class="timer-controls">
    <div v-if="!timerStore.currentSession || timerStore.currentSession.status === 'completed' || timerStore.currentSession.status === 'abandoned'" class="control-main">
      <button class="start-btn" @click="startWork">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="btn-svg">
          <circle cx="12" cy="12" r="10"/>
          <path d="M12 6v6l4 2"/>
        </svg>
        <span>开始专注</span>
        <span class="btn-duration">25 分钟</span>
      </button>
    </div>

    <template v-else>
      <div class="control-actions">
        <button
          v-if="timerStore.isRunning"
          class="ctrl-btn pause-btn"
          @click="pause"
        >
          <svg viewBox="0 0 24 24" fill="currentColor" stroke="none" class="btn-svg">
            <rect x="6" y="4" width="4" height="16" rx="1"/>
            <rect x="14" y="4" width="4" height="16" rx="1"/>
          </svg>
          暂停
        </button>

        <button
          v-if="timerStore.isPaused"
          class="ctrl-btn resume-btn"
          @click="resume"
        >
          <svg viewBox="0 0 24 24" fill="currentColor" stroke="none" class="btn-svg">
            <polygon points="5,3 19,12 5,21"/>
          </svg>
          继续
        </button>

        <button
          class="ctrl-btn complete-btn"
          @click="complete"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="btn-svg">
            <polyline points="20 6 9 17 4 12"/>
          </svg>
          完成
        </button>

        <button
          class="ctrl-btn abandon-btn"
          @click="abandon"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="btn-svg">
            <line x1="18" y1="6" x2="6" y2="18"/>
            <line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
          放弃
        </button>
      </div>
    </template>

    <div class="quick-actions">
      <button class="quick-btn short" @click="startShortBreak">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="quick-svg">
          <path d="M17 8h1a4 4 0 1 1 0 8h-1"/>
          <path d="M3 8h14v9a4 4 0 0 1-4 4H7a4 4 0 0 1-4-4Z"/>
        </svg>
        <span class="quick-text">短休息</span>
        <span class="quick-time">5分钟</span>
      </button>
      <button class="quick-btn long" @click="startLongBreak">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" class="quick-svg">
          <path d="M17.5 19H9a7 7 0 1 1 6.71-9h1.79a4.5 4.5 0 1 1 0 9Z"/>
        </svg>
        <span class="quick-text">长休息</span>
        <span class="quick-time">15分钟</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useTimerStore } from '@/stores/timer'
import { ElMessage, ElMessageBox } from 'element-plus'

const timerStore = useTimerStore()

async function startWork() {
  try {
    await timerStore.createSession(null, 'work')
  } catch (error) {
    ElMessage.error('启动计时器失败')
  }
}

async function startShortBreak() {
  try {
    await timerStore.createSession(null, 'short_break')
  } catch (error) {
    ElMessage.error('启动短休息失败')
  }
}

async function startLongBreak() {
  try {
    await timerStore.createSession(null, 'long_break')
  } catch (error) {
    ElMessage.error('启动长休息失败')
  }
}

async function pause() {
  try {
    await timerStore.controlSession('pause')
  } catch (error) {
    ElMessage.error('暂停失败')
  }
}

async function resume() {
  try {
    await timerStore.controlSession('resume')
  } catch (error) {
    ElMessage.error('继续失败')
  }
}

async function complete() {
  try {
    await timerStore.controlSession('complete')
    ElMessage.success('番茄完成!')
  } catch (error) {
    ElMessage.error('完成失败')
  }
}

async function abandon() {
  try {
    await ElMessageBox.confirm(
      '放弃当前计时？AI 将记录此次打断并调整后续排程',
      '放弃计时',
      { confirmButtonText: '确认放弃', cancelButtonText: '返回', type: 'warning' }
    )
    await timerStore.controlSession('abandon', 'other')
  } catch (error) {
    // User cancelled or API error
    if (error !== 'cancel' && error !== 'close') {
      ElMessage.error('放弃失败')
    }
  }
}
</script>

<style scoped>
.timer-controls {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 22px;
  position: relative;
  z-index: 1;
}

.control-main {
  width: 100%;
}

.start-btn {
  width: 100%;
  height: 54px;
  font-size: 15px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  background: var(--gradient-primary);
  border: none;
  font-weight: 600;
  font-family: var(--font-body);
  color: #fff;
  cursor: pointer;
  box-shadow: 0 4px 20px rgba(196, 103, 61, 0.3);
  transition: all var(--transition-normal);
}

.start-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 28px rgba(196, 103, 61, 0.4);
}

.start-btn .btn-svg {
  width: 18px;
  height: 18px;
}

.btn-duration {
  font-size: 12px;
  opacity: 0.8;
  margin-left: 4px;
  font-family: var(--font-mono);
}

.control-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  justify-content: center;
}

.ctrl-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 12px 22px;
  border-radius: var(--radius-md);
  font-size: 14px;
  font-weight: 600;
  font-family: var(--font-body);
  cursor: pointer;
  border: 1px solid transparent;
  transition: all var(--transition-normal);
}

.ctrl-btn .btn-svg {
  width: 16px;
  height: 16px;
}

.pause-btn {
  background: rgba(196, 149, 61, 0.1);
  border-color: rgba(196, 149, 61, 0.25);
  color: var(--accent-gold);
}

.pause-btn:hover {
  background: rgba(196, 149, 61, 0.18);
}

.resume-btn {
  background: rgba(196, 103, 61, 0.1);
  border-color: rgba(196, 103, 61, 0.25);
  color: var(--accent-primary);
}

.resume-btn:hover {
  background: rgba(196, 103, 61, 0.18);
}

.complete-btn {
  background: rgba(107, 139, 111, 0.1);
  border-color: rgba(107, 139, 111, 0.25);
  color: var(--accent-sage);
}

.complete-btn:hover {
  background: rgba(107, 139, 111, 0.18);
}

.abandon-btn {
  background: rgba(196, 85, 77, 0.06);
  border-color: rgba(196, 85, 77, 0.2);
  color: var(--accent-crimson);
}

.abandon-btn:hover {
  background: rgba(196, 85, 77, 0.12);
}

.quick-actions {
  display: flex;
  gap: 12px;
  width: 100%;
}

.quick-btn {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 18px 14px;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--transition-normal);
  font-family: var(--font-body);
}

.quick-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 16px rgba(60, 30, 10, 0.06);
}

.quick-btn.short:hover {
  background: rgba(107, 139, 111, 0.06);
  border-color: rgba(107, 139, 111, 0.3);
}

.quick-btn.long:hover {
  background: rgba(196, 103, 61, 0.06);
  border-color: rgba(196, 103, 61, 0.3);
}

.quick-svg {
  width: 22px;
  height: 22px;
  color: var(--text-secondary);
}

.quick-text {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}

.quick-time {
  font-size: 11px;
  color: var(--text-muted);
  font-family: var(--font-mono);
}

@media (max-width: 480px) {
  .ctrl-btn {
    padding: 10px 16px;
    font-size: 13px;
  }

  .quick-btn {
    padding: 14px 10px;
  }

  .quick-text {
    font-size: 13px;
  }
}
</style>
