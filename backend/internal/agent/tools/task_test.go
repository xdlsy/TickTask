package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"ticktask/internal/agent"
	"ticktask/internal/ai"
	"ticktask/internal/model"
	"ticktask/internal/repository"
	"ticktask/internal/service"
)

// --- mocks ---

// mockTaskSvc is an in-memory implementation of the tools-package TaskService
// interface. It records the last request it received so tests can assert on it.
type mockTaskSvc struct {
	tasks      []model.Task
	createReq  *service.CreateTaskRequest
	updateID   string
	updateReq  *service.UpdateTaskRequest
	deletedID  string
	createErr  error
	updateErr  error
	deleteErr  error
	getErr     error
	listErr    error
	createCall int
	updateCall int
	deleteCall int
	// add fields
	moveErr   error
	moveIDs   []string
	moveQuads []model.Quadrant
}

func (m *mockTaskSvc) CreateTask(req service.CreateTaskRequest) (*model.Task, error) {
	m.createCall++
	m.createReq = &req
	if m.createErr != nil {
		return nil, m.createErr
	}
	return &model.Task{ID: "new-1", Title: req.Title}, nil
}

func (m *mockTaskSvc) UpdateTask(id string, req service.UpdateTaskRequest) error {
	m.updateCall++
	m.updateID = id
	m.updateReq = &req
	return m.updateErr
}

func (m *mockTaskSvc) DeleteTask(id string) error {
	m.deleteCall++
	m.deletedID = id
	return m.deleteErr
}

func (m *mockTaskSvc) GetTask(id string) (*model.Task, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for i := range m.tasks {
		if m.tasks[i].ID == id {
			return &m.tasks[i], nil
		}
	}
	return nil, repository.ErrNotFound
}

func (m *mockTaskSvc) GetAllTasks() ([]model.Task, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.tasks, nil
}

func (m *mockTaskSvc) MoveTask(id string, targetQuadrant model.Quadrant) error {
	m.moveIDs = append(m.moveIDs, id)
	m.moveQuads = append(m.moveQuads, targetQuadrant)
	return m.moveErr
}

// mockLLM is a single-turn ChatCompletion stub. The tools package only uses
// ChatCompletion for classify_task (single-turn prompt). Tests pre-load the
// response they want returned.
type mockLLM struct {
	resp       string
	err        error
	lastPrompt string
	called     int
}

func (m *mockLLM) ChatCompletion(ctx context.Context, prompt string) (string, error) {
	m.called++
	m.lastPrompt = prompt
	if m.err != nil {
		return "", m.err
	}
	return m.resp, nil
}

func (m *mockLLM) ChatWithTools(ctx context.Context, msgs []ai.Message, tools []ai.ToolSpec) (ai.ToolResponse, error) {
	return ai.ToolResponse{}, ai.ErrFunctionCallNotSupported
}

// date helper returns a time.Time at midnight UTC on the given date string.
func mustDate(t *testing.T, s string) time.Time {
	t.Helper()
	tt, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("parse date %q: %v", s, err)
	}
	return tt
}

// --- ListTasksTool ---

func TestListTasks(t *testing.T) {
	svc := &mockTaskSvc{tasks: []model.Task{
		{ID: "1", Title: "alpha"},
		{ID: "2", Title: "beta"},
	}}
	tool := &ListTasksTool{Svc: svc}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	m, _ := json.Marshal(res)
	body := string(m)
	if !strings.Contains(body, `"alpha"`) || !strings.Contains(body, `"beta"`) {
		t.Fatalf("expected both titles in result, got %s", body)
	}
}

func TestListTasks_FilterByStatus(t *testing.T) {
	svc := &mockTaskSvc{tasks: []model.Task{
		{ID: "1", Title: "todo one", Status: model.StatusTodo},
		{ID: "2", Title: "done one", Status: model.StatusCompleted},
	}}
	tool := &ListTasksTool{Svc: svc}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"status":"completed"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	m, _ := json.Marshal(res)
	body := string(m)
	if strings.Contains(body, `"todo one"`) {
		t.Fatalf("todo task should be filtered out: %s", body)
	}
	if !strings.Contains(body, `"done one"`) {
		t.Fatalf("completed task should be present: %s", body)
	}
}

