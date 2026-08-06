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
