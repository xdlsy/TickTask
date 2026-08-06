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

func TestDataService_PreviewImport_Classification(t *testing.T) {
	cur := newSnapshot() // 1 task t1, 1 session s1, 1 worklog wl1(wi1)

	cases := []struct {
		name                       string
		file                       *model.BackupData
		module                     string
		newN, identN, confN, orphN int
	}{
		{"all new", &model.BackupData{Tasks: []model.Task{{ID: "t2"}}}, "tasks", 1, 0, 0, 1},
		{"all identical", &model.BackupData{Tasks: []model.Task{{ID: "t1", Title: "x"}}}, "tasks", 0, 1, 0, 0},
		{"conflict", &model.BackupData{Tasks: []model.Task{{ID: "t1", Title: "changed"}}}, "tasks", 0, 0, 1, 0},
		{"orphan only", &model.BackupData{Sessions: []model.PomodoroSession{{ID: "other"}}}, "sessions", 1, 0, 0, 1},
		{"empty file vs full current", &model.BackupData{}, "tasks", 0, 0, 0, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := NewDataService(&mockBackupRepo{snapshot: cur})
			prev, err := svc.PreviewImport(c.file, model.BackupSchemaVersion)
			if err != nil {
				t.Fatalf("preview: %v", err)
			}
			m := prev.Modules[c.module]
			if m == nil {
				t.Fatalf("module %s missing", c.module)
			}
			if m.New != c.newN || m.Identical != c.identN || m.Conflict != c.confN || m.Orphan != c.orphN {
				t.Errorf("%s: got new=%d ident=%d conf=%d orphan=%d", c.name, m.New, m.Identical, m.Conflict, m.Orphan)
			}
		})
	}
}

func TestDataService_PreviewImport_ConflictFieldDiff(t *testing.T) {
	svc := NewDataService(&mockBackupRepo{snapshot: newSnapshot()})
	prev, _ := svc.PreviewImport(&model.BackupData{Tasks: []model.Task{{ID: "t1", Title: "changed", Status: model.StatusCompleted}}}, model.BackupSchemaVersion)
	m := prev.Modules["tasks"]
	if len(m.Conflicts) != 1 || m.Conflicts[0].ID != "t1" {
		t.Fatalf("conflict not recorded: %+v", m.Conflicts)
	}
	fields := map[string]bool{}
	for _, f := range m.Conflicts[0].Fields {
		fields[f.Field] = true
	}
	if !fields["title"] || !fields["status"] {
		t.Errorf("expected title+status diffs, got %+v", m.Conflicts[0].Fields)
	}
}

func TestDataService_PreviewImport_SettingsDiff(t *testing.T) {
	svc := NewDataService(&mockBackupRepo{snapshot: newSnapshot()})
	filePomo := model.DefaultPomodoroSettings()
	filePomo.WorkDuration = 1800
	fileAI := model.DefaultAISettings()
	fileAI.APIKey = "secret"
	file := &model.BackupData{Settings: model.SettingsBundle{Pomodoro: filePomo, AI: fileAI}}

	prev, _ := svc.PreviewImport(file, model.BackupSchemaVersion)
	m := prev.Modules["settings"]
	if m == nil {
		t.Fatal("settings module missing")
	}
	gotFields := map[string]bool{}
	for _, f := range m.SettingsConflicts {
		gotFields[f.Section+"."+f.Field] = true
	}
	if !gotFields["pomodoro.work_duration"] {
		t.Errorf("pomodoro.work_duration diff missing: %+v", m.SettingsConflicts)
	}
	if !gotFields["ai.api_key"] {
		t.Errorf("ai.api_key diff missing (backend must NOT mask): %+v", m.SettingsConflicts)
	}
}

func TestDataService_PreviewImport_WorkLogAtomicConflict(t *testing.T) {
	svc := NewDataService(&mockBackupRepo{snapshot: newSnapshot()})
	file := &model.BackupData{WorkLogs: []model.WorkLog{{ID: "wl1", Date: "2026-08-07", Items: []model.WorkItem{
		{ID: "wi1", WorkLogID: "wl1", Seq: 1},
		{ID: "wi2", WorkLogID: "wl1", Seq: 2},
	}}}}
	prev, _ := svc.PreviewImport(file, model.BackupSchemaVersion)
	m := prev.Modules["work_logs"]
	if m.Conflict != 1 {
		t.Errorf("worklog with differing items should be conflict, got conflict=%d", m.Conflict)
	}
}

func TestDataService_PreviewImport_SchemaWarning(t *testing.T) {
	svc := NewDataService(&mockBackupRepo{snapshot: newSnapshot()})
	same, _ := svc.PreviewImport(newSnapshot(), model.BackupSchemaVersion)
	if same.SchemaWarning != "" {
		t.Errorf("same-version should have no warning, got %q", same.SchemaWarning)
	}
	diff, _ := svc.PreviewImport(newSnapshot(), model.BackupSchemaVersion+1)
	if diff.SchemaWarning == "" {
		t.Error("mismatched version should produce a warning")
	}
}