func TestListTasks_FilterByQuadrant(t *testing.T) {
	svc := &mockTaskSvc{tasks: []model.Task{
		{ID: "1", Title: "q1", Quadrant: model.Quadrant1},
		{ID: "2", Title: "q2", Quadrant: model.Quadrant2},
	}}
	tool := &ListTasksTool{Svc: svc}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"quadrant":2}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	m, _ := json.Marshal(res)
	body := string(m)
	if strings.Contains(body, `"q1"`) || !strings.Contains(body, `"q2"`) {
		t.Fatalf("quadrant filter failed: %s", body)
	}
}

func TestListTasks_FilterByDue(t *testing.T) {
	due := mustDate(t, "2026-08-08")
	other := mustDate(t, "2026-09-01")
	svc := &mockTaskSvc{tasks: []model.Task{
		{ID: "1", Title: "today-task", DueDate: &due},
		{ID: "2", Title: "later-task", DueDate: &other},
	}}
	tool := &ListTasksTool{Svc: svc}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"due":"2026-08-08"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	m, _ := json.Marshal(res)
	body := string(m)
	if strings.Contains(body, `"later-task"`) || !strings.Contains(body, `"today-task"`) {
		t.Fatalf("due filter failed: %s", body)
	}
}

func TestListTasks_SchemaValidationFails(t *testing.T) {
	svc := &mockTaskSvc{}
	tool := &ListTasksTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"status":123}`))
	if err == nil {
		t.Fatal("expected schema validation error")
	}
	if !strings.Contains(err.Error(), "schema") {
		t.Fatalf("expected schema error, got %v", err)
	}
}

func TestListTasks_Preview_EqualsExecute(t *testing.T) {
	svc := &mockTaskSvc{tasks: []model.Task{{ID: "1", Title: "x"}}}
	tool := &ListTasksTool{Svc: svc}
	preview, err := tool.Preview(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	m, _ := json.Marshal(preview)
	if !strings.Contains(string(m), `"x"`) {
		t.Fatalf("preview should mirror execute for read tool: %s", m)
	}
}

// --- CreateTaskTool ---

func TestCreateTask(t *testing.T) {
	svc := &mockTaskSvc{}
	tool := &CreateTaskTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"title":"my task"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if svc.createReq == nil {
		t.Fatal("CreateTask not called")
	}
	if svc.createReq.Title != "my task" {
		t.Fatalf("title = %q", svc.createReq.Title)
	}
}

func TestCreateTask_MissingTitle(t *testing.T) {
	svc := &mockTaskSvc{}
	tool := &CreateTaskTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected required-field error")
	}
	if !strings.Contains(err.Error(), "title") {
		t.Fatalf("error should mention title: %v", err)
	}
	if svc.createCall != 0 {
		t.Fatalf("CreateTask should not be called on validation failure")
	}
}

func TestCreateTask_PriorityMapping(t *testing.T) {
	svc := &mockTaskSvc{}
	tool := &CreateTaskTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"title":"x","priority":"important_urgent"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !svc.createReq.IsImportant || !svc.createReq.IsUrgent {
		t.Fatalf("priority important_urgent should set both flags: %+v", svc.createReq)
	}
	if svc.createReq.Quadrant != model.Quadrant1 {
		t.Fatalf("quadrant = %d, want 1", svc.createReq.Quadrant)
	}
}

func TestCreateTask_DueDate(t *testing.T) {
	svc := &mockTaskSvc{}
	tool := &CreateTaskTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"title":"x","due":"2026-08-08"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if svc.createReq.DueDate == nil {
		t.Fatal("DueDate should be set")
	}
	want := "2026-08-08"
	got := svc.createReq.DueDate.Format("2006-01-02")
	if got != want {
		t.Fatalf("DueDate = %s, want %s", got, want)
	}
}

func TestCreateTask_Preview(t *testing.T) {
	svc := &mockTaskSvc{}
	tool := &CreateTaskTool{Svc: svc}
	preview, err := tool.Preview(context.Background(), json.RawMessage(`{"title":"my task"}`))
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	m, _ := json.Marshal(preview)
	body := string(m)
	if !strings.Contains(body, `"create"`) || !strings.Contains(body, `"my task"`) {
		t.Fatalf("preview should describe create action: %s", body)
	}
	if svc.createCall != 0 {
		t.Fatalf("Preview must not call CreateTask, got %d calls", svc.createCall)
	}
}

// --- UpdateTaskTool ---

func TestUpdateTask(t *testing.T) {
	svc := &mockTaskSvc{}
	tool := &UpdateTaskTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"task_id":"abc","title":"new title"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if svc.updateID != "abc" {
		t.Fatalf("update id = %q", svc.updateID)
	}
	if svc.updateReq.Title == nil || *svc.updateReq.Title != "new title" {
		t.Fatalf("title not set correctly: %+v", svc.updateReq)
	}
	// other fields should be nil (partial update)
	if svc.updateReq.Description != nil {
		t.Fatalf("description should be nil for partial update")
	}
}

func TestUpdateTask_Status(t *testing.T) {
	svc := &mockTaskSvc{}
	tool := &UpdateTaskTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"task_id":"x","status":"completed"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if svc.updateReq.Status == nil || *svc.updateReq.Status != model.StatusCompleted {
		t.Fatalf("status not set: %+v", svc.updateReq)
	}
}

func TestUpdateTask_NotFound(t *testing.T) {
	svc := &mockTaskSvc{updateErr: repository.ErrNotFound}
	tool := &UpdateTaskTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"task_id":"missing","title":"x"}`))
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestUpdateTask_Preview(t *testing.T) {
	svc := &mockTaskSvc{}
	tool := &UpdateTaskTool{Svc: svc}
	preview, err := tool.Preview(context.Background(), json.RawMessage(`{"task_id":"abc","title":"new"}`))
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	m, _ := json.Marshal(preview)
	body := string(m)
	if !strings.Contains(body, `"update"`) || !strings.Contains(body, `"abc"`) {
		t.Fatalf("preview should describe update action: %s", body)
	}
	if svc.updateCall != 0 {
		t.Fatal("Preview must not call UpdateTask")
	}
}

