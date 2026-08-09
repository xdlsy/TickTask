package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"ticktask/internal/model"
	"ticktask/internal/service"
)

// --- mocks ---

// mockWorkLogStructureSvc is an in-memory implementation of the tools-package
// WorkLogStructureSvc interface. Only StructureBrainDump is exercised by the
// structure_worklog tool.
type mockWorkLogStructureSvc struct {
	structureIn   service.BrainDumpInput
	structureOut  *service.StructuredWorkLog
	structureErr  error
	structureCall int
}

func (m *mockWorkLogStructureSvc) StructureBrainDump(input service.BrainDumpInput) (*service.StructuredWorkLog, error) {
	m.structureCall++
	m.structureIn = input
	if m.structureErr != nil {
		return nil, m.structureErr
	}
	return m.structureOut, nil
}

// mockWorkLogSaveSvc is an in-memory implementation of the tools-package
// WorkLogSaveSvc interface. Only SaveWorkLog is exercised by the save_worklog
// tool.
type mockWorkLogSaveSvc struct {
	saveIn   service.SaveWorkLogInput
	saveOut  *model.WorkLog
	saveErr  error
	saveCall int
}

func (m *mockWorkLogSaveSvc) SaveWorkLog(input service.SaveWorkLogInput) (*model.WorkLog, error) {
	m.saveCall++
	m.saveIn = input
	if m.saveErr != nil {
		return nil, m.saveErr
	}
	return m.saveOut, nil
}

type mockWorkLogReadSvc struct {
	getLog    *model.WorkLog
	getErr    error
	getDates  []string
	listLogs  []*model.WorkLog
	listErr   error
	listCalls int
	listFrom  string
	listTo    string
}

func (m *mockWorkLogReadSvc) GetWorkLog(date string) (*model.WorkLog, error) {
	m.getDates = append(m.getDates, date)
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.getLog, nil
}
func (m *mockWorkLogReadSvc) ListWorkLogs(from, to string) ([]*model.WorkLog, error) {
	m.listCalls++
	m.listFrom = from
	m.listTo = to
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listLogs, nil
}

type mockWorkLogReportSvc struct {
	genIn   *service.GenerateReportInput
	genOut  *model.WorkReport
	genErr  error
	getOut  *model.WorkReport
	getErr  error
	listOut []*model.WorkReport
	listErr error
}

func (m *mockWorkLogReportSvc) GenerateReport(input service.GenerateReportInput) (*model.WorkReport, error) {
	m.genIn = &input
	if m.genErr != nil {
		return nil, m.genErr
	}
	return m.genOut, nil
}
func (m *mockWorkLogReportSvc) GetReport(t model.WorkReportType, periodKey string) (*model.WorkReport, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.getOut, nil
}
func (m *mockWorkLogReportSvc) ListReports(t model.WorkReportType) ([]*model.WorkReport, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listOut, nil
}

// --- StructureWorklogTool ---

func TestStructureWorklog_HappyPath(t *testing.T) {
	svc := &mockWorkLogStructureSvc{
		structureOut: &service.StructuredWorkLog{
			Items: []service.StructuredItem{
				{Content: "wrote code", ProblemSolved: "bug X", Result: "shipped", Impact: "faster"},
			},
			Summary: "1 item",
		},
	}
	tool := &StructureWorklogTool{Svc: svc}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"brain_dump":"did stuff today"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if svc.structureCall != 1 {
		t.Fatalf("StructureBrainDump calls = %d, want 1", svc.structureCall)
	}
	if svc.structureIn.BrainDump != "did stuff today" {
		t.Fatalf("brain_dump echoed wrong: %q", svc.structureIn.BrainDump)
	}
	m, _ := json.Marshal(res)
	body := string(m)
	for _, want := range []string{`"content":"wrote code"`, `"summary":"1 item"`, `"items"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in result: %s", want, body)
		}
	}
}

func TestStructureWorklog_MissingBrainDump(t *testing.T) {
	svc := &mockWorkLogStructureSvc{}
	tool := &StructureWorklogTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected schema validation error for missing brain_dump")
	}
	if !strings.Contains(err.Error(), "schema") {
		t.Fatalf("expected schema error, got %v", err)
	}
	if svc.structureCall != 0 {
		t.Fatalf("StructureBrainDump should not be called on validation failure")
	}
}

func TestStructureWorklog_ServiceError(t *testing.T) {
	svc := &mockWorkLogStructureSvc{structureErr: errors.New("AI not configured")}
	tool := &StructureWorklogTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"brain_dump":"x"}`))
	if err == nil || !strings.Contains(err.Error(), "AI not configured") {
		t.Fatalf("expected wrapped service error, got %v", err)
	}
}

