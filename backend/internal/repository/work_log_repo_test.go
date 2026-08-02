// backend/internal/repository/work_log_repo_test.go
package repository

import (
	"testing"

	"ticktask/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.WorkLog{}, &model.WorkItem{}, &model.WorkReport{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// 清空表避免 cache=shared 跨测试污染
	db.Exec("DELETE FROM work_items")
	db.Exec("DELETE FROM work_logs")
	db.Exec("DELETE FROM work_reports")
	return db
}

func TestCreateAndGetWorkLogByDate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkLogRepository(db)

	log := &model.WorkLog{
		ID:    "wl-1",
		Date:  "2026-08-02",
		Items: []model.WorkItem{{ID: "wi-1", WorkLogID: "wl-1", Seq: 1, Title: "T1"}},
	}
	if err := repo.CreateWorkLog(log); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.GetWorkLogByDate("2026-08-02")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != "wl-1" {
		t.Errorf("id = %s, want wl-1", got.ID)
	}
	if len(got.Items) != 1 || got.Items[0].Title != "T1" {
		t.Errorf("items mismatch: %+v", got.Items)
	}
}

func TestGetWorkLogByDate_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkLogRepository(db)

	_, err := repo.GetWorkLogByDate("2026-08-02")
	if err != ErrNotFound {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestUpsertWorkLog_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkLogRepository(db)

	log := &model.WorkLog{
		ID:    "wl-1",
		Date:  "2026-08-02",
		Items: []model.WorkItem{{ID: "wi-1", WorkLogID: "wl-1", Seq: 1, Title: "T1"}},
	}
	if err := repo.UpsertWorkLog(log); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got, _ := repo.GetWorkLogByDate("2026-08-02")
	if len(got.Items) != 1 {
		t.Errorf("items len = %d, want 1", len(got.Items))
	}
}

func TestUpsertWorkLog_ReplaceItems(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkLogRepository(db)

	// 第一次保存：1 个 item
	repo.UpsertWorkLog(&model.WorkLog{
		ID: "wl-1", Date: "2026-08-02",
		Items: []model.WorkItem{{ID: "wi-1", WorkLogID: "wl-1", Seq: 1, Title: "T1"}},
	})
	// 第二次保存：2 个 item（不同 title）
	repo.UpsertWorkLog(&model.WorkLog{
		ID: "wl-1", Date: "2026-08-02",
		Items: []model.WorkItem{
			{ID: "wi-2", WorkLogID: "wl-1", Seq: 1, Title: "T2"},
			{ID: "wi-3", WorkLogID: "wl-1", Seq: 2, Title: "T3"},
		},
	})

	got, _ := repo.GetWorkLogByDate("2026-08-02")
	if len(got.Items) != 2 {
		t.Fatalf("items len = %d, want 2 (full replace)", len(got.Items))
	}
	for _, it := range got.Items {
		if it.Title == "T1" {
			t.Errorf("old item T1 should be deleted; got items: %+v", got.Items)
		}
	}
}

func TestGetWorkLogsInRange(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkLogRepository(db)

	repo.CreateWorkLog(&model.WorkLog{ID: "wl-1", Date: "2026-08-01"})
	repo.CreateWorkLog(&model.WorkLog{ID: "wl-2", Date: "2026-08-02"})
	repo.CreateWorkLog(&model.WorkLog{ID: "wl-3", Date: "2026-08-03"})

	logs, err := repo.GetWorkLogsInRange("2026-08-01", "2026-08-02")
	if err != nil {
		t.Fatalf("range: %v", err)
	}
	if len(logs) != 2 {
		t.Errorf("len = %d, want 2", len(logs))
	}
	// 应按 date DESC 排序
	if logs[0].Date != "2026-08-02" {
		t.Errorf("order wrong: %+v", logs)
	}
}

func TestCreateWorkReport_UniqueIndex(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkLogRepository(db)

	r1 := &model.WorkReport{ID: "wr-1", Type: model.ReportWeekly, PeriodKey: "2026-W31", StartDate: "2026-07-27", EndDate: "2026-08-02"}
	if err := repo.CreateWorkReport(r1); err != nil {
		t.Fatalf("create r1: %v", err)
	}

	r2 := &model.WorkReport{ID: "wr-2", Type: model.ReportWeekly, PeriodKey: "2026-W31", StartDate: "2026-07-27", EndDate: "2026-08-02"}
	if err := repo.CreateWorkReport(r2); err == nil {
		t.Errorf("expected unique constraint violation, got nil")
	}
}
