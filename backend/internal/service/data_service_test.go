package service

import (
	"errors"
	"testing"
	"ticktask/internal/model"
)

// mockBackupRepo 实现 repository.BackupRepository,内存快照 + 捕获 Apply。
type mockBackupRepo struct {
	snapshot    *model.BackupData
	lastPlan    *model.ApplyPlan
	applyErr    error
	clearResult *model.ClearResult
	clearErr    error
}

func (m *mockBackupRepo) ReadAll() (*model.BackupData, error) {
	return m.snapshot, nil
}
func (m *mockBackupRepo) Apply(plan model.ApplyPlan) error {
	m.lastPlan = &plan
	return m.applyErr
}

func (m *mockBackupRepo) ClearAll() (*model.ClearResult, error) {
	return m.clearResult, m.clearErr
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
	env, err := svc.Export(true)
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
	env, err := svc.Export(true)
	if err != nil {
		t.Fatalf("export empty: %v", err)
	}
	if len(env.Data.Tasks) != 0 {
		t.Error("empty snapshot should export empty arrays")
	}
}

func TestDataService_Export_ExcludesAPIKey(t *testing.T) {
	// snapshot with a real AI key present
	snap := newSnapshot()
	snap.Settings.AI = &model.AISettings{APIKey: "secret"}

	cases := []struct {
		name        string
		include     bool
		wantAPIKey  string
	}{
		{"include key (default)", true, "secret"},
		{"exclude key", false, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := NewDataService(&mockBackupRepo{snapshot: snap})
			env, err := svc.Export(c.include)
			if err != nil {
				t.Fatalf("export: %v", err)
			}
			if env.Data.Settings.AI == nil {
				t.Fatal("AI settings nil")
			}
			if env.Data.Settings.AI.APIKey != c.wantAPIKey {
				t.Errorf("api_key: got %q, want %q", env.Data.Settings.AI.APIKey, c.wantAPIKey)
			}
		})
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

func TestDataService_ApplyImport_Policies(t *testing.T) {
	// 当前库:t1(title=x);文件:t1(title=changed) + t2(new)
	cur := newSnapshot()
	file := &model.BackupData{Tasks: []model.Task{
		{ID: "t1", Title: "changed"},
		{ID: "t2", Title: "brand"},
	}}

	cases := []struct {
		policy                        string
		inserted, updated, deleted    int
	}{
		{model.PolicyAddNewOnly, 1, 0, 0},
		{model.PolicyMergeFile, 1, 1, 0},
		{model.PolicyMergeCurrent, 1, 0, 0},
		{model.PolicyReplace, 1, 1, 0},
	}
	for _, c := range cases {
		t.Run(c.policy, func(t *testing.T) {
			repo := &mockBackupRepo{snapshot: cur}
			svc := NewDataService(repo)
			res, err := svc.ApplyImport(&model.ApplyImportRequest{
				Data:    *file,
				Modules: map[string]model.ModuleApply{"tasks": {Policy: c.policy}},
			})
			if err != nil {
				t.Fatalf("apply: %v", err)
			}
			tr := res.Applied["tasks"]
			if tr.Inserted != c.inserted || tr.Updated != c.updated || tr.Deleted != c.deleted {
				t.Errorf("counts: got i%d u%d d%d, want i%d u%d d%d", tr.Inserted, tr.Updated, tr.Deleted, c.inserted, c.updated, c.deleted)
			}
			// updated>0 ⇒ t1 was resolved to file ⇒ must be in plan.
			// updated==0 ⇒ t1 kept current ⇒ must NOT be in plan.
			t1InPlan := false
			for _, x := range repo.lastPlan.Tasks {
				if x.ID == "t1" {
					t1InPlan = true
				}
			}
			if c.updated > 0 && !t1InPlan {
				t.Errorf("policy %s: expected t1 in plan (file wins), not found", c.policy)
			}
			if c.updated == 0 && t1InPlan {
				t.Errorf("policy %s: expected t1 NOT in plan (current kept), but found", c.policy)
			}
		})
	}
}

func TestDataService_ApplyImport_ReplaceDeletesOrphans(t *testing.T) {
	cur := &model.BackupData{Tasks: []model.Task{{ID: "orphan", Title: "o"}}}
	file := &model.BackupData{Tasks: []model.Task{}} // 文件没有 orphan
	repo := &mockBackupRepo{snapshot: cur}
	svc := NewDataService(repo)
	res, err := svc.ApplyImport(&model.ApplyImportRequest{
		Data:    *file,
		Modules: map[string]model.ModuleApply{"tasks": {Policy: model.PolicyReplace}},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.Applied["tasks"].Deleted != 1 {
		t.Errorf("replace should delete orphan, got deleted=%d", res.Applied["tasks"].Deleted)
	}
	if len(repo.lastPlan.DeleteTasks) != 1 || repo.lastPlan.DeleteTasks[0] != "orphan" {
		t.Errorf("orphan not in delete plan: %+v", repo.lastPlan.DeleteTasks)
	}
}

func TestDataService_ApplyImport_OverrideBeatsPolicy(t *testing.T) {
	cur := newSnapshot()
	file := &model.BackupData{Tasks: []model.Task{{ID: "t1", Title: "changed"}}}
	repo := &mockBackupRepo{snapshot: cur}
	svc := NewDataService(repo)
	_, err := svc.ApplyImport(&model.ApplyImportRequest{
		Data: *file,
		Modules: map[string]model.ModuleApply{"tasks": {
			Policy:    model.PolicyMergeFile,            // 本应把 t1 放进 plan
			Overrides: map[string]string{"t1": model.ChoiceCurrent}, // 但 override 强制保留当前
		}},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	for _, x := range repo.lastPlan.Tasks {
		if x.ID == "t1" {
			t.Errorf("override=current should keep t1 OUT of plan, but found %+v", x)
		}
	}
}

func TestDataService_ApplyImport_SettingsWritten(t *testing.T) {
	cur := newSnapshot()
	pomo := model.DefaultPomodoroSettings()
	pomo.WorkDuration = 1800
	file := &model.BackupData{Settings: model.SettingsBundle{Pomodoro: pomo, AI: model.DefaultAISettings()}}
	repo := &mockBackupRepo{snapshot: cur}
	svc := NewDataService(repo)
	_, err := svc.ApplyImport(&model.ApplyImportRequest{Data: *file, Modules: map[string]model.ModuleApply{}})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if repo.lastPlan.Settings == nil || repo.lastPlan.Settings.Pomodoro.WorkDuration != 1800 {
		t.Errorf("settings not in plan: %+v", repo.lastPlan.Settings)
	}
}

func TestDataService_ApplyImport_UnknownOverrideIgnored(t *testing.T) {
	cur := newSnapshot()
	file := newSnapshot()
	repo := &mockBackupRepo{snapshot: cur}
	svc := NewDataService(repo)
	_, err := svc.ApplyImport(&model.ApplyImportRequest{
		Data: *file,
		Modules: map[string]model.ModuleApply{"tasks": {
			Policy:    model.PolicyMergeFile,
			Overrides: map[string]string{"does-not-exist": model.ChoiceFile},
		}},
	})
	if err != nil {
		t.Errorf("unknown override id should be ignored, not error: %v", err)
	}
}

func TestDataService_ApplyImport_InvalidPolicy(t *testing.T) {
	svc := NewDataService(&mockBackupRepo{snapshot: newSnapshot()})
	_, err := svc.ApplyImport(&model.ApplyImportRequest{
		Data:    *newSnapshot(),
		Modules: map[string]model.ModuleApply{"tasks": {Policy: "bogus"}},
	})
	if err == nil {
		t.Fatal("invalid policy should error")
	}
	if !errors.Is(err, ErrInvalidPolicy) {
		t.Errorf("error should wrap ErrInvalidPolicy, got %v", err)
	}
}
