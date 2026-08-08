package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"ticktask/internal/ai"
	"ticktask/internal/model"
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
	Repo        repository.AgentRepository
	LLMFactory  func() ai.LLMClient
	SettingsRepo repository.SettingRepository
	Registry    ToolRegistry
	Hub         HubBroadcaster
	System      string
	Tracer      TraceRecorder
}

type AgentService interface {
	SendMessage(ctx context.Context, convID, text string) error
	Confirm(ctx context.Context, msgID string, decision string) error
	RunTool(ctx context.Context, name string, args json.RawMessage) (any, error)
	Status() AgentStatus
	TestConnection(ctx context.Context, settings *model.AISettings) TestResult
}

// AgentStatus describes runtime capability flags surfaced to the frontend via
// GET /api/agent/status. Task 22 replaces the stub implementation with one that
// reads the live settings (provider / API-key presence / function-calling
// capability).
type AgentStatus struct {
	Configured              bool   `json:"configured"`
	SupportsFunctionCalling bool   `json:"supports_function_calling"`
	Provider                string `json:"provider"`
}

// TestResult is the payload returned by TestConnection / POST /api/agent/test.
// Unlike Status (which only inspects settings), TestConnection actually sends
// a minimal LLM round-trip and reports whether the provider accepted the call.
type TestResult struct {
	OK         bool   `json:"ok"`
	Provider   string `json:"provider"`
	Model      string `json:"model,omitempty"`
	LatencyMs  int64  `json:"latency_ms"`
	StatusCode int    `json:"status_code,omitempty"`
	Error      string `json:"error,omitempty"`
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
	if d.Tracer == nil {
		d.Tracer = noopTracer{}
	}
	return &agentService{AgentDeps: d, pending: make(map[string]chan string)}
}

// systemPromptWithDate appends the current local date/time to the configured
// system prompt so the model can resolve relative references like "today" or
// "this week" when routing to date-scoped tools (e.g. list_schedule, which
// requires explicit YYYY-MM-DD from/to). Computed per turn so it stays current
// across long-lived processes.
func (s *agentService) systemPromptWithDate() string {
	now := time.Now()
	return s.System + fmt.Sprintf(
		"\n\nCurrent date: %s (%s). Current time: %s. "+
			"Use this date when the user refers to \"today\", \"this week\", or any relative date.",
		now.Format("2006-01-02"), now.Weekday(), now.Format("15:04"),
	)
}

func (s *agentService) SendMessage(ctx context.Context, convID, text string) error {
	if _, err := s.Repo.GetConversation(convID); err != nil {
		return err
	}
	if _, err := s.Repo.AppendMessage(convID, "user", text, nil, nil, nil, nil, nil, nil); err != nil {
		return err
	}
	return s.runTurn(ctx, convID, text, 0)
}

