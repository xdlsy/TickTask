import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { wsClient } from '@/utils/websocket'
import type { WSMessageHandler } from '@/utils/websocket'

class MockWebSocket {
  url: string
  readyState: number
  onopen: (() => void) | null
  onmessage: ((event: MessageEvent) => void) | null
  onclose: (() => void) | null
  onerror: ((error: Event) => void) | null

  static OPEN = 1
  static CLOSED = 3

  constructor(url: string) {
    this.url = url
    this.readyState = MockWebSocket.CLOSED
    this.onopen = null
    this.onmessage = null
    this.onclose = null
    this.onerror = null
  }
}

describe('WebSocket Client', () => {
  let mockWs: MockWebSocket | null = null

  beforeEach(() => {
    vi.useFakeTimers()
    vi.spyOn(console, 'error').mockImplementation(() => {})
    vi.spyOn(console, 'log').mockImplementation(() => {})

    mockWs = null
    const wsFn = vi.fn((url: string) => {
      mockWs = new MockWebSocket(url)
      return mockWs as unknown as WebSocket
    })
    ;(wsFn as any).OPEN = MockWebSocket.OPEN
    ;(wsFn as any).CLOSED = MockWebSocket.CLOSED
    vi.stubGlobal('WebSocket', wsFn)
  })

  afterEach(() => {
    vi.restoreAllMocks()
    wsClient['handlers'].clear()
    wsClient['ws'] = null
    wsClient['reconnectAttempts'] = 0
  })

  describe('on/off', () => {
    it('should register and unregister handler', () => {
      const handler: WSMessageHandler = vi.fn()
      wsClient.on('timer_tick', handler)

      wsClient['handleMessage']({
        type: 'timer_tick',
        session_id: '1',
        remaining: 100,
        total: 200,
        percentage: 50
      })

      expect(handler).toHaveBeenCalledTimes(1)

      wsClient.off('timer_tick', handler)
      wsClient['handleMessage']({
        type: 'timer_tick',
        session_id: '1',
        remaining: 100,
        total: 200,
        percentage: 50
      })

      expect(handler).toHaveBeenCalledTimes(1)
    })

    it('should not fail when unregistering from empty handlers', () => {
      const handler: WSMessageHandler = vi.fn()
      expect(() => wsClient.off('timer_tick', handler)).not.toThrow()
    })

    it('should call all registered handlers for a message type', () => {
      const h1 = vi.fn()
      const h2 = vi.fn()
      wsClient.on('timer_complete', h1)
      wsClient.on('timer_complete', h2)

      wsClient['handleMessage']({ type: 'timer_complete', session_id: '1' })

      expect(h1).toHaveBeenCalledTimes(1)
      expect(h2).toHaveBeenCalledTimes(1)
    })

    it('should not call handlers for different message type', () => {
      const handler: WSMessageHandler = vi.fn()
      wsClient.on('timer_tick', handler)

      wsClient['handleMessage']({ type: 'session_state', id: '1', status: 'running' })

      expect(handler).not.toHaveBeenCalled()
    })
  })

  describe('connect', () => {
    it('should create a WebSocket connection', () => {
      wsClient.connect()

      expect(globalThis.WebSocket).toHaveBeenCalledWith('ws://localhost:8080/ws')
    })

    it('should not reconnect if already open', () => {
      wsClient.connect()
      expect(globalThis.WebSocket).toHaveBeenCalledTimes(1)

      if (mockWs) mockWs.readyState = MockWebSocket.OPEN
      wsClient.connect()

      expect(globalThis.WebSocket).toHaveBeenCalledTimes(1)
    })

    it('should handle WebSocket onopen event', () => {
      wsClient.connect()
      mockWs!.onopen?.()

      expect(wsClient['reconnectAttempts']).toBe(0)
    })

    it('should handle WebSocket onmessage event', () => {
      const handler: WSMessageHandler = vi.fn()
      wsClient.on('timer_tick', handler)
      wsClient.connect()

      mockWs!.onmessage?.(new MessageEvent('message', {
        data: JSON.stringify({
          type: 'timer_tick',
          session_id: 's1',
          remaining: 50,
          total: 100,
          percentage: 50
        })
      }))

      expect(handler).toHaveBeenCalled()
    })

    it('should handle invalid JSON in onmessage', () => {
      wsClient.connect()
      mockWs!.onmessage?.(new MessageEvent('message', { data: 'invalid json' }))

      expect(console.error).toHaveBeenCalledWith(
        'Failed to parse WebSocket message:',
        expect.any(Error)
      )
    })
  })

  describe('disconnect', () => {
    it('should close and nullify WebSocket', () => {
      wsClient.connect()
      const closeSpy = vi.fn()
      if (mockWs) mockWs.close = closeSpy

      wsClient.disconnect()

      expect(closeSpy).toHaveBeenCalled()
      expect(wsClient['ws']).toBeNull()
    })
  })

  describe('send', () => {
    it('should send JSON data through WebSocket', () => {
      wsClient.connect()
      const sendSpy = vi.fn()
      if (mockWs) mockWs.send = sendSpy

      wsClient.send({ type: 'test', data: 'hello' })

      expect(sendSpy).toHaveBeenCalledWith(JSON.stringify({ type: 'test', data: 'hello' }))
    })

    it('should not throw when sending without connection', () => {
      expect(() => wsClient.send({ data: 'test' })).not.toThrow()
    })
  })

  describe('reconnect', () => {
    it('should attempt reconnect on close', () => {
      wsClient.connect()
      // Track if setTimeout was called via advancing timers
      const beforeCount = (globalThis.WebSocket as ReturnType<typeof vi.fn>).mock.calls.length

      mockWs!.onclose?.()

      // setTimeout should have been scheduled; advancing should trigger reconnect
      vi.advanceTimersByTime(1000)
      expect((globalThis.WebSocket as ReturnType<typeof vi.fn>).mock.calls.length).toBeGreaterThan(beforeCount)
    })

    it('should use exponential backoff', () => {
      wsClient.connect()
      wsClient['reconnectAttempts'] = 1

      const beforeCount = (globalThis.WebSocket as ReturnType<typeof vi.fn>).mock.calls.length
      mockWs!.onclose?.()

      // 1000ms should NOT trigger reconnect (delay is 2000ms)
      vi.advanceTimersByTime(1000)
      expect((globalThis.WebSocket as ReturnType<typeof vi.fn>).mock.calls.length).toBe(beforeCount)

      // 2000ms total should trigger it
      vi.advanceTimersByTime(1000)
      expect((globalThis.WebSocket as ReturnType<typeof vi.fn>).mock.calls.length).toBeGreaterThan(beforeCount)
    })

    it('should stop reconnecting after max attempts', () => {
      wsClient.connect()
      wsClient['reconnectAttempts'] = 5

      const beforeCount = (globalThis.WebSocket as ReturnType<typeof vi.fn>).mock.calls.length
      mockWs!.onclose?.()

      vi.advanceTimersByTime(10000)
      expect((globalThis.WebSocket as ReturnType<typeof vi.fn>).mock.calls.length).toBe(beforeCount)
    })
  })
})
