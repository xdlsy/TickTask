import { defineStore } from 'pinia'
import { api } from '@/api/client'
import { wsClient } from '@/utils/websocket'
import type { AgentConversation, AgentMessage, AgentStatus, AgentWsEvent, WSMessage } from '@/types'

interface PendingToolCall {
  messageId: string
  toolName: string
  args: Record<string, unknown>
  preview?: unknown
}

// Note on return types: the `api.agent.*` group returns `AxiosResponse<T>` at
// the type level, but consumers (and the test mock) treat the resolved value as
// the bare payload. The store unwraps via `as unknown as T` so both the unit
// tests and `vue-tsc` agree. Future tasks that re-shape the api group can drop
// these casts.
async function unwrap<T>(p: Promise<unknown>): Promise<T> {
  return (await p) as T
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
      // Accept events when no conversation is bound yet (so streaming works
      // before the first response resolves) or when ids match.
      if (this.currentConvId !== null && e.conversation_id !== this.currentConvId) return
      this.streamingMessageId = e.message_id
      this.streamingText += e.delta_text
    },
    onAgentTool(e: Extract<AgentWsEvent, { type: 'agent_tool' }>) {
      if (this.currentConvId !== null && e.conversation_id !== this.currentConvId) return
      if (e.status === 'pending_confirmation') {
        this.pendingConfirm = {
          messageId: e.message_id!, toolName: e.tool_name, args: e.args, preview: e.preview,
        }
      }
      // Update or append the tool_call message locally based on message_id
    },
    onAgentDone(e: Extract<AgentWsEvent, { type: 'agent_done' }>) {
      if (this.currentConvId !== null && e.conversation_id !== this.currentConvId) return
      if (this.streamingText) {
        this.messages.push({
          id: this.streamingMessageId || 'ast-' + Date.now(),
          conversation_id: this.currentConvId!,
          role: 'assistant', content: this.streamingText, created_at: new Date().toISOString(),
        })
      }
      this.streamingText = ''
      this.streamingMessageId = null
      this.isThinking = false
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

