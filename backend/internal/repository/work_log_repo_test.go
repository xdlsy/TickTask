// backend/internal/repository/work_log_repo_test.go
package repository

import (
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"ticktask/internal/model"
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

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.WorkLog{}, &model.WorkItem{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

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

func TestAppendItem_ToExistingWorkLog(t *testing.T) {
	db := newTestDB(t)
	repo := NewWorkLogRepository(db)
	log := &model.WorkLog{ID: uuid.New().String(), Date: "2026-08-02"}
	if err := repo.CreateWorkLog(log); err != nil {
		t.Fatalf("create log: %v", err)
	}
	item := model.WorkItem{
		Activity:  strPtr("晨会"),
		StartTime: strPtr("09:00"),
		EndTime:   strPtr("10:00"),
		Quadrant:  intPtr(1),
		Source:    "manual",
		Seq:       1,
	}
	if err := repo.AppendItem(log.ID, item); err != nil {
		t.Fatalf("append: %v", err)
	}
	got, err := repo.GetWorkLogByDate("2026-08-02")
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(got.Items) != 1 || got.Items[0].Activity == nil || *got.Items[0].Activity != "晨会" {
		t.Fatalf("unexpected items: %+v", got.Items)
	}
}

func TestAppendItem_WorkLogNotExists(t *testing.T) {
	db := newTestDB(t)
	repo := NewWorkLogRepository(db)
	err := repo.AppendItem("nonexistent", model.WorkItem{Source: "manual"})
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateItem_ManualSucceeds(t *testing.T) {
	db := newTestDB(t)
	repo := NewWorkLogRepository(db)
	log := &model.WorkLog{ID: uuid.New().String(), Date: "2026-08-02"}
	repo.CreateWorkLog(log)
	item := model.WorkItem{ID: uuid.New().String(), WorkLogID: log.ID, Source: "manual", Activity: strPtr("old"), Seq: 1}
	repo.AppendItem(log.ID, item)
	updates := map[string]any{"activity": "new"}
	if err := repo.UpdateItem(log.ID, item.ID, updates); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ := repo.GetWorkLogByDate("2026-08-02")
	if *got.Items[0].Activity != "new" {
		t.Fatalf("activity not updated: %+v", got.Items[0])
	}
}

func TestUpdateItem_AIItemReturnsErrItemNotEditable(t *testing.T) {
	db := newTestDB(t)
	repo := NewWorkLogRepository(db)
	log := &model.WorkLog{ID: uuid.New().String(), Date: "2026-08-02"}
	repo.CreateWorkLog(log)
	item := model.WorkItem{ID: uuid.New().String(), WorkLogID: log.ID, Source: "ai", Title: "x", Seq: 1}
	repo.AppendItem(log.ID, item)
	err := repo.UpdateItem(log.ID, item.ID, map[string]any{"title": "y"})
	if err != ErrItemNotEditable {
		t.Fatalf("expected ErrItemNotEditable, got %v", err)
	}
}

func TestUpdateItem_NotExistsReturnsErrItemNotFound(t *testing.T) {
	db := newTestDB(t)
	repo := NewWorkLogRepository(db)
	log := &model.WorkLog{ID: uuid.New().String(), Date: "2026-08-02"}
	repo.CreateWorkLog(log)
	err := repo.UpdateItem(log.ID, "nonexistent", map[string]any{"activity": "x"})
	if err != ErrItemNotFound {
		t.Fatalf("expected ErrItemNotFound, got %v", err)
	}
}

func TestDeleteItem_ManualSucceeds(t *testing.T) {
	db := newTestDB(t)
	repo := NewWorkLogRepository(db)
	log := &model.WorkLog{ID: uuid.New().String(), Date: "2026-08-02"}
	repo.CreateWorkLog(log)
	item := model.WorkItem{ID: uuid.New().String(), WorkLogID: log.ID, Source: "manual", Seq: 1}
	repo.AppendItem(log.ID, item)
	if err := repo.DeleteItem(log.ID, item.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, _ := repo.GetWorkLogByDate("2026-08-02")
	if len(got.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(got.Items))
	}
}

func TestDeleteItem_AIItemReturnsErrItemNotEditable(t *testing.T) {
	db := newTestDB(t)
	repo := NewWorkLogRepository(db)
	log := &model.WorkLog{ID: uuid.New().String(), Date: "2026-08-02"}
	repo.CreateWorkLog(log)
	item := model.WorkItem{ID: uuid.New().String(), WorkLogID: log.ID, Source: "ai", Title: "x", Seq: 1}
	repo.AppendItem(log.ID, item)
	err := repo.DeleteItem(log.ID, item.ID)
	if err != ErrItemNotEditable {
		t.Fatalf("expected ErrItemNotEditable, got %v", err)
	}
}
