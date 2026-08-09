<template>
  <div class="turn">
    <div v-if="turn.user" class="msg user">
      <div class="role">YOU</div>
      <div class="bubble">{{ turn.user.content }}</div>
    </div>

    <div v-if="turn.segments.length || turn.live" class="agent-block">
      <div class="ava">A</div>
      <div class="agent-body">
        <template v-for="s in turn.segments" :key="s.message.id">
          <MarkdownText v-if="s.kind === 'text'" :content="s.message.content" class="bubble" />
          <ToolRow v-else :message="s.message" />
        </template>

        <div v-if="turn.live && turn.live.text" class="bubble live-stream">
          <MarkdownText :content="turn.live.text" />
          <span class="caret" />
        </div>
        <div v-else-if="turn.live" class="pulse"><span /></div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import MarkdownText from './MarkdownText.vue'
import ToolRow from './ToolRow.vue'
import type { Turn } from './useAgentTurns'
defineProps<{ turn: Turn }>()
</script>

<style scoped>
.turn { display: flex; flex-direction: column; gap: 10px; }
.msg { display: flex; flex-direction: column; gap: 4px; }
.msg.user { align-items: flex-end; }
.role {
  font-family: var(--font-mono);
  font-size: 10px;
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
  font-size: 13px;
  line-height: 1.6;
  max-width: 90%;
}
.msg.user .bubble {
  background: var(--accent-primary);
  color: var(--bg-primary);
  border-color: transparent;
}
.agent-block { display: flex; gap: 10px; }
.ava {
  width: 26px;
  height: 26px;
  flex-shrink: 0;
  border-radius: 50%;
  background: var(--bg-elevated);
  border: 1px solid rgba(230, 162, 60, 0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: var(--font-display);
  font-size: 13px;
  color: var(--accent-primary);
}
.agent-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-top: 2px;
}
.agent-body :deep(.tool-row) { margin-left: 2px; }
.live-stream { display: flex; flex-direction: column; }
.caret {
  display: inline-block;
  width: 6px;
  height: 13px;
  background: var(--accent-primary);
  margin-top: 2px;
  animation: caret-blink 1s steps(1) infinite;
}
@keyframes caret-blink { 0%, 100% { opacity: 1; } 50% { opacity: 0; } }
.pulse { display: flex; align-items: center; padding: 6px 0; }
.pulse span {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--accent-primary);
  animation: pulse-glow 1.2s infinite ease-out;
}
@keyframes pulse-glow {
  0%, 100% { box-shadow: 0 0 0 0 rgba(230, 162, 60, 0.5); }
  50% { box-shadow: 0 0 0 5px rgba(230, 162, 60, 0); }
}
</style>