func (s *agentService) runTurn(ctx context.Context, convID, userText string, toolCount int) error {
	trace := TurnTrace{ConversationID: convID, UserText: userText}
	defer func() { s.Tracer.RecordTurn(convID, trace) }()

	client := s.LLMFactory()
	if client == nil {
		s.broadcast(convID, websocket.EventAgentDone, map[string]any{
			"conversation_id": convID, "finish_reason": "error",
			"error": "AI 未配置 — 请先在设置中填写 API Key 或切换到 CLI provider",
		})
		return fmt.Errorf("AI client not configured")
	}
	for toolCount < MaxToolCallsPerTurn {
		history, err := s.Repo.LoadRecentMessages(convID, MaxContextMessages)
		if err != nil {
			return err
		}
		msgs := buildLLMMessages(s.systemPromptWithDate(), history)
		resp, err := client.ChatWithTools(ctx, msgs, s.Registry.ToOpenAITools())
		if err != nil {
			s.broadcastDone(convID, "error")
			return err
		}
		if resp.Content != "" || len(resp.ToolCalls) > 0 {
			var toolCallsPtr *string
			if len(resp.ToolCalls) > 0 {
				if tcJSON, err := json.Marshal(resp.ToolCalls); err == nil {
					toolCallsPtr = strPtr(string(tcJSON))
				}
			}
			msgID, err := s.Repo.AppendMessage(convID, "assistant", resp.Content, nil, nil, nil, nil, toolCallsPtr, nil)
			if err != nil {
				s.broadcastDone(convID, "error")
				return err
			}
			if resp.Content != "" {
				s.broadcast(convID, websocket.EventAgentMessage, map[string]any{
					"conversation_id": convID, "message_id": msgID, "delta_text": resp.Content,
				})
				trace.AssistantText += resp.Content
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
				s.broadcastTool(convID, "", tc.Name, tc.Args, "failed", nil, nil, fmt.Sprintf("tool not found: %s", tc.Name))
				s.appendToolResult(convID, tc, "failed", `{"error":"not found"}`)
				trace.Steps = append(trace.Steps, traceStep(tc, PermRead, "failed", nil, fmt.Sprintf("tool not found: %s", tc.Name)))
				continue
			}
			// Schema-validate args BEFORE the permission branch: a malformed
			// tool call should never reach Execute (PermRead) or the user
			// confirmation flow (PermWrite/Dangerous). The error is re-fed to
			// the LLM via the appended tool_result so the next turn can correct
			// itself. Note: ValidateArgs prefixes its errors with "schema:".
			if err := ValidateArgs(tool.Schema().Function.Parameters, tc.Args); err != nil {
				s.broadcastTool(convID, "", tc.Name, tc.Args, "failed", nil, nil, err.Error())
				s.appendToolResult(convID, tc, "failed", fmt.Sprintf(`{"schema_error":%q}`, err.Error()))
				trace.Steps = append(trace.Steps, traceStep(tc, PermRead, "failed", nil, err.Error()))
				continue
			}
			perm := tool.Schema().Permission
			if perm == PermRead {
				s.broadcastTool(convID, "", tc.Name, tc.Args, "started", nil, nil, "")
				result, err := tool.Execute(ctx, tc.Args)
				if err != nil {
					s.broadcastTool(convID, "", tc.Name, tc.Args, "failed", nil, nil, err.Error())
					s.appendToolResult(convID, tc, "failed", fmt.Sprintf(`{"error":%q}`, err.Error()))
					trace.Steps = append(trace.Steps, traceStep(tc, perm, "failed", nil, err.Error()))
				} else {
					rjson, _ := json.Marshal(result)
					s.broadcastTool(convID, "", tc.Name, tc.Args, "succeeded", result, nil, "")
					s.appendToolResult(convID, tc, "succeeded", string(rjson))
					trace.Steps = append(trace.Steps, traceStep(tc, perm, "succeeded", result, ""))
				}
			} else {
				// PermWrite / PermDangerous — require user confirmation
				preview, _ := tool.Preview(ctx, tc.Args)
				status := "pending_confirmation"
				msgID, _ := s.Repo.AppendMessage(convID, "tool_call", "", &tc.Name, strPtr(string(tc.Args)), nil, &status, nil, nil)
				s.broadcastTool(convID, msgID, tc.Name, tc.Args, "pending_confirmation", nil, preview, "")
				ch := make(chan string, 1)
				s.setPending(msgID, ch)
				select {
				case decision := <-ch:
					if decision != "approve" {
						s.broadcastTool(convID, msgID, tc.Name, tc.Args, "rejected", nil, nil, "")
						s.Repo.UpdateMessage(msgID, strPtr("rejected"), strPtr(`{"rejected":true}`))
						s.clearPending(msgID)
						trace.Steps = append(trace.Steps, traceStep(tc, perm, "rejected", nil, ""))
						continue
					}
				case <-time.After(ConfirmationTimeout):
					s.broadcastTool(convID, msgID, tc.Name, tc.Args, "rejected", nil, nil, "timeout")
					s.Repo.UpdateMessage(msgID, strPtr("rejected"), strPtr(`{"error":"timeout"}`))
					s.clearPending(msgID)
					trace.Steps = append(trace.Steps, traceStep(tc, perm, "rejected", nil, "timeout"))
					continue
				case <-ctx.Done():
					return ctx.Err()
				}
				s.clearPending(msgID)
				result, err := tool.Execute(ctx, tc.Args)
				if err != nil {
					s.broadcastTool(convID, msgID, tc.Name, tc.Args, "failed", nil, nil, err.Error())
					s.Repo.UpdateMessage(msgID, strPtr("failed"), strPtr(fmt.Sprintf(`{"error":%q}`, err.Error())))
					trace.Steps = append(trace.Steps, traceStep(tc, perm, "failed", nil, err.Error()))
				} else {
					rjson, _ := json.Marshal(result)
					s.broadcastTool(convID, msgID, tc.Name, tc.Args, "succeeded", result, nil, "")
					s.Repo.UpdateMessage(msgID, strPtr("succeeded"), strPtr(string(rjson)))
					trace.Steps = append(trace.Steps, traceStep(tc, perm, "succeeded", result, ""))
				}
			}
		}
	}
	s.broadcastDone(convID, "max_tools")
	return nil
}

