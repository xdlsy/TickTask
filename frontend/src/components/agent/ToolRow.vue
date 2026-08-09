<template>
  <div :class="['tool-row', perm, statusClass]" data-testid="tool-row">
    <span class="dot" />
    <div class="main">
      <div class="head">
        <code class="name">{{ message.tool_name }}</code>
        <span v-if="summary.argHint" class="arg">{{ summary.argHint }}</span>
        <span class="status" :class="statusClass">{{ statusLabel }}</span>
        <span v-if="summary.resultHint" class="res">{{ summary.resultHint }}</span>
        <button class="toggle" data-testid="tool-toggle" @click="open = !open">⌄</button>
      </div>
      <div v-if="message.tool_status === 'failed' && errorText" class="err">{{ errorText }}</div>

      <div v-if="message.tool_status === 'pending_confirmation'" class="confirm">
        <pre v-if="previewText" class="preview">{{ previewText }}</pre>
        <div class="actions">
          <el-button size="small" type="primary" data-testid="tool-confirm-approve" @click="decide('approve')">✓ 批准</el-button>
          <el-button size="small" data-testid="tool-confirm-reject" @click="decide('reject')">✕ 拒绝</el-button>
        </div>
      </div>

      <pre v-if="open" class="json" data-testid="tool-json">{{ prettyArgs }}<template v-if="message.tool_result">
{{ prettyResult }}</template></pre>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useAgentStore } from '@/stores/agent'
import type { AgentMessage } from '@/types'
import { classifyPermission, summarizeTool } from './toolFormatters'

const props = defineProps<{ message: AgentMessage }>()
const store = useAgentStore()
const open = ref(false)

const perm = computed(() => {
  if (props.message.tool_status === 'failed') return 'failed'
  return classifyPermission(props.message.tool_name)
})
const statusClass = computed(() => `s-${props.message.tool_status || 'unknown'}`)
const summary = computed(() => summarizeTool(props.message))

const STATUS_LABELS: Record<string, string> = {
  started: '执行中',
  pending_confirmation: '待确认',
  succeeded: '已执行',
  failed: '失败',
  rejected: '已取消',
}
const statusLabel = computed(() => STATUS_LABELS[props.message.tool_status || ''] || props.message.tool_status || '')

const errorText = computed(() => {
  const raw = props.message.tool_result ?? ''
  if (raw.startsWith('{')) {
    try {
      const p = JSON.parse(raw) as { error?: string; message?: string }
      if (typeof p.error === 'string') return p.error
      if (typeof p.message === 'string') return p.message
    } catch { /* fall through */ }
  }
  return raw
})

const previewText = computed(() => {
  const pc = store.pendingConfirm
  if (!pc || pc.messageId !== props.message.id) return ''
  const p = pc.preview
  if (p == null || p === '') return ''
  if (typeof p === 'string') return p
  try { return JSON.stringify(p, null, 2) } catch { return String(p) }
})

function pretty(s: string | undefined): string {
  if (!s) return ''
  try { return JSON.stringify(JSON.parse(s), null, 2) } catch { return s }
}
const prettyArgs = computed(() => pretty(props.message.tool_args))
const prettyResult = computed(() => pretty(props.message.tool_result))

function decide(d: 'approve' | 'reject') {
  store.confirmToolCall(props.message.id, d)
}
</script>

<style scoped>
.tool-row {
  position: relative;
  display: flex;
  gap: 8px;
  padding: 2px 0 2px 14px;
  border-left: 1.5px dashed var(--border-color);
  font-family: var(--font-mono);
  font-size: 12px;
}
.tool-row .dot {
  position: absolute;
  left: -6px;
  top: 6px;
  width: 9px;
  height: 9px;
  border-radius: 50%;
  background: var(--text-muted);
}
.tool-row.read .dot { background: var(--accent-sage); }
.tool-row.write .dot { background: var(--accent-gold); }
.tool-row.danger .dot { background: var(--accent-crimson); }
.tool-row.failed .dot { background: var(--accent-crimson); }
.tool-row.s-pending_confirmation .dot { background: var(--accent-gold); }
.tool-row.s-started .dot {
  background: var(--accent-gold);
  animation: trow-glow 1.2s infinite ease-out;
}
@keyframes trow-glow {
  0%, 100% { box-shadow: 0 0 0 0 rgba(214, 180, 90, 0.5); }
  50% { box-shadow: 0 0 0 4px rgba(214, 180, 90, 0); }
}
.main { flex: 1; min-width: 0; }
.head { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
.name { color: var(--text-primary); font-size: 11.5px; letter-spacing: 0.02em; }
.arg { color: var(--text-muted); font-size: 10px; }
.status { font-size: 9px; color: var(--text-muted); }
.status.s-succeeded { color: var(--accent-sage); }
.status.s-failed { color: var(--accent-crimson); }
.status.s-rejected { color: var(--text-muted); }
.status.s-pending_confirmation { color: var(--accent-gold); }
.res { color: var(--accent-sage); font-size: 9.5px; }
.toggle {
  margin-left: auto;
  background: transparent;
  border: 0;
  color: var(--text-muted);
  cursor: pointer;
  font-size: 11px;
  padding: 0 2px;
}
.err { color: var(--accent-crimson); font-size: 10.5px; margin-top: 2px; }
.confirm { margin-top: 6px; display: flex; flex-direction: column; gap: 6px; }
.preview {
  margin: 0;
  padding: 6px 8px;
  background: rgba(214, 180, 90, 0.06);
  border: 1px solid rgba(214, 180, 90, 0.3);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  font-size: 10px;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 140px;
  overflow: auto;
}
.actions { display: flex; gap: 6px; }
.json {
  margin: 4px 0 0;
  padding: 6px 8px;
  background: rgba(239, 231, 215, 0.04);
  border-radius: 4px;
  color: var(--text-secondary);
  font-size: 10px;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 180px;
  overflow: auto;
}
</style>