func TestStructureWorklog_SchemaValidationFails(t *testing.T) {
	svc := &mockWorkLogStructureSvc{}
	tool := &StructureWorklogTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"brain_dump":42}`))
	if err == nil {
		t.Fatal("expected schema validation error for non-string brain_dump")
	}
	if !strings.Contains(err.Error(), "schema") {
		t.Fatalf("expected schema error, got %v", err)
	}
}

func TestStructureWorklog_Preview_EqualsExecute(t *testing.T) {
	// PermRead tool — preview mirrors execute (same auto-execute semantics).
	svc := &mockWorkLogStructureSvc{
		structureOut: &service.StructuredWorkLog{
			Items:   []service.StructuredItem{{Content: "x"}},
			Summary: "s",
		},
	}
	tool := &StructureWorklogTool{Svc: svc}
	preview, err := tool.Preview(context.Background(), json.RawMessage(`{"brain_dump":"x"}`))
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if svc.structureCall != 1 {
		t.Fatalf("preview should call service for read tool, got %d", svc.structureCall)
	}
	m, _ := json.Marshal(preview)
	if !strings.Contains(string(m), `"content":"x"`) {
		t.Fatalf("preview should mirror execute for read tool: %s", m)
	}
}

// --- SaveWorklogTool ---

func TestSaveWorklog_HappyPath(t *testing.T) {
	svc := &mockWorkLogSaveSvc{
		saveOut: &model.WorkLog{
			ID:   "log-1",
			Date: "2026-08-08",
			Items: []model.WorkItem{
				{ID: "item-1", WorkLogID: "log-1", Seq: 1, Content: "c1", Source: "ai"},
			},
		},
	}
	tool := &SaveWorklogTool{Svc: svc}
	args := json.RawMessage(`{
		"date":"2026-08-08",
		"summary":"day summary",
		"items":[
			{"seq":1,"content":"c1","problem_solved":"p1","result":"r1","impact":"i1"}
		]
	}`)
	res, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if svc.saveCall != 1 {
		t.Fatalf("SaveWorkLog calls = %d, want 1", svc.saveCall)
	}
	if svc.saveIn.Date != "2026-08-08" {
		t.Fatalf("input date = %q", svc.saveIn.Date)
	}
	if svc.saveIn.Summary != "day summary" {
		t.Fatalf("input summary = %q", svc.saveIn.Summary)
	}
	if len(svc.saveIn.Items) != 1 {
		t.Fatalf("input items len = %d, want 1", len(svc.saveIn.Items))
	}
	got := svc.saveIn.Items[0]
	if got.Seq != 1 || got.Content != "c1" || got.ProblemSolved != "p1" || got.Result != "r1" || got.Impact != "i1" {
		t.Fatalf("mapped item wrong: %+v", got)
	}
	m, _ := json.Marshal(res)
	body := string(m)
	if !strings.Contains(body, `"log-1"`) || !strings.Contains(body, `"item-1"`) {
		t.Fatalf("expected saved log in result: %s", body)
	}
}

func TestSaveWorklog_MissingItems(t *testing.T) {
	svc := &mockWorkLogSaveSvc{}
	tool := &SaveWorklogTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"date":"2026-08-08"}`))
	if err == nil {
		t.Fatal("expected schema validation error for missing items")
	}
	if !strings.Contains(err.Error(), "schema") {
		t.Fatalf("expected schema error, got %v", err)
	}
	if svc.saveCall != 0 {
		t.Fatalf("SaveWorkLog should not be called on validation failure")
	}
}

