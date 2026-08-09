import { defineStore } from 'pinia'
import type { AxiosResponse } from 'axios'
import { api } from '@/api/client'
import { wsClient } from '@/utils/websocket'
import type { AgentConversation, AgentMessage, AgentStatus, AgentWsEvent, WSMessage } from '@/types'

interface PendingToolCall {
  messageId: string
  toolName: string
  args: Record<string, unknown>
  preview?: unknown
}

// `api.agent.*` returns `AxiosResponse<T>`; this helper extracts the `.data`
// payload so the store never holds an AxiosResponse in state.
async function unwrap<T>(p: Promise<AxiosResponse<T>>): Promise<T> {
  return (await p).data
}

// Monotonic id for locally-synthesized messages (read-tool events carry no id).
let localSeq = 0
const nextLocalId = (prefix: string) => `${prefix}-${++localSeq}`

function safeStringify(v: unknown): string {
  try { return JSON.stringify(v) } catch { return String(v) }
}

export const useAgentStore = defineStore('agent', {
  state: () => ({
    isOpen: false,
    status: { configured: false, supports_function_calling: false, provider: '' } as AgentStatus,
    conversations: [] as AgentConversation[],
    currentConvId: null as string | null,
    messages: [] as AgentMessage[],
    streamingText: '',
    streamingMessageId: null as string | null,
    pendingConfirm: null as PendingToolCall | null,
    isThinking: false,
  }),
  actions: {
    openDrawer() { this.isOpen = true },
    closeDrawer() { this.isOpen = false },
    toggleDrawer() { this.isOpen = !this.isOpen },
    async checkStatus() { this.status = await unwrap<AgentStatus>(api.agent.status()) },
    async listConversations() {
      const r = await unwrap<{ items: AgentConversation[]; total: number }>(api.agent.listConversations())
      this.conversations = r.items
    },
    async createConversation() {
      const c = await unwrap<AgentConversation>(api.agent.createConversation())
      this.conversations.unshift(c)
      this.currentConvId = c.id
      this.messages = []
      return c
    },
    async switchConversation(id: string) {
      this.currentConvId = id
      this.messages = await unwrap<AgentMessage[]>(api.agent.getMessages(id))
    },
    async sendMessage(text: string) {
      if (!this.currentConvId) await this.createConversation()
      this.messages.push({
        id: 'local-' + Date.now(), conversation_id: this.currentConvId!,
        role: 'user', content: text, created_at: new Date().toISOString(),
      })
      this.isThinking = true
      this.streamingText = ''
      await api.agent.chat(this.currentConvId!, text)
    },
    async runTool(name: string, args: Record<string, unknown>) {
      const r = await unwrap<{ result: unknown }>(api.agent.runTool(name, args))
      return r.result
    },
    async confirmToolCall(messageId: string, decision: 'approve' | 'reject') {
      await api.agent.confirm(messageId, decision)
      this.pendingConfirm = null
    },
    onAgentMessage(e: Extract<AgentWsEvent, { type: 'agent_message' }>) {
      if (this.currentConvId !== null && e.conversation_id !== this.currentConvId) return
      // A new message_id means the prior assistant segment is complete.
      if (this.streamingMessageId !== null && this.streamingMessageId !== e.message_id && this.streamingText) {
        this.flushStreaming()
      }
      this.streamingMessageId = e.message_id
      this.streamingText += e.delta_text
    },
    onAgentTool(e: Extract<AgentWsEvent, { type: 'agent_tool' }>) {
      if (this.currentConvId !== null && e.conversation_id !== this.currentConvId) return
      // Tool arrives after the preceding text segment: flush it so order is text → tool.
      this.flushStreaming()
      if (e.status === 'pending_confirmation') {
        this.pendingConfirm = {
          messageId: e.message_id!, toolName: e.tool_name, args: e.args, preview: e.preview,
        }
      } else if (this.pendingConfirm && e.message_id && e.message_id === this.pendingConfirm.messageId) {
        this.pendingConfirm = null
      }
      const base = {
        role: 'tool_call' as const,
        conversation_id: this.currentConvId!,
        content: '',
        tool_name: e.tool_name,
        tool_args: safeStringify(e.args),
        tool_status: e.status,
        created_at: new Date().toISOString(),
      }
      if (e.message_id) {
        const idx = this.messages.findIndex((m) => m.id === e.message_id)
        const msg: AgentMessage = {
          id: e.message_id, ...base,
          tool_result: e.result != null ? safeStringify(e.result) : (e.error ? `{"error":${JSON.stringify(e.error)}}` : undefined),
        }
        if (idx >= 0) this.messages[idx] = { ...this.messages[idx], ...msg }
        else this.messages.push(msg)
      } else {
        // Read tool: started creates a row; a terminal status updates the most recent
        // same-named row still in 'started' (backend runs tools serially within a turn).
        if (e.status === 'started') {
          this.messages.push({ id: nextLocalId('tool'), ...base })
        } else {
          for (let i = this.messages.length - 1; i >= 0; i--) {
            const m = this.messages[i]
            if (m.tool_name === e.tool_name && m.tool_status === 'started') {
              this.messages[i] = {
                ...m, tool_status: e.status,
                tool_result: e.result != null ? safeStringify(e.result) : (e.error ? `{"error":${JSON.stringify(e.error)}}` : m.tool_result),
              }
              break
            }
          }
        }
      }
    },
    onAgentDone(e: Extract<AgentWsEvent, { type: 'agent_done' }>) {
      if (this.currentConvId !== null && e.conversation_id !== this.currentConvId) return
      this.flushStreaming()
      this.isThinking = false
    },
    flushStreaming() {
      if (this.streamingText) {
        this.messages.push({
          id: this.streamingMessageId || nextLocalId('ast'),
          conversation_id: this.currentConvId!,
          role: 'assistant',
          content: this.streamingText,
          created_at: new Date().toISOString(),
        })
      }
      this.streamingText = ''
      this.streamingMessageId = null
    },
    handleWsEvent(e: AgentWsEvent) {
      if (e.type === 'agent_message') this.onAgentMessage(e)
      else if (e.type === 'agent_tool') this.onAgentTool(e)
      else if (e.type === 'agent_done') this.onAgentDone(e)
    },
  },
})

// Self-register WebSocket listeners (Approach A from the task brief):
// keeps utils/websocket.ts decoupled from stores, avoiding circular imports.
// The registry passes a generic WSMessage; we narrow at the boundary.
function dispatchAgentWs(m: WSMessage) {
  if (m.type === 'agent_message' || m.type === 'agent_tool' || m.type === 'agent_done') {
    useAgentStore().handleWsEvent(m as AgentWsEvent)
  }
}

let wsDispatchRegistered = false
export function setupAgentWsDispatch() {
  if (wsDispatchRegistered) return
  wsDispatchRegistered = true
  wsClient.on('agent_message', dispatchAgentWs)
  wsClient.on('agent_tool', dispatchAgentWs)
  wsClient.on('agent_done', dispatchAgentWs)
}

// Register eagerly on first import.
setupAgentWsDispatch()

