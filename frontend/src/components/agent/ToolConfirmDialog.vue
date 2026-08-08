<template>
  <div v-if="pending" class="tool-confirm" data-testid="tool-confirm">
    <div class="header">
      <span class="icon">⏸</span>
      <span class="label">工具调用待确认</span>
    </div>
    <div class="tool-line">
      <code class="tool-name">{{ pending.toolName }}</code>
    </div>
    <pre v-if="argsText" class="args">{{ argsText }}</pre>
    <div v-if="previewText" class="preview">
      <div class="preview-label">预览</div>
      <pre class="preview-body">{{ previewText }}</pre>
    </div>
    <div class="actions">
      <el-button
        size="small"
        type="primary"
        data-testid="confirm-approve"
        :loading="busy === 'approve'"
        @click="onApprove"
      >
        ✓ 确认执行
      </el-button>
      <el-button
        size="small"
        data-testid="confirm-reject"
        :loading="busy === 'reject'"
        @click="onReject"
      >
        ✕ 拒绝
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useAgentStore } from '@/stores/agent'

const store = useAgentStore()
const busy = ref<'approve' | 'reject' | null>(null)

const pending = computed(() => store.pendingConfirm)

const argsText = computed(() => {
  const args = pending.value?.args
  if (!args || Object.keys(args).length === 0) return ''
  try {
    return JSON.stringify(args, null, 2)
  } catch {
    return String(args)
  }
})

const previewText = computed(() => {
  const p = pending.value?.preview
  if (p == null || p === '') return ''
  if (typeof p === 'string') return p
  try {
    return JSON.stringify(p, null, 2)
  } catch {
    return String(p)
  }
})

async function decide(decision: 'approve' | 'reject') {
  if (!pending.value || busy.value) return
  busy.value = decision
  try {
    await store.confirmToolCall(pending.value.messageId, decision)
  } finally {
    busy.value = null
  }
}

const onApprove = () => decide('approve')
const onReject = () => decide('reject')
</script>

<style scoped>
.tool-confirm {
  margin-top: 12px;
  padding: 10px 12px;
  border: 1px solid var(--accent-gold);
  border-left-width: 3px;
  border-radius: var(--radius-sm);
  background: rgba(214, 180, 90, 0.06);
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.header {
  display: flex;
  align-items: center;
  gap: 6px;
  font-family: var(--font-mono);
  font-size: 10.5px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--accent-gold);
}
.header .icon {
  font-size: 12px;
}
.tool-name {
  font-family: var(--font-mono);
  font-size: 13px;
  color: var(--text-primary);
}
.args,
.preview-body {
  margin: 0;
  padding: 6px 8px;
  background: rgba(255, 255, 255, 0.02);
  border-radius: 2px;
  color: var(--text-secondary);
  font-family: var(--font-mono);
  font-size: 11px;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 160px;
  overflow: auto;
}
.preview-label {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--text-muted);
  margin-bottom: 2px;
}
.actions {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
}
</style>