// traceStep builds a TraceStep from the in-scope tool-call decision and its
// terminal outcome, keeping each append site in runTurn a one-liner.
func traceStep(tc ai.ToolCall, perm ToolPermission, status string, result any, errMsg string) TraceStep {
	return TraceStep{ToolCall: tc, Permission: perm, Status: status, Result: result, Error: errMsg}
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

// Status reports live agent capability flags derived from the current AI
// settings. It is consumed by GET /api/agent/status and powers the frontend's
// "agent available" indicator.
//
//   - Configured: true when an API key (or CLI provider, which needs none) is
//     present in settings.
//   - SupportsFunctionCalling: false for the CLI-subprocess providers, which
//     shell out to `claude`/`gemini` etc. and have no structured tool-call
//     surface. OpenAI, Anthropic, and "custom" (OpenAI-compatible) all
//     support tools.
//   - Provider: echoed back so the frontend can show the active backend.
//
// SettingsRepo is optional: if absent (e.g. in unit tests), Status returns a
// conservative all-false zero value so the handler still compiles and serves.
func (s *agentService) Status() AgentStatus {
	if s.SettingsRepo == nil {
		return AgentStatus{}
	}
	settings, _ := s.SettingsRepo.GetAISettings()
	if settings == nil {
		return AgentStatus{}
	}
	return AgentStatus{
		Configured:              settings.APIKey != "" || isCLIProvider(settings.Provider),
		SupportsFunctionCalling: !isCLIProvider(settings.Provider),
		Provider:                settings.Provider,
	}
}

// TestConnection sends a minimal ChatCompletion to verify the configured LLM
// provider actually accepts calls. settings controls what to test:
//   - settings == nil: use the saved DB settings via LLMFactory (test the
//     currently-configured key)
//   - settings != nil: construct a one-shot client from these settings via
//     ai.NewClientFromSettings (test the form values the user just typed,
//     before saving)
//
// In neither branch are settings written back to the DB. The non-nil branch
// intentionally bypasses LLMFactory so the test reflects exactly what was
// passed in, not what is currently persisted.
func (s *agentService) TestConnection(ctx context.Context, settings *model.AISettings) TestResult {
	var client ai.LLMClient
	var provider, model string

	if settings != nil {
		client = ai.NewClientFromSettings(settings)
		provider = settings.Provider
		model = settings.Model
	} else {
		current := s.readSettings()
		client = s.LLMFactory()
		provider = current.Provider
		model = current.Model
	}

	result := TestResult{Provider: provider, Model: model}
	if client == nil {
		result.Error = "AI 未配置 — 请填写 API Key 或切换到 CLI provider"
		return result
	}

	t0 := time.Now()
	callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if _, err := client.ChatCompletion(callCtx, "hi"); err != nil {
		result.Error = err.Error()
		result.LatencyMs = time.Since(t0).Milliseconds()
		return result
	}
	result.LatencyMs = time.Since(t0).Milliseconds()
	result.OK = true
	return result
}

func (s *agentService) readSettings() *model.AISettings {
	if s.SettingsRepo == nil {
		return model.DefaultAISettings()
	}
	settings, _ := s.SettingsRepo.GetAISettings()
	if settings == nil {
		return model.DefaultAISettings()
	}
	return settings
}

// isCLIProvider returns true for providers that shell out to a local CLI
// subprocess (e.g. `claude`/`gemini`) and therefore cannot participate in the
// structured tool-calling flow. main.go maps the `"claude"` provider string
// to CLIClient; the design spec uses the literal `"cli"`. Both are treated as
// CLI-mode here for forwards compatibility.
func isCLIProvider(provider string) bool {
	return provider == "cli" || provider == "claude"
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

func (s *agentService) broadcastTool(convID, msgID, name string, args json.RawMessage, status string, result any, preview any, errMsg string) {
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
	if preview != nil {
		payload["preview"] = preview
	}
	if errMsg != "" {
		payload["error"] = errMsg
	}
	s.broadcast(convID, websocket.EventAgentTool, payload)
}

func (s *agentService) appendToolResult(convID string, tc ai.ToolCall, status, resultJSON string) {
	statusPtr := &status
	resultPtr := &resultJSON
	s.Repo.AppendMessage(convID, "tool_result", "", &tc.Name, nil, resultPtr, statusPtr, nil, strPtr(tc.ID))
}

func strPtr(s string) *string { return &s }
