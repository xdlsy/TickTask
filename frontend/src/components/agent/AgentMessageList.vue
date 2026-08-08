<template>
  <div class="messages">
    <div v-for="m in messages" :key="m.id" :class="['msg', m.role]">
      <div class="role">{{ roleLabel(m.role) }}</div>
      <div v-if="m.content" class="bubble">{{ m.content }}</div>
      <ToolCard
        v-if="m.role === 'tool_call' || m.role === 'tool_result'"
        :message="m"
      />
    </div>
    <div v-if="streamingText || isThinking" class="msg agent">
      <div class="role">Agent</div>
      <div class="bubble">
        {{ streamingText
        }}<span v-if="isThinking && !streamingText" class="typing"
          ><span></span><span></span><span></span></span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { AgentMessage } from '@/types'
import ToolCard from './ToolCard.vue'

defineProps<{ messages: AgentMessage[]; streamingText: string; isThinking: boolean }>()

const roleLabel = (r: string) => (r === 'user' ? '你' : 'Agent')
</script>

<style scoped>
.messages {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 4px 0 16px;
}
.msg {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.msg.user {
  align-items: flex-end;
}
.role {
  font-family: var(--font-mono);
  font-size: 10.5px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--text-muted);
}
.bubble {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  padding: 10px 14px;
  color: var(--text-primary);
  font-family: var(--font-body);
  font-size: 13px;
  line-height: 1.6;
  max-width: 90%;
  word-wrap: break-word;
}
.msg.user .bubble {
  background: var(--accent-primary);
  color: var(--bg-primary);
  border-color: transparent;
}
.typing span {
  display: inline-block;
  width: 4px;
  height: 4px;
  margin: 0 1px;
  border-radius: 50%;
  background: currentColor;
  opacity: 0.45;
  animation: blink 1.2s infinite ease-in-out;
}
.typing span:nth-child(2) {
  animation-delay: 0.2s;
}
.typing span:nth-child(3) {
  animation-delay: 0.4s;
}
@keyframes blink {
  0%,
  80%,
  100% {
    opacity: 0.25;
  }
  40% {
    opacity: 0.9;
  }
}
</style>
