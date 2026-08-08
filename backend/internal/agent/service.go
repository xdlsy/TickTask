package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"ticktask/internal/ai"
	"ticktask/internal/repository"
	"ticktask/internal/websocket"
)

// HubBroadcaster abstracts the websocket.Hub so the agent service can broadcast
// events without depending on the concrete Hub type. Mirrors *websocket.Hub's
// Broadcast(message interface{}) signature.
type HubBroadcaster interface {
	Broadcast(message interface{})
}

type AgentDeps struct {
	Repo     repository.AgentRepository
	LLM      ai.LLMClient
	Registry ToolRegistry
	Hub      HubBroadcaster
	System   string
}

type AgentService interface {
	SendMessage(ctx context.Context, convID, text string) error
	Confirm(ctx context.Context, msgID string, decision string) error
	RunTool(ctx context.Context, name string, args json.RawMessage) (any, error)
}

type agentService struct {
	AgentDeps
	pending   map[string]chan string // msgID -> decision channel
	pendingMu sync.Mutex
}

func NewAgentService(d AgentDeps) AgentService {
	if d.System == "" {
		d.System = DefaultSystemPrompt
	}
	return &agentService{AgentDeps: d, pending: make(map[string]chan string)}
}

func (s *agentService) SendMessage(ctx context.Context, convID, text string) error {
	if _, err := s.Repo.GetConversation(convID); err != nil {
		return err
	}
	if _, err := s.Repo.AppendMessage(convID, "user", text, nil, nil, nil, nil); err != nil {
		return err
	}
	return s.runTurn(ctx, convID, 0)
}

func (s *agentService) runTurn(ctx context.Context, convID string, toolCount int) error {
	for toolCount < MaxToolCallsPerTurn {
		history, err := s.Repo.LoadRecentMessages(convID, MaxContextMessages)
		if err != nil {
			return err
		}
		msgs := buildLLMMessages(s.System, history)
		resp, err := s.LLM.ChatWithTools(ctx, msgs, s.Registry.ToOpenAITools())
		if err != nil {
			s.broadcastDone(convID, "error")
			return err
		}
		if resp.Content != "" {
			s.broadcast(convID, websocket.EventAgentMessage, map[string]any{
				"conversation_id": convID, "delta_text": resp.Content,
			})
			if _, err := s.Repo.AppendMessage(convID, "assistant", resp.Content, nil, nil, nil, nil); err != nil {
				s.broadcastDone(convID, "error")
				return err
			}
		}
		if len(resp.ToolCalls) == 0 {
			s.broadcastDone(convID, "stop")
			return nil
		}
		for _, tc := range resp.ToolCalls {
			toolCount++
			tool, err := s.Registry.Lookup(tc.Name)
			if err != nil {
				s.broadcastTool(convID, "", tc.Name, tc.Args, "failed", nil, fmt.Sprintf("tool not found: %s", tc.Name))
				s.appendToolResult(convID, tc, "failed", `{"error":"not found"}`)
				continue
			}
			perm := tool.Schema().Permission
			if perm == PermRead {
				s.broadcastTool(convID, "", tc.Name, tc.Args, "started", nil, "")
				result, err := tool.Execute(ctx, tc.Args)
				if err != nil {
					s.broadcastTool(convID, "", tc.Name, tc.Args, "failed", nil, err.Error())
					s.appendToolResult(convID, tc, "failed", fmt.Sprintf(`{"error":%q}`, err.Error()))
				} else {
					rjson, _ := json.Marshal(result)
					s.broadcastTool(convID, "", tc.Name, tc.Args, "succeeded", result, "")
					s.appendToolResult(convID, tc, "succeeded", string(rjson))
				}
			} else {
				// PermWrite / PermDangerous — require user confirmation
				preview, _ := tool.Preview(ctx, tc.Args)
				status := "pending_confirmation"
				msgID, _ := s.Repo.AppendMessage(convID, "tool_call", "", &tc.Name, strPtr(string(tc.Args)), nil, &status)
				s.broadcastTool(convID, msgID, tc.Name, tc.Args, "pending_confirmation", preview, "")
				ch := make(chan string, 1)
				s.setPending(msgID, ch)
				select {
				case decision := <-ch:
					if decision != "approve" {
						s.broadcastTool(convID, msgID, tc.Name, tc.Args, "rejected", nil, "")
						s.Repo.UpdateMessage(msgID, strPtr("rejected"), strPtr(`{"rejected":true}`))
						s.clearPending(msgID)
						continue
					}
				case <-time.After(ConfirmationTimeout):
					s.broadcastTool(convID, msgID, tc.Name, tc.Args, "rejected", nil, "timeout")
					s.Repo.UpdateMessage(msgID, strPtr("rejected"), strPtr(`{"error":"timeout"}`))
					s.clearPending(msgID)
					continue
				case <-ctx.Done():
					return ctx.Err()
				}
				s.clearPending(msgID)
				result, err := tool.Execute(ctx, tc.Args)
				if err != nil {
					s.broadcastTool(convID, msgID, tc.Name, tc.Args, "failed", nil, err.Error())
					s.Repo.UpdateMessage(msgID, strPtr("failed"), strPtr(fmt.Sprintf(`{"error":%q}`, err.Error())))
				} else {
					rjson, _ := json.Marshal(result)
					s.broadcastTool(convID, msgID, tc.Name, tc.Args, "succeeded", result, "")
					s.Repo.UpdateMessage(msgID, strPtr("succeeded"), strPtr(string(rjson)))
				}
			}
		}
	}
	s.broadcastDone(convID, "max_tools")
	return nil
}

