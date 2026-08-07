<template>
  <Teleport to="body">
    <Transition name="terminal-overlay">
      <div v-if="visible" class="terminal-overlay">
        <div class="terminal-window">
          <!-- Title Bar -->
          <div class="terminal-titlebar">
            <div class="titlebar-dots">
              <span class="dot dot-red"></span>
              <span class="dot dot-yellow"></span>
              <span class="dot dot-green" @click="$emit('close')"></span>
            </div>
            <div class="titlebar-title">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" width="14" height="14">
                <polyline points="4 17 10 11 4 5"/>
                <line x1="12" y1="19" x2="20" y2="19"/>
              </svg>
              <span>排程终端 · {{ toolName }}</span>
              <span v-if="status === 'started'" class="status-badge running">运行中</span>
              <span v-else-if="status === 'completed'" class="status-badge done">完成</span>
              <span v-else-if="status === 'error'" class="status-badge error">失败</span>
            </div>
            <div class="titlebar-actions">
              <button class="titlebar-btn" @click="toggleWrap">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="14" height="14">
                  <path d="M4 6h16M4 12h16M4 18h16"/>
                </svg>
              </button>
            </div>
          </div>

          <!-- Terminal Body -->
          <div ref="terminalBody" class="terminal-body">
            <!-- Welcome banner -->
            <div class="terminal-welcome">
              <span class="welcome-text">日程排程引擎</span>
              <span class="welcome-cmd">$ claude -p "执行 skill: docs/skills/auto-schedule"</span>
            </div>

            <!-- Initial waiting (before any message) -->
            <div v-if="status === ''" class="terminal-waiting">
              <span class="cursor-prompt">></span>
              <span class="cursor-blink">▊</span> 正在分析任务并生成日程...
            </div>

            <!-- Active streaming -->
            <div
              v-for="(line, idx) in displayLines"
              :key="idx"
              class="terminal-line"
              :class="{ stderr: line.isStderr }"
            >
              <span class="line-text">{{ line.text }}</span>
            </div>

            <!-- Running cursor -->
            <div v-if="status === 'started'" class="terminal-cursor">
              <span class="cursor-blink">▊</span>
            </div>

            <!-- Completion message -->
            <div v-if="status === 'completed' || status === 'error'" class="terminal-completion" :class="status">
              <div class="completion-line">─── {{ status === 'completed' ? '生成完成' : '生成失败' }} ───</div>
              <div v-if="statusMessage" class="completion-message">{{ statusMessage }}</div>
              <div v-if="statusDetail" class="completion-detail">{{ statusDetail }}</div>
            </div>
          </div>

          <!-- Reasoning Preview (collapsed by default) -->
          <div v-if="reasoning && status === 'completed'" class="terminal-reasoning">
            <div class="reasoning-toggle" @click="showReasoning = !showReasoning">
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                width="12"
                height="12"
                :class="{ rotated: showReasoning }"
              >
                <polyline points="9 18 15 12 9 6"/>
              </svg>
              <span>排程总结</span>
            </div>
            <div v-if="showReasoning" class="reasoning-content">{{ reasoning }}</div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, ref, watch, nextTick } from 'vue'
import type { TerminalLine } from '@/stores/schedule'

const props = defineProps<{
  visible: boolean
  lines: TerminalLine[]
  status: string
  statusMessage: string
  statusDetail: string
  reasoning: string
  toolName: string
}>()

defineEmits<{
  (e: 'close'): void
}>()

const terminalBody = ref<HTMLElement | null>(null)
const showReasoning = ref(false)
const wrapLines = ref(true)

const displayLines = computed(() => {
  if (wrapLines.value) return props.lines
  // Chunk mode: merge consecutive chunks of same type
  return props.lines
})

const status = computed(() => props.status)

// Auto-scroll to bottom when new lines arrive
watch(
  () => props.lines.length,
  async () => {
    await nextTick()
    if (terminalBody.value) {
      terminalBody.value.scrollTop = terminalBody.value.scrollHeight
    }
  }
)

// Reset reasoning visibility when terminal opens
watch(
  () => props.visible,
  (v) => {
    if (v) showReasoning.value = false
  }
)

function toggleWrap() {
  wrapLines.value = !wrapLines.value
}
</script>

<style scoped>
.terminal-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(10, 9, 6, 0.62);
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
}

.terminal-window {
  width: 680px;
  max-width: 90vw;
  max-height: 80vh;
  border-radius: var(--radius-lg);
  overflow: hidden;
  background: var(--bg-primary);
  border: 1px solid var(--border-accent);
  box-shadow: var(--shadow-pop), 0 0 0 1px rgba(230, 162, 60, 0.06);
  display: flex;
  flex-direction: column;
}

