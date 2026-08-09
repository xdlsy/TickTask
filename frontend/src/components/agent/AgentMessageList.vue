<template>
  <div ref="root" class="messages" data-testid="messages">
    <AgentTurn v-for="t in turns" :key="t.id" :turn="t" />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import type { AgentMessage } from '@/types'
import AgentTurn from './AgentTurn.vue'
import { useAgentTurns } from './useAgentTurns'

const props = defineProps<{
  messages: AgentMessage[]
  streamingText: string
  isThinking: boolean
}>()

const turns = useAgentTurns(
  computed(() => props.messages),
  computed(() => props.streamingText),
  computed(() => props.isThinking),
)

const root = ref<HTMLElement | null>(null)

// `.messages` itself isn't the scroll container — Element Plus' `.el-drawer__body`
// is. Walk up to the first scrollable ancestor so this stays decoupled from the
// drawer's layout and class names.
function scrollParent(): HTMLElement | null {
  let node: HTMLElement | null = root.value
  while (node) {
    const { overflowY } = window.getComputedStyle(node)
    if (overflowY === 'auto' || overflowY === 'scroll') return node
    node = node.parentElement
  }
  return null
}

const STICK_THRESHOLD_PX = 80

function isNearBottom(): boolean {
  const sc = scrollParent()
  if (!sc) return true
  return sc.scrollHeight - sc.scrollTop - sc.clientHeight <= STICK_THRESHOLD_PX
}

function stickToBottom() {
  const sc = scrollParent()
  if (sc) sc.scrollTop = sc.scrollHeight
}

// Conversation switched/cleared → array replaced. Jump to bottom unconditionally.
watch(
  () => props.messages,
  () => nextTick(stickToBottom),
  { flush: 'post' },
)

// New committed message arrived. User's own input always pins; agent/tool replies
// follow only if the user is still watching the tail.
watch(
  () => props.messages.length,
  (newLen, oldLen) => {
    if (newLen <= (oldLen ?? 0)) return
    const last = props.messages[newLen - 1]
    if (last?.role === 'user' || isNearBottom()) nextTick(stickToBottom)
  },
  { flush: 'post' },
)

// Streaming tokens + typing indicator: follow only while already at the bottom.
watch(
  () => [props.streamingText, props.isThinking],
  () => {
    if (isNearBottom()) nextTick(stickToBottom)
  },
  { flush: 'post' },
)

onMounted(() => nextTick(stickToBottom))
</script>

<style scoped>
.messages {
  display: flex;
  flex-direction: column;
  gap: 18px;
  padding: 4px 0 16px;
}
</style>