// --- DeleteTaskTool ---

func TestDeleteTask(t *testing.T) {
	svc := &mockTaskSvc{}
	tool := &DeleteTaskTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"task_id":"abc"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if svc.deletedID != "abc" {
		t.Fatalf("deleted id = %q, want abc", svc.deletedID)
	}
}

func TestDeleteTask_Preview(t *testing.T) {
	svc := &mockTaskSvc{}
	tool := &DeleteTaskTool{Svc: svc}
	preview, err := tool.Preview(context.Background(), json.RawMessage(`{"task_id":"abc"}`))
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	m, _ := json.Marshal(preview)
	body := string(m)
	if !strings.Contains(body, `"delete"`) || !strings.Contains(body, `"abc"`) {
		t.Fatalf("preview should describe delete action: %s", body)
	}
	if svc.deleteCall != 0 {
		t.Fatal("Preview must not call DeleteTask")
	}
}

// --- ClassifyTaskTool ---

func TestClassifyTask_ByID(t *testing.T) {
	task := &model.Task{ID: "x1", Title: "Write report", Description: "Quarterly"}
	svc := &mockTaskSvc{tasks: []model.Task{*task}}
	llm := &mockLLM{resp: `{"important":true,"urgent":false,"reason":"critical"}`}
	tool := &ClassifyTaskTool{Svc: svc, LLM: llm}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"task_id":"x1"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	m, _ := json.Marshal(res)
	body := string(m)
	if !strings.Contains(body, `"important":true`) {
		t.Fatalf("expected important=true in result: %s", body)
	}
	if !strings.Contains(body, `"quadrant":2`) {
		t.Fatalf("important+not urgent should be quadrant 2: %s", body)
	}
}

