package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ticktask/internal/agent"
	"ticktask/internal/model"
	"ticktask/internal/repository"
)

// --- Tests ------------------------------------------------------------------

func TestAgentHandler_CreateConversation(t *testing.T) {
	svc := &mockAgentSvc{}
	repo := &mockAgentRepo{conv: &model.AgentConversation{ID: "c1"}}
	h := NewAgentHandler(svc, repo)
	r := setupTestRouter()
	h.Register(r.Group("/api/agent"))

	req, _ := http.NewRequest("POST", "/api/agent/conversations", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	var got model.AgentConversation
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, w.Body.String())
	}
	if got.ID != "c1" {
		t.Errorf("conversation id = %q, want c1", got.ID)
	}
}

func TestAgentHandler_ListConversations(t *testing.T) {
	svc := &mockAgentSvc{}
	repo := &mockAgentRepo{
		convs: []*model.AgentConversation{{ID: "a"}, {ID: "b"}},
		total: 2,
	}
	h := NewAgentHandler(svc, repo)
	r := setupTestRouter()
	h.Register(r.Group("/api/agent"))

	// Verify pagination params are forwarded: page=2 size=1 → repo receives (2,1).
	req, _ := http.NewRequest("GET", "/api/agent/conversations?page=2&size=1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if repo.lastPage != 2 || repo.lastSize != 1 {
		t.Errorf("pagination forwarded = (%d,%d), want (2,1)", repo.lastPage, repo.lastSize)
	}
	var body struct {
		Items []*model.AgentConversation `json:"items"`
		Total int                        `json:"total"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, w.Body.String())
	}
	if body.Total != 2 || len(body.Items) != 2 {
		t.Errorf("body = %+v, want total=2 items=2", body)
	}
}

func TestAgentHandler_GetMessages(t *testing.T) {
	svc := &mockAgentSvc{}
	repo := &mockAgentRepo{
		msgs: []*model.AgentMessage{{ID: "m1", ConversationID: "c1", Role: "user"}},
	}
	h := NewAgentHandler(svc, repo)
	r := setupTestRouter()
	h.Register(r.Group("/api/agent"))

	req, _ := http.NewRequest("GET", "/api/agent/conversations/c1/messages", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	if repo.lastListMessagesID != "c1" {
		t.Errorf("ListMessages called with %q, want c1", repo.lastListMessagesID)
	}
	var got []*model.AgentMessage
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, w.Body.String())
	}
	if len(got) != 1 || got[0].ID != "m1" {
		t.Errorf("got = %+v, want 1 message m1", got)
	}
}

func TestAgentHandler_DeleteConversation(t *testing.T) {
	svc := &mockAgentSvc{}
	repo := &mockAgentRepo{}
	h := NewAgentHandler(svc, repo)
	r := setupTestRouter()
	h.Register(r.Group("/api/agent"))

	req, _ := http.NewRequest("DELETE", "/api/agent/conversations/c1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body=%s", w.Code, w.Body.String())
	}
	if repo.lastDeleteID != "c1" {
		t.Errorf("DeleteConversation called with %q, want c1", repo.lastDeleteID)
	}
}

func TestAgentHandler_Chat(t *testing.T) {
	svc := &mockAgentSvc{}
	repo := &mockAgentRepo{}
	h := NewAgentHandler(svc, repo)
	r := setupTestRouter()
	h.Register(r.Group("/api/agent"))

	body, _ := json.Marshal(map[string]string{
		"conversation_id": "c1",
		"text":            "hello",
	})
	req, _ := http.NewRequest("POST", "/api/agent/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202; body=%s", w.Code, w.Body.String())
	}
	// The handler must spawn a background goroutine for SendMessage that is
	// decoupled from the request context (otherwise the agent would be cancelled
	// as soon as the 202 response completes). Wait briefly for the goroutine.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if atomic.LoadInt32(&svc.sendCalls) > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := atomic.LoadInt32(&svc.sendCalls); got != 1 {
		t.Fatalf("sendCalls = %d, want 1 (background goroutine not spawned)", got)
	}
	if svc.lastConvID != "c1" || svc.lastText != "hello" {
		t.Errorf("SendMessage(%q,%q), want (c1,hello)", svc.lastConvID, svc.lastText)
	}
}

func TestAgentHandler_RunTool(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		svc := &mockAgentSvc{toolResult: "ok"}
		repo := &mockAgentRepo{}
		h := NewAgentHandler(svc, repo)
		r := setupTestRouter()
		h.Register(r.Group("/api/agent"))

		body, _ := json.Marshal(map[string]any{"tool": "list_tasks", "args": map[string]string{"status": "todo"}})
		req, _ := http.NewRequest("POST", "/api/agent/run-tool", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
		}
		if svc.lastTool != "list_tasks" {
			t.Errorf("tool = %q, want list_tasks", svc.lastTool)
		}
	})
	t.Run("not_found", func(t *testing.T) {
		svc := &mockAgentSvc{toolErr: agent.ErrToolNotFound}
		repo := &mockAgentRepo{}
		h := NewAgentHandler(svc, repo)
		r := setupTestRouter()
		h.Register(r.Group("/api/agent"))

		body, _ := json.Marshal(map[string]any{"tool": "nope", "args": map[string]string{}})
		req, _ := http.NewRequest("POST", "/api/agent/run-tool", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
		}
	})
}

func TestAgentHandler_Confirm(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		svc := &mockAgentSvc{}
		repo := &mockAgentRepo{}
		h := NewAgentHandler(svc, repo)
		r := setupTestRouter()
		h.Register(r.Group("/api/agent"))

		body, _ := json.Marshal(map[string]string{"message_id": "m1", "decision": "approve"})
		req, _ := http.NewRequest("POST", "/api/agent/confirm", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
		}
		if svc.lastConfirmMsgID != "m1" || svc.lastConfirmDecision != "approve" {
			t.Errorf("Confirm(%q,%q), want (m1,approve)", svc.lastConfirmMsgID, svc.lastConfirmDecision)
		}
	})
	t.Run("not_found", func(t *testing.T) {
		svc := &mockAgentSvc{confirmErr: agent.ErrToolNotFound}
		repo := &mockAgentRepo{}
		h := NewAgentHandler(svc, repo)
		r := setupTestRouter()
		h.Register(r.Group("/api/agent"))

		body, _ := json.Marshal(map[string]string{"message_id": "m1", "decision": "approve"})
		req, _ := http.NewRequest("POST", "/api/agent/confirm", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404; body=%s", w.Code, w.Body.String())
		}
	})
}

func TestAgentHandler_Status(t *testing.T) {
	svc := &mockAgentSvc{}
	repo := &mockAgentRepo{}
	h := NewAgentHandler(svc, repo)
	r := setupTestRouter()
	h.Register(r.Group("/api/agent"))

	req, _ := http.NewRequest("GET", "/api/agent/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, w.Body.String())
	}
	if _, ok := body["configured"]; !ok {
		t.Fatalf("missing 'configured' field; body=%s", w.Body.String())
	}
	if _, ok := body["provider"]; !ok {
		t.Fatalf("missing 'provider' field; body=%s", w.Body.String())
	}
}

// --- Mocks ------------------------------------------------------------------

// mockAgentSvc implements agent.AgentService for handler tests.
type mockAgentSvc struct {
	sendCalls           int32 // accessed atomically from chat goroutine
	lastConvID          string
	lastText            string
	lastTool            string
	lastToolArgs        json.RawMessage
	toolResult          any
	toolErr             error
	lastConfirmMsgID    string
	lastConfirmDecision string
	confirmErr          error
	statusResult        agent.AgentStatus
	testResult          agent.TestResult
	mu                  sync.Mutex
}

func (m *mockAgentSvc) SendMessage(ctx context.Context, convID, text string) error {
	atomic.AddInt32(&m.sendCalls, 1)
	m.mu.Lock()
	m.lastConvID = convID
	m.lastText = text
	m.mu.Unlock()
	return nil
}

func (m *mockAgentSvc) Confirm(ctx context.Context, msgID, decision string) error {
	m.mu.Lock()
	m.lastConfirmMsgID = msgID
	m.lastConfirmDecision = decision
	m.mu.Unlock()
	return m.confirmErr
}

func (m *mockAgentSvc) RunTool(ctx context.Context, name string, args json.RawMessage) (any, error) {
	m.mu.Lock()
	m.lastTool = name
	m.lastToolArgs = args
	m.mu.Unlock()
	return m.toolResult, m.toolErr
}

func (m *mockAgentSvc) Status() agent.AgentStatus {
	if m.statusResult != (agent.AgentStatus{}) {
		return m.statusResult
	}
	return agent.AgentStatus{
		Configured:              true,
		SupportsFunctionCalling: true,
		Provider:                "openai",
	}
}

func (m *mockAgentSvc) TestConnection(ctx context.Context) agent.TestResult {
	if m.testResult != (agent.TestResult{}) {
		return m.testResult
	}
	return agent.TestResult{OK: true, Provider: "openai", Model: "gpt-4o-mini", LatencyMs: 1}
}

// mockAgentRepo implements repository.AgentRepository for handler tests.
type mockAgentRepo struct {
	conv               *model.AgentConversation
	createErr          error
	convs              []*model.AgentConversation
	total              int
	listErr            error
	lastPage           int
	lastSize           int
	msgs               []*model.AgentMessage
	listMsgsErr        error
	lastListMessagesID string
	deleteErr          error
	lastDeleteID       string
}

func (m *mockAgentRepo) CreateConversation() (*model.AgentConversation, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	if m.conv != nil {
		return m.conv, nil
	}
	return &model.AgentConversation{ID: "new"}, nil
}

func (m *mockAgentRepo) GetConversation(id string) (*model.AgentConversation, error) {
	return nil, repository.ErrNotFound
}

func (m *mockAgentRepo) ListConversations(page, size int) ([]*model.AgentConversation, int, error) {
	m.lastPage = page
	m.lastSize = size
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	return m.convs, m.total, nil
}

func (m *mockAgentRepo) DeleteConversation(id string) error {
	m.lastDeleteID = id
	return m.deleteErr
}

func (m *mockAgentRepo) AppendMessage(convID, role, content string, toolName, toolArgs, toolResult, toolStatus *string) (string, error) {
	return "msg", nil
}

func (m *mockAgentRepo) LoadRecentMessages(convID string, limit int) ([]*model.AgentMessage, error) {
	return nil, nil
}

func (m *mockAgentRepo) ListMessages(convID string) ([]*model.AgentMessage, error) {
	m.lastListMessagesID = convID
	if m.listMsgsErr != nil {
		return nil, m.listMsgsErr
	}
	return m.msgs, nil
}

func (m *mockAgentRepo) UpdateMessage(id string, status, result *string) error {
	return nil
}

func (m *mockAgentRepo) GetMessage(id string) (*model.AgentMessage, error) {
	return nil, repository.ErrNotFound
}
