package websocket

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type Hub struct {
	clients map[*Client]bool
	mu      sync.RWMutex
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 开发环境允许所有来源
	},
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[*Client]bool),
	}
}

func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client] = true
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
	}
}

func (h *Hub) Broadcast(message interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	data, err := json.Marshal(message)
	if err != nil {
		return
	}

	for client := range h.clients {
		select {
		case client.send <- data:
		default:
			delete(h.clients, client)
			close(client.send)
		}
	}
}

type TimerTickMessage struct {
	Type       string `json:"type"` // "timer_tick"
	SessionID  string `json:"session_id"`
	Remaining  int    `json:"remaining"`
	Total      int    `json:"total"`
	Percentage int    `json:"percentage"`
}

func (h *Hub) BroadcastTimerTick(sessionID string, remaining, total, percentage int) {
	h.Broadcast(TimerTickMessage{
		Type:       "timer_tick",
		SessionID:  sessionID,
		Remaining:  remaining,
		Total:      total,
		Percentage: percentage,
	})
}

type SessionStateMessage struct {
	Type   string `json:"type"` // "session_state"
	ID     string `json:"id"`
	Status string `json:"status"`
}

func (h *Hub) BroadcastSessionState(sessionID string, status string) {
	h.Broadcast(SessionStateMessage{
		Type:   "session_state",
		ID:     sessionID,
		Status: string(status),
	})
}

type TimerCompleteMessage struct {
	Type      string `json:"type"` // "timer_complete"
	SessionID string `json:"session_id"`
}

func (h *Hub) BroadcastTimerComplete(sessionID string) {
	h.Broadcast(TimerCompleteMessage{
		Type:      "timer_complete",
		SessionID: sessionID,
	})
}

type TaskUpdatedMessage struct {
	Type  string `json:"type"` // "task_updated"
	Task  any    `json:"task"`
}

func (h *Hub) BroadcastTaskUpdated(task any) {
	h.Broadcast(TaskUpdatedMessage{
		Type: "task_updated",
		Task: task,
	})
}

// TerminalOutputMessage 终端输出流式推送
type TerminalOutputMessage struct {
	Type     string `json:"type"` // "terminal_output"
	Chunk    string `json:"chunk"`
	IsStderr bool   `json:"is_stderr"`
}

func (h *Hub) BroadcastTerminalOutput(chunk string, isStderr bool) {
	h.Broadcast(TerminalOutputMessage{
		Type:     "terminal_output",
		Chunk:    chunk,
		IsStderr: isStderr,
	})
}

// TerminalStatusMessage 终端状态变更推送
type TerminalStatusMessage struct {
	Type    string `json:"type"` // "terminal_status"
	Status  string `json:"status"`
	Message string `json:"message"`
	Detail  string `json:"detail"`
}

func (h *Hub) BroadcastTerminalStatus(status, message, detail string) {
	h.Broadcast(TerminalStatusMessage{
		Type:    "terminal_status",
		Status:  status,
		Message: message,
		Detail:  detail,
	})
}

// WebSocketHandler 处理 WebSocket 连接
func (h *Hub) WebSocketHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := &Client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 256),
	}

	h.Register(client)
	defer h.Unregister(client)

	// 启动读取和写入协程
	go client.writePump()
	client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *Client) writePump() {
	defer c.conn.Close()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		}
	}
}
