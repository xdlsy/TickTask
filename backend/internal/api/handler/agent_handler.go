package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"ticktask/internal/agent"
	"ticktask/internal/repository"
)

// AgentHandler wires the 8 endpoints under /api/agent to the AgentService and
// AgentRepository.
type AgentHandler interface {
	Register(rg *gin.RouterGroup)
}

type agentHandler struct {
	Svc  agent.AgentService
	Repo repository.AgentRepository
}

// NewAgentHandler constructs an AgentHandler bound to the given service and repo.
func NewAgentHandler(svc agent.AgentService, repo repository.AgentRepository) AgentHandler {
	return &agentHandler{Svc: svc, Repo: repo}
}

func (h *agentHandler) Register(rg *gin.RouterGroup) {
	rg.POST("/conversations", h.createConversation)
	rg.GET("/conversations", h.listConversations)
	rg.GET("/conversations/:id/messages", h.getMessages)
	rg.DELETE("/conversations/:id", h.deleteConversation)
	rg.POST("/chat", h.chat)
	rg.POST("/run-tool", h.runTool)
	rg.POST("/confirm", h.confirm)
	rg.GET("/status", h.status)
}

// POST /conversations — create a fresh conversation.
func (h *agentHandler) createConversation(c *gin.Context) {
	conv, err := h.Repo.CreateConversation()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, conv)
}

// GET /conversations?page=&size= — paginated list.
func (h *agentHandler) listConversations(c *gin.Context) {
	page := atoiDefault(c.Query("page"), 1)
	size := atoiDefault(c.Query("size"), 20)
	items, total, err := h.Repo.ListConversations(page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

// GET /conversations/:id/messages — full message history, oldest first.
func (h *agentHandler) getMessages(c *gin.Context) {
	msgs, err := h.Repo.ListMessages(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, msgs)
}

// DELETE /conversations/:id — remove a conversation and its messages.
func (h *agentHandler) deleteConversation(c *gin.Context) {
	if err := h.Repo.DeleteConversation(c.Param("id")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// chatInput is the body for POST /chat.
type chatInput struct {
	ConversationID string `json:"conversation_id"`
	Text           string `json:"text"`
}

// POST /chat — accept the user message and immediately return 202. The agent's
// runTurn runs in a background goroutine and streams events over WebSocket.
//
// The goroutine MUST NOT bind to c.Request.Context(): that context is cancelled
// the moment this handler returns (right after writing the 202), which would
// cancel the agent's runTurn mid-flight. The agent's runTurn is bounded by its
// own MaxToolCallsPerTurn + ConfirmationTimeout instead.
func (h *agentHandler) chat(c *gin.Context) {
	var in chatInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	go func() {
		_ = h.Svc.SendMessage(context.Background(), in.ConversationID, in.Text)
	}()
	c.Status(http.StatusAccepted)
}

// runToolInput is the body for POST /run-tool.
type runToolInput struct {
	Tool string          `json:"tool"`
	Args json.RawMessage `json:"args"`
}

// POST /run-tool — headless one-shot tool execution (no conversation).
func (h *agentHandler) runTool(c *gin.Context) {
	var in runToolInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := h.Svc.RunTool(c.Request.Context(), in.Tool, in.Args)
	if err != nil {
		if errors.Is(err, agent.ErrToolNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": res})
}

// confirmInput is the body for POST /confirm.
type confirmInput struct {
	MessageID string `json:"message_id"`
	Decision  string `json:"decision"`
}

// POST /confirm — approve or reject a pending PermWrite/Dangerous tool call.
func (h *agentHandler) confirm(c *gin.Context) {
	var in confirmInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.Svc.Confirm(c.Request.Context(), in.MessageID, in.Decision); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

// GET /status — runtime capability flags surfaced to the frontend.
func (h *agentHandler) status(c *gin.Context) {
	c.JSON(http.StatusOK, h.Svc.Status())
}

// atoiDefault parses a decimal integer with a fallback. Negative or unparsable
// values fall back to def.
func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return def
	}
	return n
}