func TestSaveWorklog_MissingDate(t *testing.T) {
	svc := &mockWorkLogSaveSvc{}
	tool := &SaveWorklogTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"items":[]}`))
	if err == nil {
		t.Fatal("expected schema validation error for missing date")
	}
	if !strings.Contains(err.Error(), "schema") {
		t.Fatalf("expected schema error, got %v", err)
	}
}

func TestSaveWorklog_ServiceError(t *testing.T) {
	svc := &mockWorkLogSaveSvc{saveErr: service.ErrWorkLogAlreadyExists}
	tool := &SaveWorklogTool{Svc: svc}
	args := json.RawMessage(`{"date":"2026-08-08","items":[{"seq":1,"content":"c1"}]}`)
	_, err := tool.Execute(context.Background(), args)
	if err == nil || !errors.Is(err, service.ErrWorkLogAlreadyExists) {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}

func TestSaveWorklog_Preview(t *testing.T) {
	// PermWrite tool — preview returns a plan WITHOUT invoking the service.
	svc := &mockWorkLogSaveSvc{}
	tool := &SaveWorklogTool{Svc: svc}
	args := json.RawMessage(`{
		"date":"2026-08-08",
		"items":[
			{"seq":1,"content":"c1"},
			{"seq":2,"content":"c2"}
		]
	}`)
	preview, err := tool.Preview(context.Background(), args)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if svc.saveCall != 0 {
		t.Fatalf("Preview must not call SaveWorkLog, got %d", svc.saveCall)
	}
	m, _ := json.Marshal(preview)
	body := string(m)
	for _, want := range []string{`"save_worklog"`, `"2026-08-08"`, `"items_count":2`} {
		if !strings.Contains(body, want) {
			t.Fatalf("preview missing %q: %s", want, body)
		}
	}
}

func TestSaveWorklog_Preview_MalformedArgs(t *testing.T) {
	// Preview must tolerate partial / malformed args without side effects.
	svc := &mockWorkLogSaveSvc{}
	tool := &SaveWorklogTool{Svc: svc}
	preview, err := tool.Preview(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("preview should not error on missing args: %v", err)
	}
	if svc.saveCall != 0 {
		t.Fatalf("Preview must not call SaveWorkLog")
	}
	m, _ := json.Marshal(preview)
	if !strings.Contains(string(m), `"save_worklog"`) {
		t.Fatalf("preview should expose action even on partial args: %s", m)
	}
}

func TestGetWorkLog_DelegatesByDate(t *testing.T) {
	svc := &mockWorkLogReadSvc{getLog: &model.WorkLog{Date: "2026-08-08"}}
	tool := &GetWorklogTool{Svc: svc}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"date":"2026-08-08"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(svc.getDates) != 1 || svc.getDates[0] != "2026-08-08" {
		t.Errorf("dates = %+v", svc.getDates)
	}
	m, _ := json.Marshal(res)
	if !strings.Contains(string(m), "2026-08-08") {
		t.Errorf("result should include the log: %s", m)
	}
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{}`)); err == nil {
		t.Error("expected error for missing date")
	}
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"date":"not-a-date"}`)); err == nil {
		t.Error("expected error for invalid date")
	}
}

func TestListWorklogs_Range(t *testing.T) {
	svc := &mockWorkLogReadSvc{listLogs: []*model.WorkLog{{Date: "2026-08-08"}}}
	tool := &ListWorklogsTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"from":"2026-08-01","to":"2026-08-31"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if svc.listCalls != 1 {
		t.Errorf("ListWorkLogs calls = %d", svc.listCalls)
	}
	if svc.listFrom != "2026-08-01" || svc.listTo != "2026-08-31" {
		t.Errorf("from/to not forwarded: from=%s to=%s", svc.listFrom, svc.listTo)
	}
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"from":"2026-08-01"}`)); err == nil {
		t.Error("expected error for missing to")
	}
}

func TestListWorklogs_InvalidDate(t *testing.T) {
	svc := &mockWorkLogReadSvc{}
	tool := &ListWorklogsTool{Svc: svc}
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"from":"garbage","to":"2026-08-31"}`)); err == nil {
		t.Error("expected error for invalid from")
	}
	if svc.listCalls != 0 {
		t.Errorf("service must not be called on invalid date")
	}
}

func TestListWorklogs_ServiceError(t *testing.T) {
	svc := &mockWorkLogReadSvc{listErr: errors.New("db locked")}
	tool := &ListWorklogsTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"from":"2026-08-01","to":"2026-08-31"}`))
	if err == nil || !strings.Contains(err.Error(), "db locked") {
		t.Fatalf("expected wrapped service error, got %v", err)
	}
}

func TestGenerateWorkReport_Delegates(t *testing.T) {
	svc := &mockWorkLogReportSvc{genOut: &model.WorkReport{Type: model.ReportWeekly}}
	tool := &GenerateWorkReportTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"type":"weekly","period_key":"2026-W32"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if svc.genIn == nil || svc.genIn.Type != model.ReportWeekly || svc.genIn.PeriodKey != "2026-W32" {
		t.Errorf("GenerateReport input = %+v", svc.genIn)
	}
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"type":"bogus","period_key":"x"}`)); err == nil {
		t.Error("expected error for bad report type")
	}
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"type":"weekly"}`)); err == nil {
		t.Error("expected error for missing period_key")
	}
}

func TestGetWorkReport_ListWhenNoPeriod(t *testing.T) {
	svc := &mockWorkLogReportSvc{listOut: []*model.WorkReport{{Type: model.ReportWeekly}}}
	tool := &GetWorkReportTool{Svc: svc}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"type":"weekly"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	m, _ := json.Marshal(res)
	if !strings.Contains(string(m), `"list":`) {
		t.Errorf("omitting period_key should list reports: %s", m)
	}
}

func TestGetWorkReport_GetByPeriod(t *testing.T) {
	svc := &mockWorkLogReportSvc{getOut: &model.WorkReport{Type: model.ReportMonthly}}
	tool := &GetWorkReportTool{Svc: svc}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"type":"monthly","period_key":"2026-08"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if svc.getOut == nil {
		t.Error("expected GetReport path")
	}
	m, _ := json.Marshal(res)
	if !strings.Contains(string(m), "monthly") {
		t.Errorf("result should include the report: %s", m)
	}
}
