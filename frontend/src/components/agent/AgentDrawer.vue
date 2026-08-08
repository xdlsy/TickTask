<template>
  <el-drawer
    v-if="open"
    v-model="open"
    direction="rtl"
    size="480px"
    :with-header="false"
    data-testid="agent-drawer"
  >
    <div class="drawer-header">
      <span class="title">🤖 Agent</span>
      <div class="actions">
        <el-button text @click="showHistory = !showHistory">
          {{ showHistory ? '返回' : '历史' }}
        </el-button>
        <el-button text @click="close">✕</el-button>
      </div>
    </div>
    <ConversationList v-if="showHistory" />
    <template v-else>
      <AgentMessageList
        :messages="store.messages"
        :streaming-text="store.streamingText"
        :is-thinking="store.isThinking"
      />
      <ToolConfirmDialog v-if="store.pendingConfirm" />
    </template>
    <template #footer>
      <div v-if="!store.status.configured" class="not-configured-hint">
        AI 未配置 — 请先到 <router-link to="/settings">设置</router-link> 填写 API Key,或切换 provider 到 CLI 模式
      </div>
      <AgentInput
        v-else-if="!showHistory"
        :disabled="store.isThinking"
        @send="onSend"
      />
    </template>
  </el-drawer>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useAgentStore } from '@/stores/agent'
import AgentMessageList from './AgentMessageList.vue'
import AgentInput from './AgentInput.vue'
import ConversationList from './ConversationList.vue'
import ToolConfirmDialog from './ToolConfirmDialog.vue'

const store = useAgentStore()
const showHistory = ref(false)
const open = computed({
  get: () => store.isOpen,
  set: (v: boolean) => (v ? store.openDrawer() : store.closeDrawer()),
})
const close = () => store.closeDrawer()
const onSend = (text: string) => store.sendMessage(text)
</script>

<style scoped>
.drawer-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 0 12px;
  border-bottom: 1px solid var(--border-color);
  margin-bottom: 12px;
}
.title {
  font-family: var(--font-display);
  font-variation-settings: 'opsz' 60;
  font-size: 16px;
  font-weight: 480;
  color: var(--text-primary);
  letter-spacing: -0.02em;
}
.actions {
  display: flex;
  gap: 4px;
}
.not-configured-hint {
  font-size: 12px;
  color: var(--text-muted);
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  padding: 10px 12px;
  line-height: 1.5;
}
.not-configured-hint a {
  color: var(--accent-primary);
  text-decoration: underline;
}
</style>
