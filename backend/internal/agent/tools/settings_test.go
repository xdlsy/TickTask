package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"ticktask/internal/model"
)

type mockSettingsReader struct {
	out *model.PomodoroSettings
	err error
}

func (m *mockSettingsReader) GetPomodoroSettings() (*model.PomodoroSettings, error) {
	return m.out, m.err
}

func TestGetSettings_Delegates(t *testing.T) {
	svc := &mockSettingsReader{out: &model.PomodoroSettings{WorkDuration: 25, ShortBreakDuration: 5}}
	tool := &GetSettingsTool{Svc: svc}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	m, _ := json.Marshal(res)
	if !strings.Contains(string(m), `"work_duration":25`) {
		t.Errorf("result should include pomodoro settings: %s", m)
	}
}

func TestGetSettings_PreviewMirrorsExecute(t *testing.T) {
	svc := &mockSettingsReader{out: &model.PomodoroSettings{}}
	tool := &GetSettingsTool{Svc: svc}
	pv, err := tool.Preview(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if pv == nil {
		t.Fatal("preview should mirror execute for read tool")
	}
}

func TestGetSettings_ServiceError(t *testing.T) {
	svc := &mockSettingsReader{err: errors.New("db locked")}
	tool := &GetSettingsTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err == nil || !strings.Contains(err.Error(), "db locked") {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}