func (s *agentService) Confirm(ctx context.Context, msgID, decision string) error {
	ch, ok := s.getPending(msgID)
	if !ok {
		return ErrToolNotFound
	}
	select {
	case ch <- decision:
		return nil
	default:
		return ErrToolNotFound
	}
}

func (s *agentService) setPending(msgID string, ch chan string) {
	s.pendingMu.Lock()
	s.pending[msgID] = ch
	s.pendingMu.Unlock()
}

func (s *agentService) getPending(msgID string) (chan string, bool) {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	ch, ok := s.pending[msgID]
	return ch, ok
}

func (s *agentService) clearPending(msgID string) {
	s.pendingMu.Lock()
	delete(s.pending, msgID)
	s.pendingMu.Unlock()
}

func (s *agentService) RunTool(ctx context.Context, name string, args json.RawMessage) (any, error) {
	t, err := s.Registry.Lookup(name)
	if err != nil {
		return nil, err
	}
	return t.Execute(ctx, args)
}

func (s *agentService) broadcast(convID, eventType string, payload map[string]any) {
	msg := map[string]any{"type": eventType}
	for k, v := range payload {
		msg[k] = v
	}
	s.Hub.Broadcast(msg)
}

func (s *agentService) broadcastDone(convID, reason string) {
	s.broadcast(convID, websocket.EventAgentDone, map[string]any{
		"conversation_id": convID, "finish_reason": reason,
	})
}

func (s *agentService) broadcastTool(convID, msgID, name string, args json.RawMessage, status string, result any, errMsg string) {
	payload := map[string]any{
		"conversation_id": convID, "tool_name": name,
		"args": json.RawMessage(args), "status": status,
	}
	if msgID != "" {
		payload["message_id"] = msgID
	}
	if result != nil {
		payload["result"] = result
	}
	if errMsg != "" {
		payload["error"] = errMsg
	}
	s.broadcast(convID, websocket.EventAgentTool, payload)
}

func (s *agentService) appendToolResult(convID string, tc ai.ToolCall, status, resultJSON string) {
	statusPtr := &status
	resultPtr := &resultJSON
	s.Repo.AppendMessage(convID, "tool_result", "", &tc.Name, nil, resultPtr, statusPtr)
}

func strPtr(s string) *string { return &s }