/* Title Bar */
.terminal-titlebar {
  display: flex;
  align-items: center;
  padding: 10px 14px;
  background: var(--bg-elevated);
  border-bottom: 1px solid var(--border-color);
  user-select: none;
}

.titlebar-dots {
  display: flex;
  gap: 6px;
  margin-right: 14px;
}

.dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
}

.dot-red { background: #ff5f57; }
.dot-yellow { background: #febc2e; }
.dot-green { background: #28c840; cursor: pointer; }

.titlebar-title {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  font-family: var(--font-mono);
  color: var(--text-muted);
  letter-spacing: 0.02em;
}

.status-badge {
  font-size: 10px;
  padding: 1px 7px;
  border-radius: 999px;
  font-weight: 500;
  font-family: var(--font-mono);
  letter-spacing: 0.06em;
}

.status-badge.running {
  background: var(--gold-fill);
  color: var(--accent-gold);
}

.status-badge.done {
  background: var(--sage-fill);
  color: var(--accent-sage);
}

.status-badge.error {
  background: var(--crimson-fill);
  color: var(--accent-crimson);
}

.titlebar-actions {
  display: flex;
  gap: 4px;
}

.titlebar-btn {
  background: none;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  padding: 4px;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  transition: background var(--transition-fast), color var(--transition-fast);
}

.titlebar-btn:hover {
  background: rgba(239, 231, 215, 0.06);
  color: var(--text-secondary);
}

/* Terminal Body */
.terminal-body {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  font-family: var(--font-mono);
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-primary);
  min-height: 200px;
  max-height: 400px;
  background: var(--bg-primary);
}

.terminal-body::-webkit-scrollbar {
  width: 4px;
}

.terminal-body::-webkit-scrollbar-track {
  background: transparent;
}

.terminal-body::-webkit-scrollbar-thumb {
  background: rgba(239, 231, 215, 0.12);
  border-radius: 2px;
}

.terminal-welcome {
  margin-bottom: 12px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--border-color);
}

.welcome-text {
  display: block;
  font-size: 10px;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.3em;
  margin-bottom: 5px;
}

.welcome-cmd {
  display: block;
  font-size: 12px;
  color: var(--accent-primary);
}

.terminal-waiting {
  color: var(--text-muted);
  display: flex;
  align-items: center;
  gap: 6px;
}

.terminal-line {
  white-space: pre-wrap;
  word-break: break-all;
  color: var(--text-secondary);
}

.terminal-line.stderr {
  color: var(--accent-crimson);
}

.line-text {
  /* terminal output text */
}

.terminal-cursor {
  display: flex;
  gap: 8px;
  color: var(--accent-primary);
}

.cursor-prompt {
  color: var(--accent-primary);
}

.cursor-blink {
  animation: blink 1s step-end infinite;
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

/* Completion */
.terminal-completion {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--border-accent);
}

.terminal-completion.completed .completion-line {
  color: var(--accent-sage);
}

.terminal-completion.error .completion-line {
  color: var(--accent-crimson);
}

.completion-line {
  font-size: 12px;
  margin-bottom: 6px;
  letter-spacing: 0.04em;
}

.completion-message {
  font-size: 13px;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.completion-detail {
  font-size: 12px;
  color: var(--text-muted);
}

/* Reasoning */
.terminal-reasoning {
  border-top: 1px solid var(--border-accent);
  padding: 10px 16px;
  background: var(--bg-secondary);
}

.reasoning-toggle {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-muted);
  cursor: pointer;
  user-select: none;
  transition: color var(--transition-fast);
}

.reasoning-toggle:hover {
  color: var(--accent-primary);
}

.reasoning-toggle svg {
  transition: transform 0.15s ease;
}

.reasoning-toggle svg.rotated {
  transform: rotate(90deg);
}

.reasoning-content {
  margin-top: 8px;
  font-size: 13px;
  color: var(--text-secondary);
  line-height: 1.6;
  max-height: 100px;
  overflow-y: auto;
}

/* Transitions */
.terminal-overlay-enter-active {
  transition: opacity 0.2s ease;
}

.terminal-overlay-enter-active .terminal-window {
  transition: transform 0.2s ease, opacity 0.2s ease;
}

.terminal-overlay-leave-active {
  transition: opacity 0.25s ease;
}

.terminal-overlay-leave-active .terminal-window {
  transition: transform 0.25s ease, opacity 0.25s ease;
}

.terminal-overlay-enter-from {
  opacity: 0;
}

.terminal-overlay-enter-from .terminal-window {
  transform: scale(0.96) translateY(12px);
  opacity: 0;
}

.terminal-overlay-leave-to {
  opacity: 0;
}

.terminal-overlay-leave-to .terminal-window {
  transform: scale(0.96) translateY(12px);
  opacity: 0;
}
</style>
