<template>
  <div :class="['tool-card', permClass, statusClass]" data-testid="tool-card">
    <div class="tool-name">
      <span class="icon">{{ statusIcon }}</span>
      <code>{{ message.tool_name }}</code>
      <el-tag size="small" :type="tagType">{{ statusLabel }}</el-tag>
    </div>
    <pre class="tool-args">{{ message.tool_args }}</pre>
    <div v-if="message.tool_result" class="tool-result">{{ displayResult }}</div>
    <div v-if="message.tool_status === 'pending_confirmation'" class="actions">
      <el-button size="small" type="primary" data-testid="confirm-btn" @click="onApprove">✓ 确认</el-button>
      <el-button size="small" @click="onReject">✕ 取消</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useAgentStore } from '@/stores/agent'
import type { AgentMessage } from '@/types'

const props = defineProps<{ message: AgentMessage }>()
const store = useAgentStore()

// Tools classified as mutations ("write" permission) — everything else that
// doesn't fall in danger is treated as a read.
const WRITE_TOOLS = ['create_task', 'update_task', 'start_pomodoro', 'stop_pomodoro', 'generate_schedule', 'save_worklog']
const DANGER_TOOLS = ['delete_task', 'delete_schedule']

const isDanger = computed(() => DANGER_TOOLS.includes(props.message.tool_name || ''))
const isFailed = computed(() => props.message.tool_status === 'failed')

const permClass = computed<string>(() => {
  if (isFailed.value) return 'failed'
  if (isDanger.value) return 'danger'
  return WRITE_TOOLS.includes(props.message.tool_name || '') ? 'write' : 'read'
})

const statusClass = computed(() => `status-${props.message.tool_status || 'unknown'}`)

const STATUS_LABELS: Record<string, string> = {
  started: '执行中',
  pending_confirmation: '待确认',
  succeeded: '已执行',
  failed: '失败',
  rejected: '已取消',
}
const statusLabel = computed(() => STATUS_LABELS[props.message.tool_status || ''] || props.message.tool_status || '')

const TAG_TYPES: Record<string, 'success' | 'warning' | 'danger' | 'info'> = {
  succeeded: 'success',
  pending_confirmation: 'warning',
  failed: 'danger',
  rejected: 'info',
  started: 'info',
}
const tagType = computed(() => TAG_TYPES[props.message.tool_status || ''] ?? 'info')

const STATUS_ICONS: Record<string, string> = {
  succeeded: '✓',
  pending_confirmation: '⏸',
  failed: '⚠',
  rejected: '✕',
  started: '▶',
}
const statusIcon = computed(() => STATUS_ICONS[props.message.tool_status || ''] || '·')

// `tool_result` is a JSON string from the backend; surface the embedded error
// message for failures so users see why a tool blew up.
const displayResult = computed<string>(() => {
  const raw = props.message.tool_result ?? ''
  if (raw.startsWith('{')) {
    try {
      const parsed = JSON.parse(raw) as { error?: string; message?: string }
      if (typeof parsed.error === 'string') return parsed.error
      if (typeof parsed.message === 'string') return parsed.message
    } catch {
      // fall through to raw
    }
  }
  return raw
})

const onApprove = () => store.confirmToolCall(props.message.id, 'approve')
const onReject = () => store.confirmToolCall(props.message.id, 'reject')
</script>

<style scoped>
.tool-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px 10px;
  border: 1px solid var(--border-color);
  border-left-width: 3px;
  border-radius: var(--radius-sm);
  background: var(--bg-secondary);
  font-family: var(--font-mono);
  font-size: 12px;
}
.tool-card.read {
  border-left-color: var(--accent-sage);
}
.tool-card.write {
  border-left-color: var(--accent-gold);
}
.tool-card.danger {
  border-left-color: var(--accent-crimson);
}
.tool-card.failed {
  border-left-color: var(--accent-crimson);
  background: rgba(216, 111, 84, 0.08);
}
.tool-name {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--text-primary);
}
.tool-name .icon {
  font-size: 11px;
}
.tool-name code {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-primary);
  letter-spacing: 0.02em;
}
.tool-args {
  margin: 0;
  padding: 4px 6px;
  background: rgba(255, 255, 255, 0.02);
  border-radius: 2px;
  color: var(--text-muted);
  font-family: var(--font-mono);
  font-size: 11px;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 80px;
  overflow: auto;
}
.tool-result {
  color: var(--text-primary);
  font-size: 11px;
  white-space: pre-wrap;
  word-break: break-all;
}
.tool-card.failed .tool-result {
  color: var(--accent-crimson);
}
.actions {
  display: flex;
  gap: 6px;
  margin-top: 2px;
}
</style>
