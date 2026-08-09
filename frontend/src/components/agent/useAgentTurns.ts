import { computed, type ComputedRef, type Ref } from 'vue'
import type { AgentMessage } from '@/types'

export type Segment =
  | { kind: 'text'; message: AgentMessage }
  | { kind: 'tool'; message: AgentMessage }

export interface Turn {
  id: string
  user?: AgentMessage
  segments: Segment[]
  live?: { text: string }
}

/**
 * Group a flat message list into turns. A turn = one user message plus every
 * following non-user message up to the next user message. The in-flight
 * streaming state (streamingText/isThinking) attaches a `live` segment to the
 * last turn so the view can render the typing indicator / streaming bubble.
 */
export function groupIntoTurns(
  messages: AgentMessage[],
  streamingText: string,
  isThinking: boolean,
): Turn[] {
  const turns: Turn[] = []
  let current: Turn | null = null

  for (const m of messages) {
    if (m.role === 'user') {
      current = { id: m.id, user: m, segments: [] }
      turns.push(current)
      continue
    }
    if (!current) {
      // Orphan assistant/tool before any user (defensive): give it its own turn.
      current = { id: 'orphan-' + m.id, segments: [] }
      turns.push(current)
    }
    if (m.role === 'assistant') {
      if (!m.content) continue // skip empty bubbles
      current.segments.push({ kind: 'text', message: m })
    } else {
      // tool_call / tool_result
      current.segments.push({ kind: 'tool', message: m })
    }
  }

  if (streamingText || isThinking) {
    let last = turns[turns.length - 1]
    if (!last) {
      last = { id: 'live', segments: [] }
      turns.push(last)
    }
    last.live = { text: streamingText }
  }

  return turns
}

/** Reactive wrapper for use in components. */
export function useAgentTurns(
  messages: Ref<AgentMessage[]> | ComputedRef<AgentMessage[]>,
  streamingText: Ref<string>,
  isThinking: Ref<boolean>,
): ComputedRef<Turn[]> {
  return computed(() => groupIntoTurns(messages.value, streamingText.value, isThinking.value))
}