func TestClassifyTask_ByText(t *testing.T) {
	svc := &mockTaskSvc{}
	llm := &mockLLM{resp: `{"important":false,"urgent":true,"reason":"deadline"}`}
	tool := &ClassifyTaskTool{Svc: svc, LLM: llm}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"title":"fix bug","description":"urgent issue"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	m, _ := json.Marshal(res)
	body := string(m)
	if !strings.Contains(body, `"quadrant":3`) {
		t.Fatalf("not important + urgent should be quadrant 3: %s", body)
	}
	if !strings.Contains(llm.lastPrompt, "fix bug") {
		t.Fatalf("prompt should contain title: %s", llm.lastPrompt)
	}
}

func TestClassifyTask_MissingArgs(t *testing.T) {
	svc := &mockTaskSvc{}
	llm := &mockLLM{}
	tool := &ClassifyTaskTool{Svc: svc, LLM: llm}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error for missing args")
	}
}

func TestClassifyTask_LLMError(t *testing.T) {
	svc := &mockTaskSvc{tasks: []model.Task{{ID: "x1", Title: "t"}}}
	llm := &mockLLM{err: errors.New("network down")}
	tool := &ClassifyTaskTool{Svc: svc, LLM: llm}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"task_id":"x1"}`))
	if err == nil {
		t.Fatal("expected error from LLM")
	}
	if !strings.Contains(err.Error(), "network down") {
		t.Fatalf("error should wrap LLM error: %v", err)
	}
}

func TestClassifyTask_MalformedJSON(t *testing.T) {
	svc := &mockTaskSvc{tasks: []model.Task{{ID: "x1", Title: "t"}}}
	llm := &mockLLM{resp: "not a json object"}
	tool := &ClassifyTaskTool{Svc: svc, LLM: llm}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"task_id":"x1"}`))
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestClassifyTask_TaskNotFound(t *testing.T) {
	svc := &mockTaskSvc{getErr: repository.ErrNotFound}
	llm := &mockLLM{}
	tool := &ClassifyTaskTool{Svc: svc, LLM: llm}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"task_id":"missing"}`))
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

// --- RegisterAll ---

func TestRegisterAll_RegistersFiveTools(t *testing.T) {
	reg := agent.NewToolRegistry()
	RegisterAll(reg, Deps{Tasks: &mockTaskSvc{}, LLM: &mockLLM{}})
	for _, name := range []string{"list_tasks", "create_task", "update_task", "delete_task", "classify_task", "move_task"} {
		if _, err := reg.Lookup(name); err != nil {
			t.Errorf("tool %q not registered: %v", name, err)
		}
	}
}

func TestMoveTask_Delegates(t *testing.T) {
	svc := &mockTaskSvc{}
	tool := &MoveTaskTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"task_id":"t-1","quadrant":2}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(svc.moveIDs) != 1 || svc.moveIDs[0] != "t-1" || svc.moveQuads[0] != model.Quadrant2 {
		t.Errorf("MoveTask args = %+v / %+v", svc.moveIDs, svc.moveQuads)
	}
	// out-of-range quadrant rejected before service
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"task_id":"t-1","quadrant":9}`)); err == nil {
		t.Error("expected error for quadrant 9")
	}
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"task_id":"t-1","quadrant":0}`)); err == nil {
		t.Error("expected error for quadrant 0")
	}
	if svc.moveIDs != nil && len(svc.moveIDs) != 1 {
		t.Errorf("service must not be called on invalid quadrant")
	}
	// missing required fields
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"quadrant":2}`)); err == nil {
		t.Error("expected error for missing task_id")
	}
}

func TestMoveTask_PreviewNoSideEffect(t *testing.T) {
	svc := &mockTaskSvc{}
	tool := &MoveTaskTool{Svc: svc}
	pv, err := tool.Preview(context.Background(), json.RawMessage(`{"task_id":"t-1","quadrant":2}`))
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	m, _ := json.Marshal(pv)
	if !strings.Contains(string(m), "move_task") || len(svc.moveIDs) != 0 {
		t.Fatalf("preview must echo plan and not call service: %s", m)
	}
}
