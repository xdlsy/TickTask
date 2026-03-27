<template>
  <div class="timer-controls">
    <!-- 未开始状态 -->
    <div v-if="!timerStore.currentSession || timerStore.currentSession.status === 'completed' || timerStore.currentSession.status === 'abandoned'" class="control-main">
      <el-button type="primary" size="large" class="start-btn" @click="startWork">
        <span class="btn-icon">🎯</span>
        <span>开始专注</span>
        <span class="btn-duration">25 分钟</span>
      </el-button>
    </div>

    <!-- 运行中/暂停状态 -->
    <template v-else>
      <div class="control-actions">
        <el-button
          v-if="timerStore.isRunning"
          type="warning"
          round
          @click="pause"
        >
          <span class="btn-icon">⏸</span> 暂停
        </el-button>

        <el-button
          v-if="timerStore.isPaused"
          type="primary"
          round
          @click="resume"
        >
          <span class="btn-icon">▶</span> 继续
        </el-button>

        <el-button
          type="success"
          round
          @click="complete"
        >
          <span class="btn-icon">✓</span> 完成
        </el-button>

        <el-button
          type="danger"
          round
          plain
          @click="abandon"
        >
          <span class="btn-icon">✕</span> 放弃
        </el-button>
      </div>
    </template>

    <!-- 快捷操作 -->
    <div class="quick-actions">
      <button class="quick-btn short" @click="startShortBreak">
        <span class="quick-icon">☕</span>
        <span class="quick-text">短休息</span>
        <span class="quick-time">5分钟</span>
      </button>
      <button class="quick-btn long" @click="startLongBreak">
        <span class="quick-icon">🌴</span>
        <span class="quick-text">长休息</span>
        <span class="quick-time">15分钟</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useTimerStore } from '@/stores/timer'
import { ElMessage } from 'element-plus'

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
    ElMessage.success('🎉 番茄完成！')
  } catch (error) {
    ElMessage.error('完成失败')
  }
}

async function abandon() {
  try {
    await timerStore.controlSession('abandon')
  } catch (error) {
    ElMessage.error('放弃失败')
  }
}
</script>

<style scoped>
.timer-controls {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
}

.control-main {
  width: 100%;
}

.start-btn {
  width: 100%;
  height: 56px;
  font-size: 16px;
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.start-btn .btn-icon {
  font-size: 18px;
}

.btn-duration {
  font-size: 13px;
  opacity: 0.85;
  margin-left: 4px;
}

.control-actions {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  justify-content: center;
}

.control-actions .el-button {
  min-width: 90px;
}

.btn-icon {
  margin-right: 4px;
}

/* 快捷操作 */
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
  gap: 4px;
  padding: 16px 12px;
  background: #f9fafb;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.quick-btn:hover {
  background: #f3f4f6;
  border-color: #d1d5db;
}

.quick-btn.short:hover {
  background: #f0fdf4;
  border-color: #86efac;
}

.quick-btn.long:hover {
  background: #eff6ff;
  border-color: #93c5fd;
}

.quick-icon {
  font-size: 20px;
}

.quick-text {
  font-size: 14px;
  font-weight: 500;
  color: #374151;
}

.quick-time {
  font-size: 11px;
  color: #9ca3af;
}

/* 响应式 */
@media (max-width: 480px) {
  .control-actions {
    gap: 8px;
  }

  .control-actions .el-button {
    min-width: 80px;
    font-size: 13px;
  }

  .quick-btn {
    padding: 12px 8px;
  }

  .quick-icon {
    font-size: 18px;
  }

  .quick-text {
    font-size: 13px;
  }
}
</style>