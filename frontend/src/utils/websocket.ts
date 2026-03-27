import type { WSMessage, WSMessageType } from '@/types'

export type WSMessageHandler = (message: WSMessage) => void

class WebSocketClient {
  private ws: WebSocket | null = null
  private handlers: Map<WSMessageType, Set<WSMessageHandler>> = new Map()
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5
  private reconnectDelay = 1000

  connect(url: string = 'ws://localhost:8080/ws') {
    if (this.ws?.readyState === WebSocket.OPEN) {
      return
    }

    try {
      this.ws = new WebSocket(url)

      this.ws.onopen = () => {
        console.log('WebSocket connected')
        this.reconnectAttempts = 0
      }

      this.ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data) as WSMessage
          this.handleMessage(message)
        } catch (e) {
          console.error('Failed to parse WebSocket message:', e)
        }
      }

      this.ws.onclose = () => {
        console.log('WebSocket disconnected')
        this.attemptReconnect(url)
      }

      this.ws.onerror = (error) => {
        console.error('WebSocket error:', error)
      }
    } catch (error) {
      console.error('Failed to create WebSocket connection:', error)
    }
  }

  private attemptReconnect(url: string) {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++
      const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1)
      console.log(`Attempting to reconnect in ${delay}ms...`)
      setTimeout(() => this.connect(url), delay)
    }
  }

  disconnect() {
    this.ws?.close()
    this.ws = null
  }

  on(messageType: WSMessageType, handler: WSMessageHandler) {
    if (!this.handlers.has(messageType)) {
      this.handlers.set(messageType, new Set())
    }
    this.handlers.get(messageType)!.add(handler)
  }

  off(messageType: WSMessageType, handler: WSMessageHandler) {
    this.handlers.get(messageType)?.delete(handler)
  }

  private handleMessage(message: WSMessage) {
    const handlers = this.handlers.get(message.type)
    if (handlers) {
      handlers.forEach(handler => handler(message))
    }
  }

  send(data: any) {
    this.ws?.send(JSON.stringify(data))
  }
}

export const wsClient = new WebSocketClient()
