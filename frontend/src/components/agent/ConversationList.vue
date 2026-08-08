<template>
  <div class="conversation-list" data-testid="conversation-list">
    <div class="header">
      <span class="title">历史会话</span>
      <el-button
        size="small"
        type="primary"
        data-testid="new-conversation"
        :loading="creating"
        @click="onCreate"
      >
        + 新建
      </el-button>
    </div>
    <div v-if="loading" class="state">加载中…</div>
    <div v-else-if="store.conversations.length === 0" class="state empty">
      暂无历史会话
    </div>
    <ul v-else class="items">
      <li
        v-for="c in store.conversations"
        :key="c.id"
        :class="['item', { active: c.id === store.currentConvId }]"
        data-testid="conversation-item"
        @click="onSwitch(c.id)"
      >
        <div class="item-title">{{ c.title || '(未命名)' }}</div>
        <div class="item-meta">
          <span class="count">{{ c.message_count }} 条</span>
          <span class="time">{{ formatTime(c.updated_at || c.created_at) }}</span>
        </div>
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useAgentStore } from '@/stores/agent'

const store = useAgentStore()
const loading = ref(false)
const creating = ref(false)

onMounted(async () => {
  loading.value = true
  try {
    await store.listConversations()
  } finally {
    loading.value = false
  }
})

async function onSwitch(id: string) {
  if (id === store.currentConvId) return
  await store.switchConversation(id)
}

async function onCreate() {
  if (creating.value) return
  creating.value = true
  try {
    await store.createConversation()
  } finally {
    creating.value = false
  }
}

function formatTime(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  const now = new Date()
  const sameDay = d.toDateString() === now.toDateString()
  if (sameDay) {
    return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  }
  return d.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' })
}
</script>

<style scoped>
.conversation-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 4px 0 16px;
}
.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.title {
  font-family: var(--font-display);
  font-variation-settings: 'opsz' 60;
  font-size: 15px;
  font-weight: 480;
  color: var(--text-primary);
  letter-spacing: -0.02em;
}
.state {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-muted);
  padding: 12px 0;
  text-align: center;
}
.state.empty {
  color: var(--text-muted);
}
.items {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.item {
  padding: 8px 10px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  background: var(--bg-secondary);
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: 2px;
  transition: border-color 0.15s ease, background 0.15s ease;
}
.item:hover {
  border-color: var(--accent-primary);
}
.item.active {
  border-left: 3px solid var(--accent-primary);
  padding-left: 8px;
  background: rgba(230, 162, 60, 0.06);
}
.item-title {
  font-family: var(--font-body);
  font-size: 13px;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.item-meta {
  display: flex;
  gap: 8px;
  font-family: var(--font-mono);
  font-size: 10px;
  color: var(--text-muted);
  letter-spacing: 0.05em;
}
</style>
