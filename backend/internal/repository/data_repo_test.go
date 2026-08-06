package repository

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"ticktask/internal/model"
)

func newDataTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Task{}, &model.PomodoroSession{}, &model.Schedule{},
		&model.Setting{}, &model.DailyStats{},
		&model.WorkLog{}, &model.WorkItem{}, &model.WorkReport{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestBackupRepo_ReadAll_Empty(t *testing.T) {
	repo := NewDataRepository(newDataTestDB(t))
	data, err := repo.ReadAll()
	if err != nil {
		t.Fatalf("readall: %v", err)
	}
	if len(data.Tasks) != 0 || len(data.WorkLogs) != 0 {
		t.Errorf("empty db should yield empty slices: %+v", data)
	}
	if data.Settings.Pomodoro == nil || data.Settings.AI == nil {
		t.Error("empty db should still yield default settings bundle")
	}
	if data.Settings.Pomodoro.WorkDuration != 1500 {
		t.Errorf("default work_duration = %d, want 1500", data.Settings.Pomodoro.WorkDuration)
	}
}

func TestBackupRepo_ReadAll_AssembledAndNested(t *testing.T) {
	db := newDataTestDB(t)
	// Fixture 调整：Task.Quadrant / PomodoroSession.Type / PomodoroSession.PlannedDuration
	// 在 model 中是 gorm:"not null" 且无默认值，必须在 fixture 显式提供。
	db.Create(&model.Task{ID: "t1", Title: "task", Status: model.StatusTodo, Quadrant: model.Quadrant1})
	db.Create(&model.PomodoroSession{ID: "s1", TaskID: strPtr("t1"), Type: model.SessionWork, PlannedDuration: 1500})
	db.Create(&model.WorkLog{ID: "wl1", Date: "2026-08-07"})
	db.Create(&model.WorkItem{ID: "wi1", WorkLogID: "wl1", Seq: 1, Source: "ai"})
	db.Create(&model.Setting{Key: "pomodoro.settings", Value: `{"work_duration":1800,"short_break_duration":300,"long_break_duration":900,"long_break_after":4,"auto_start_break":false,"auto_start_work":false,"enable_sound":true,"buffer_ratio":20,"task_time_preferences":"{\"management\":\"any\",\"dev\":\"any\"}"}`})
	db.Create(&model.Setting{Key: "ai.settings", Value: `{"provider":"anthropic","api_key":"k","base_url":"u","model":"m"}`})
	db.Create(&model.DailyStats{}) // 必须被排除

	repo := NewDataRepository(db)
	data, err := repo.ReadAll()
	if err != nil {
		t.Fatalf("readall: %v", err)
	}
	if len(data.Tasks) != 1 || data.Tasks[0].ID != "t1" {
		t.Errorf("tasks: %+v", data.Tasks)
	}
	if len(data.Sessions) != 1 || data.Sessions[0].ID != "s1" {
		t.Errorf("sessions: %+v", data.Sessions)
	}
	if len(data.WorkLogs) != 1 || len(data.WorkLogs[0].Items) != 1 {
		t.Errorf("work_logs items not nested: %+v", data.WorkLogs)
	}
	if data.Settings.Pomodoro.WorkDuration != 1800 {
		t.Errorf("pomodoro not assembled: %d", data.Settings.Pomodoro.WorkDuration)
	}
	if data.Settings.AI.Provider != "anthropic" {
		t.Errorf("ai not assembled: %+v", data.Settings.AI)
	}
}

func TestBackupRepo_Apply_InsertUpdateDeleteOrphan(t *testing.T) {
	db := newDataTestDB(t)
	db.Create(&model.Task{ID: "exist", Title: "old", Status: model.StatusTodo, Quadrant: model.Quadrant1})
	repo := NewDataRepository(db)

	plan := model.ApplyPlan{
		Tasks: []model.Task{
			{ID: "exist", Title: "new", Status: model.StatusTodo, Quadrant: model.Quadrant1}, // 更新
			{ID: "fresh", Title: "brand", Status: model.StatusTodo, Quadrant: model.Quadrant1}, // 新增
		},
		DeleteTasks: []string{"exist-orphan"}, // 不存在,无副作用
	}
	if err := repo.Apply(plan); err != nil {
		t.Fatalf("apply: %v", err)
	}

	var got []model.Task
	db.Find(&got)
	byID := map[string]model.Task{}
	for _, x := range got {
		byID[x.ID] = x
	}
	if byID["exist"].Title != "new" {
		t.Errorf("exist should be updated to 'new', got %q", byID["exist"].Title)
	}
	if _, ok := byID["fresh"]; !ok {
		t.Error("fresh should be inserted")
	}
}

func TestBackupRepo_Apply_SettingsUpsert(t *testing.T) {
	db := newDataTestDB(t)
	repo := NewDataRepository(db)
	pomo := model.DefaultPomodoroSettings()
	pomo.WorkDuration = 999
	ai := model.DefaultAISettings()
	ai.Model = "mymodel"
	if err := repo.Apply(model.ApplyPlan{Settings: &model.SettingsBundle{Pomodoro: pomo, AI: ai}}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	data, _ := repo.ReadAll()
	if data.Settings.Pomodoro.WorkDuration != 999 || data.Settings.AI.Model != "mymodel" {
		t.Errorf("settings not persisted: %+v", data.Settings)
	}
}

func TestBackupRepo_Apply_WorkLogReplacesItemsAtomically(t *testing.T) {
	db := newDataTestDB(t)
	db.Create(&model.WorkLog{ID: "wl1", Date: "2026-08-07"})
	db.Create(&model.WorkItem{ID: "old-item", WorkLogID: "wl1", Seq: 1, Source: "ai"})
	repo := NewDataRepository(db)

	plan := model.ApplyPlan{
		WorkLogs: []model.WorkLog{
			{ID: "wl1", Date: "2026-08-07", Summary: "upd", Items: []model.WorkItem{
				{ID: "new-a", WorkLogID: "wl1", Seq: 1, Source: "ai"},
				{ID: "new-b", WorkLogID: "wl1", Seq: 2, Source: "ai"},
			}},
		},
	}
	if err := repo.Apply(plan); err != nil {
		t.Fatalf("apply: %v", err)
	}

	var items []model.WorkItem
	db.Where("work_log_id = ?", "wl1").Find(&items)
	if len(items) != 2 {
		t.Fatalf("expected 2 items after replace, got %d", len(items))
	}
	var log model.WorkLog
	db.Where("id = ?", "wl1").First(&log)
	if log.Summary != "upd" {
		t.Errorf("log scalar not updated: %q", log.Summary)
	}
}

func TestBackupRepo_Apply_TransactionRollback(t *testing.T) {
	db := newDataTestDB(t)
	db.Create(&model.WorkLog{ID: "wl1", Date: "2026-08-01", Summary: "keep"})
	repo := NewDataRepository(db)

	// wl1 正常更新;wl2 的 items 含重复 PK → Create 报错 → 整体回滚
	plan := model.ApplyPlan{
		WorkLogs: []model.WorkLog{
			{ID: "wl1", Date: "2026-08-01", Summary: "changed"},
			{ID: "wl2", Date: "2026-08-02", Items: []model.WorkItem{
				{ID: "dup", WorkLogID: "wl2", Seq: 1, Source: "ai"},
				{ID: "dup", WorkLogID: "wl2", Seq: 2, Source: "ai"},
			}},
		},
	}
	err := repo.Apply(plan)
	if err == nil {
		t.Fatal("expected error from duplicate PK, got nil")
	}

	var log model.WorkLog
	db.Where("id = ?", "wl1").First(&log)
	if log.Summary != "keep" {
		t.Errorf("wl1 should be unchanged after rollback, got %q", log.Summary)
	}
}

func TestBackupRepo_Apply_DeleteOrphans(t *testing.T) {
	db := newDataTestDB(t)
	db.Create(&model.Task{ID: "t-del", Title: "x", Status: model.StatusTodo, Quadrant: model.Quadrant1})
	db.Create(&model.WorkLog{ID: "wl-del", Date: "2026-08-01"})
	db.Create(&model.WorkItem{ID: "wi-del", WorkLogID: "wl-del", Seq: 1, Source: "ai"})
	repo := NewDataRepository(db)

	if err := repo.Apply(model.ApplyPlan{
		DeleteTasks:    []string{"t-del"},
		DeleteWorkLogs: []string{"wl-del"},
	}); err != nil {
		t.Fatalf("apply: %v", err)
	}

	var n int64
	db.Model(&model.Task{}).Count(&n)
	if n != 0 {
		t.Errorf("task orphan should be deleted, count=%d", n)
	}
	db.Model(&model.WorkItem{}).Where("work_log_id = ?", "wl-del").Count(&n)
	if n != 0 {
		t.Errorf("worklog orphan's items should be deleted, count=%d", n)
	}
}
