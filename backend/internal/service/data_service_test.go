package service

import (
	"testing"
	"ticktask/internal/model"
)

// mockBackupRepo 实现 repository.BackupRepository,内存快照 + 捕获 Apply。
type mockBackupRepo struct {
	snapshot *model.BackupData
	lastPlan *model.ApplyPlan
	applyErr error
}

func (m *mockBackupRepo) ReadAll() (*model.BackupData, error) {
	return m.snapshot, nil
}
func (m *mockBackupRepo) Apply(plan model.ApplyPlan) error {
	m.lastPlan = &plan
	return m.applyErr
}

func newSnapshot() *model.BackupData {
	return &model.BackupData{
		Tasks:    []model.Task{{ID: "t1", Title: "x"}},
		Sessions: []model.PomodoroSession{{ID: "s1"}},
		Settings: model.SettingsBundle{Pomodoro: model.DefaultPomodoroSettings(), AI: model.DefaultAISettings()},
		WorkLogs: []model.WorkLog{{ID: "wl1", Date: "2026-08-07", Items: []model.WorkItem{{ID: "wi1", WorkLogID: "wl1"}}}},
	}
}

func TestDataService_Export(t *testing.T) {
	svc := NewDataService(&mockBackupRepo{snapshot: newSnapshot()})
	env, err := svc.Export()
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if env.App != "ticktask" || env.SchemaVersion != model.BackupSchemaVersion {
		t.Errorf("envelope meta wrong: %+v", env)
	}
	if env.ExportedAt.IsZero() {
		t.Error("exported_at should be set")
	}
	if len(env.Data.Tasks) != 1 || len(env.Data.WorkLogs) != 1 {
		t.Errorf("data lost: %+v", env.Data)
	}
	if env.Data.WorkLogs[0].Items[0].ID != "wi1" {
		t.Error("items not nested in export")
	}
	if env.Data.Settings.Pomodoro.WorkDuration != 1500 {
		t.Errorf("pomodoro default wrong: %d", env.Data.Settings.Pomodoro.WorkDuration)
	}
}

func TestDataService_Export_Empty(t *testing.T) {
	svc := NewDataService(&mockBackupRepo{snapshot: &model.BackupData{}})
	env, err := svc.Export()
	if err != nil {
		t.Fatalf("export empty: %v", err)
	}
	if len(env.Data.Tasks) != 0 {
		t.Error("empty snapshot should export empty arrays")
	}
}
