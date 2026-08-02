package database

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"ticktask/internal/model"
)

func newMigrateTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(&model.WorkLog{}, &model.WorkItem{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestMigrateWorkItemsTitleBackfill_FillsEmptyTitleFromActivity(t *testing.T) {
	db := newMigrateTestDB(t)
	activity := "晨会"
	db.Create(&model.WorkLog{ID: "wl-1", Date: "2026-08-01"})
	db.Create(&model.WorkItem{
		ID: "wi-1", WorkLogID: "wl-1", Seq: 1,
		Title: "", Activity: &activity, Source: "manual",
	})

	if err := MigrateWorkItemsTitleBackfill(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	var got model.WorkItem
	db.First(&got, "id = ?", "wi-1")
	if got.Title != activity {
		t.Fatalf("expected title=%q, got %q", activity, got.Title)
	}
}

func TestMigrateWorkItemsTitleBackfill_SkipsAIItems(t *testing.T) {
	db := newMigrateTestDB(t)
	db.Create(&model.WorkLog{ID: "wl-1", Date: "2026-08-01"})
	db.Create(&model.WorkItem{
		ID: "wi-1", WorkLogID: "wl-1", Seq: 1,
		Title: "AI 生成的标题", Source: "ai", // Activity nil
	})

	if err := MigrateWorkItemsTitleBackfill(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	var got model.WorkItem
	db.First(&got, "id = ?", "wi-1")
	if got.Title != "AI 生成的标题" {
		t.Fatalf("AI item title should be untouched, got %q", got.Title)
	}
}

func TestMigrateWorkItemsTitleBackfill_Idempotent(t *testing.T) {
	db := newMigrateTestDB(t)
	activity := "晨会"
	db.Create(&model.WorkLog{ID: "wl-1", Date: "2026-08-01"})
	db.Create(&model.WorkItem{
		ID: "wi-1", WorkLogID: "wl-1", Seq: 1,
		Title: "", Activity: &activity, Source: "manual",
	})

	// 第一次
	if err := MigrateWorkItemsTitleBackfill(db); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	var got model.WorkItem
	db.First(&got, "id = ?", "wi-1")
	firstTitle := got.Title

	// 第二次
	if err := MigrateWorkItemsTitleBackfill(db); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	db.First(&got, "id = ?", "wi-1")
	if got.Title != firstTitle {
		t.Fatalf("second run changed title: %q → %q", firstTitle, got.Title)
	}
}
