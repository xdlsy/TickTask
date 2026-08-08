import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import axios from 'axios'
import { api } from './client'

// Mock axios at module level (same pattern as client.test.ts).
vi.mock('axios', () => {
  const mockAxios = vi.fn()
  mockAxios.create = vi.fn(() => mockAxios)
  mockAxios.get = vi.fn()
  mockAxios.post = vi.fn()
  mockAxios.put = vi.fn()
  mockAxios.delete = vi.fn()
  mockAxios.patch = vi.fn()
  mockAxios.interceptors = {
    request: { use: vi.fn() },
    response: { use: vi.fn() }
  }
  return { default: mockAxios }
})

describe('api.agent', () => {
  let mockAxios: ReturnType<typeof axios.create>

  beforeEach(() => {
    vi.clearAllMocks()
    mockAxios = axios.create()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

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
    expect(typeof api.agent.test).toBe('function')
  })

  it('test() POSTs to /agent/test with empty body when called with no args', async () => {
    ;(mockAxios.post as ReturnType<typeof vi.fn>).mockResolvedValue({
      data: { ok: true, provider: 'openai', latency_ms: 10 }
    })

    await api.agent.test()

    expect(mockAxios.post).toHaveBeenCalledWith('/agent/test', {})
  })

  it('test() forwards optional settings body for temp-key testing', async () => {
    ;(mockAxios.post as ReturnType<typeof vi.fn>).mockResolvedValue({
      data: { ok: true, provider: 'openai', latency_ms: 10 }
    })

    await api.agent.test({ api_key: 'sk-tmp', provider: 'openai' })

    expect(mockAxios.post).toHaveBeenCalledWith('/agent/test', {
      api_key: 'sk-tmp',
      provider: 'openai'
    })
  })
})
