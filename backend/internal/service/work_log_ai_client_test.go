package service

import (
	"encoding/json"
	"strings"
	"testing"

	"ticktask/internal/model"
)

func TestStripCodeFence_Plain(t *testing.T) {
	in := `{"a":1}`
	got := stripCodeFence(in)
	if got != in {
		t.Errorf("got=%q want=%q", got, in)
	}
}

func TestStripCodeFence_JSONBlock(t *testing.T) {
	in := "```json\n{\"a\":1}\n```"
	got := stripCodeFence(in)
	want := `{"a":1}`
	if got != want {
		t.Errorf("got=%q want=%q", got, want)
	}
}

func TestStripCodeFence_BareCodeBlock(t *testing.T) {
	in := "```\n{\"a\":1}\n```"
	got := stripCodeFence(in)
	want := `{"a":1}`
	if got != want {
		t.Errorf("got=%q want=%q", got, want)
	}
}

func TestStripCodeFence_TrimsWhitespace(t *testing.T) {
	in := "  \n{\"a\":1}\n  \n"
	got := stripCodeFence(in)
	want := `{"a":1}`
	if got != want {
		t.Errorf("got=%q want=%q", got, want)
	}
}

func TestParseReportSummary_Valid(t *testing.T) {
	raw := `{"core_work":"cw","main_progress":"mp","open_issues":"oi","next_focus":"nf"}`
	s, err := parseReportSummary(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if s.CoreWork != "cw" {
		t.Errorf("CoreWork = %q, want 'cw'", s.CoreWork)
	}
	if s.MainProgress != "mp" {
		t.Errorf("MainProgress = %q, want 'mp'", s.MainProgress)
	}
	if s.OpenIssues != "oi" {
		t.Errorf("OpenIssues = %q, want 'oi'", s.OpenIssues)
	}
	if s.NextFocus != "nf" {
		t.Errorf("NextFocus = %q, want 'nf'", s.NextFocus)
	}
}

func TestParseReportSummary_StripsCodeFence(t *testing.T) {
	raw := "```json\n{\"core_work\":\"cw\",\"main_progress\":\"mp\",\"open_issues\":\"oi\",\"next_focus\":\"nf\"}\n```"
	s, err := parseReportSummary(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if s.CoreWork != "cw" {
		t.Errorf("CoreWork = %q, want 'cw'", s.CoreWork)
	}
}

func TestParseReportSummary_BadJSON(t *testing.T) {
	raw := `not json`
	_, err := parseReportSummary(raw)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "parse report JSON") {
		t.Errorf("err = %v, want contains 'parse report JSON'", err)
	}
	// raw should be included for debuggability
	if !strings.Contains(err.Error(), "raw=") {
		t.Errorf("err = %v, want contains 'raw=' for debugging", err)
	}
}

func TestItemsToJSON_Structure(t *testing.T) {
	items := []model.WorkItem{
		{Title: "T1", Content: "C1", ProblemSolved: "P1", Result: "R1", Impact: "I1"},
	}
	s := itemsToJSON(items)
	var got []map[string]string
	if err := json.Unmarshal([]byte(s), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	want := map[string]string{
		"title":          "T1",
		"content":        "C1",
		"problem_solved": "P1",
		"result":         "R1",
		"impact":         "I1",
	}
	for k, v := range want {
		if got[0][k] != v {
			t.Errorf("got[0][%q] = %q, want %q", k, got[0][k], v)
		}
	}
}

func TestItemsToJSON_Empty(t *testing.T) {
	s := itemsToJSON(nil)
	if s != "[]" {
		t.Errorf("got=%q, want '[]'", s)
	}
}

func TestReportsToJSON_Structure(t *testing.T) {
	reports := []*model.WorkReport{
		{Type: model.ReportWeekly, PeriodKey: "2026-W31", StartDate: "2026-07-27", EndDate: "2026-08-02", SummaryJSON: `{"core_work":"x"}`},
	}
	s := reportsToJSON(reports)
	var got []map[string]string
	if err := json.Unmarshal([]byte(s), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0]["period_key"] != "2026-W31" {
		t.Errorf("period_key = %q, want '2026-W31'", got[0]["period_key"])
	}
	if got[0]["summary_json"] != `{"core_work":"x"}` {
		t.Errorf("summary_json = %q", got[0]["summary_json"])
	}
}

func TestFormatContextForPrompt_HasSummary(t *testing.T) {
	ctx := TodayContext{
		Date:           "2026-08-02",
		CompletedTasks: []TaskBrief{{ID: "t1", Title: "写周报"}},
		PomodoroSummary: SessionSummary{
			Count:        3,
			TotalMinutes: 75,
		},
	}
	s := formatContextForPrompt(ctx)
	if !strings.Contains(s, "写周报") {
		t.Errorf("missing task title: %s", s)
	}
	if !strings.Contains(s, "已完成任务 1 条") {
		t.Errorf("missing task count: %s", s)
	}
	if !strings.Contains(s, "3 个") {
		t.Errorf("missing session count: %s", s)
	}
	if !strings.Contains(s, "75 分钟") {
		t.Errorf("missing total minutes: %s", s)
	}
}

func TestTruncated_Short(t *testing.T) {
	in := "short"
	if got := truncated(in, 100); got != in {
		t.Errorf("got=%q want=%q", got, in)
	}
}

func TestTruncated_Long(t *testing.T) {
	in := strings.Repeat("a", 150)
	got := truncated(in, 100)
	if !strings.HasSuffix(got, "...(truncated)") {
		t.Errorf("missing suffix: %q", got)
	}
	if len(got) > 120 {
		t.Errorf("got too long: %d", len(got))
	}
}
