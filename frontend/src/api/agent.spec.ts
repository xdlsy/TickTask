import { describe, it, expect } from 'vitest'
import { api } from './client'

describe('api.agent', () => {
  it('exposes the agent sub-group with expected methods', () => {
    expect(api.agent).toBeDefined()
    expect(typeof api.agent.createConversation).toBe('function')
    expect(typeof api.agent.listConversations).toBe('function')
    expect(typeof api.agent.getMessages).toBe('function')
    expect(typeof api.agent.deleteConversation).toBe('function')
    expect(typeof api.agent.chat).toBe('function')
    expect(typeof api.agent.runTool).toBe('function')
    expect(typeof api.agent.confirm).toBe('function')
    expect(typeof api.agent.status).toBe('function')
  })
})
