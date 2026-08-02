package service

import (
	"errors"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"ticktask/internal/model"
	"ticktask/internal/repository"
)

func newQuickTestDB(t *testing.T) (*gorm.DB, repository.WorkLogRepository) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(&model.WorkLog{}, &model.WorkItem{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db, repository.NewWorkLogRepository(db)
}

func newQuickService(t *testing.T) *WorkLogService {
	db, repo := newQuickTestDB(t)
	_ = db
	return NewWorkLogService(repo, nil, nil, nil)
}

func TestAddQuickEntry_AutoCreatesWorkLog(t *testing.T) {
	svc := newQuickService(t)
	in := CreateQuickEntryInput{
		Activity:  "晨会",
		StartTime: "09:00",
		EndTime:   "10:00",
		Quadrant:  1,
	}
	item, err := svc.AddQuickEntry("2026-08-02", in)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if item.Activity == nil || *item.Activity != "晨会" {
		t.Fatalf("bad item: %+v", item)
	}
	if item.Source != "manual" {
		t.Fatalf("expected source=manual, got %s", item.Source)
	}
	// 验证 WorkLog 自动建了
	log, err := svc.repo.GetWorkLogByDate("2026-08-02")
	if err != nil {
		t.Fatalf("reload log: %v", err)
	}
	if len(log.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(log.Items))
	}
}

func TestAddQuickEntry_AppendsToExistingWorkLog(t *testing.T) {
	svc := newQuickService(t)
	// 先建一条
	svc.AddQuickEntry("2026-08-02", CreateQuickEntryInput{Activity: "a", StartTime: "09:00", EndTime: "10:00", Quadrant: 1})
	// 再建一条
	_, err := svc.AddQuickEntry("2026-08-02", CreateQuickEntryInput{Activity: "b", StartTime: "10:00", EndTime: "11:00", Quadrant: 2})
	if err != nil {
		t.Fatalf("add 2nd: %v", err)
	}
	log, _ := svc.repo.GetWorkLogByDate("2026-08-02")
	if len(log.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(log.Items))
	}
	// 不依赖 Preload 顺序：按 activity 名找第二条
	var second *model.WorkItem
	for i := range log.Items {
		if log.Items[i].Activity != nil && *log.Items[i].Activity == "b" {
			second = &log.Items[i]
			break
		}
	}
	if second == nil {
		t.Fatalf("item with activity='b' not found in %+v", log.Items)
	}
	if second.Seq != 2 {
		t.Fatalf("expected seq=2, got %d", second.Seq)
	}
}

func TestAddQuickEntry_RejectsEndBeforeStart(t *testing.T) {
	svc := newQuickService(t)
	_, err := svc.AddQuickEntry("2026-08-02", CreateQuickEntryInput{
		Activity: "x", StartTime: "11:00", EndTime: "10:00", Quadrant: 1,
	})
	if err == nil {
		t.Fatal("expected error for end < start")
	}
}

func TestAddQuickEntry_RejectsEqualStartEnd(t *testing.T) {
	svc := newQuickService(t)
	_, err := svc.AddQuickEntry("2026-08-02", CreateQuickEntryInput{
		Activity: "x", StartTime: "09:00", EndTime: "09:00", Quadrant: 1,
	})
	if err == nil {
		t.Fatal("expected error for equal start/end")
	}
}

func TestAddQuickEntry_RejectsBadDate(t *testing.T) {
	svc := newQuickService(t)
	_, err := svc.AddQuickEntry("not-a-date", CreateQuickEntryInput{
		Activity: "x", StartTime: "09:00", EndTime: "10:00", Quadrant: 1,
	})
	if err == nil {
		t.Fatal("expected error for bad date")
	}
}

func TestUpdateQuickEntry_HappyPath(t *testing.T) {
	svc := newQuickService(t)
	item, _ := svc.AddQuickEntry("2026-08-02", CreateQuickEntryInput{Activity: "old", StartTime: "09:00", EndTime: "10:00", Quadrant: 1})
	err := svc.UpdateQuickEntry("2026-08-02", item.ID, UpdateQuickEntryInput{
		Activity: strPtrService("new"),
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	log, _ := svc.repo.GetWorkLogByDate("2026-08-02")
	if *log.Items[0].Activity != "new" {
		t.Fatalf("not updated: %+v", log.Items[0])
	}
}

func TestUpdateQuickEntry_NotFound(t *testing.T) {
	svc := newQuickService(t)
	err := svc.UpdateQuickEntry("2026-08-02", "nonexistent", UpdateQuickEntryInput{Activity: strPtrService("x")})
	if !errors.Is(err, repository.ErrItemNotFound) {
		t.Fatalf("expected ErrItemNotFound, got %v", err)
	}
}

func TestDeleteQuickEntry_HappyPath(t *testing.T) {
	svc := newQuickService(t)
	item, _ := svc.AddQuickEntry("2026-08-02", CreateQuickEntryInput{Activity: "x", StartTime: "09:00", EndTime: "10:00", Quadrant: 1})
	if err := svc.DeleteQuickEntry("2026-08-02", item.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	log, _ := svc.repo.GetWorkLogByDate("2026-08-02")
	if len(log.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(log.Items))
	}
}

func TestAddQuickEntry_RejectsBadTimeFormat(t *testing.T) {
	svc := newQuickService(t)
	// "25:00" 是非法小时值，time.Parse 应失败 — 必须先校验格式再做词法比较
	_, err := svc.AddQuickEntry("2026-08-02", CreateQuickEntryInput{
		Activity: "x", StartTime: "25:00", EndTime: "10:00", Quadrant: 1,
	})
	if err == nil {
		t.Fatal("expected error for invalid time format")
	}
}

func TestUpdateQuickEntry_RejectsBadTimeFormat(t *testing.T) {
	svc := newQuickService(t)
	item, _ := svc.AddQuickEntry("2026-08-02", CreateQuickEntryInput{
		Activity: "x", StartTime: "09:00", EndTime: "10:00", Quadrant: 1,
	})
	bad := "99:99"
	err := svc.UpdateQuickEntry("2026-08-02", item.ID, UpdateQuickEntryInput{
		StartTime: &bad,
	})
	if err == nil {
		t.Fatal("expected error for invalid time format")
	}
}

// 局部 helper：避免污染包级别
func strPtrService(s string) *string { return &s }

func TestUpdateSummary_UpdatesSummary(t *testing.T) {
	svc := newQuickService(t)
	// 先建一个 WorkLog
	svc.AddQuickEntry("2026-08-01", CreateQuickEntryInput{
		Activity: "x", StartTime: "09:00", EndTime: "10:00", Quadrant: 1,
	})

	err := svc.UpdateSummary("2026-08-01", "今日小结文本")
	if err != nil {
		t.Fatalf("update summary: %v", err)
	}

	log, _ := svc.repo.GetWorkLogByDate("2026-08-01")
	if log.Summary != "今日小结文本" {
		t.Fatalf("expected summary updated, got %q", log.Summary)
	}
}

func TestUpdateSummary_MissingWorkLogReturnsNotFound(t *testing.T) {
	svc := newQuickService(t)
	err := svc.UpdateSummary("2099-12-31", "x")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateSummary_InvalidDateReturnsBadRequest(t *testing.T) {
	svc := newQuickService(t)
	err := svc.UpdateSummary("not-a-date", "x")
	if err == nil || !strings.HasPrefix(err.Error(), "invalid date:") {
		t.Fatalf("expected invalid date error, got %v", err)
	}
}

func TestAddQuickEntry_WithOptionalFields_SyncsTitle(t *testing.T) {
	svc := newQuickService(t)
	in := CreateQuickEntryInput{
		Activity:      "登录开发",
		StartTime:     "09:00",
		EndTime:       "10:00",
		Quadrant:      2,
		Content:       "完成 OAuth 联调",
		ProblemSolved: "回调失败",
		Result:        "成功率提升",
		Impact:        "释放测试资源",
	}
	item, err := svc.AddQuickEntry("2026-08-03", in)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if item.Title != "登录开发" {
		t.Fatalf("expected title='登录开发', got %q", item.Title)
	}
	if item.Content != "完成 OAuth 联调" {
		t.Fatalf("expected content set, got %q", item.Content)
	}
	if item.ProblemSolved != "回调失败" {
		t.Fatalf("expected problem_solved set, got %q", item.ProblemSolved)
	}
	if item.Result != "成功率提升" {
		t.Fatalf("expected result set, got %q", item.Result)
	}
	if item.Impact != "释放测试资源" {
		t.Fatalf("expected impact set, got %q", item.Impact)
	}
}

func TestAddQuickEntry_NoOptionalFields_PersistsWithEmptyStrings(t *testing.T) {
	svc := newQuickService(t)
	in := CreateQuickEntryInput{
		Activity:  "x",
		StartTime: "09:00",
		EndTime:   "10:00",
		Quadrant:  1,
	}
	item, err := svc.AddQuickEntry("2026-08-03", in)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if item.Title != "x" {
		t.Fatalf("expected title synced from activity, got %q", item.Title)
	}
	if item.Content != "" || item.ProblemSolved != "" || item.Result != "" || item.Impact != "" {
		t.Fatalf("optional fields should be empty, got %+v", item)
	}
}

func TestUpdateQuickEntry_ActivityChangeSyncsTitle(t *testing.T) {
	svc := newQuickService(t)
	item, _ := svc.AddQuickEntry("2026-08-03", CreateQuickEntryInput{
		Activity: "old", StartTime: "09:00", EndTime: "10:00", Quadrant: 1,
	})

	newActivity := "new activity"
	err := svc.UpdateQuickEntry("2026-08-03", item.ID, UpdateQuickEntryInput{
		Activity: &newActivity,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	log, _ := svc.repo.GetWorkLogByDate("2026-08-03")
	for _, it := range log.Items {
		if it.ID == item.ID {
			if it.Title != "new activity" {
				t.Fatalf("expected title synced, got %q", it.Title)
			}
			if it.Activity == nil || *it.Activity != "new activity" {
				t.Fatalf("expected activity updated, got %v", it.Activity)
			}
		}
	}
}

func TestUpdateQuickEntry_OptionalFieldsUpdate(t *testing.T) {
	svc := newQuickService(t)
	item, _ := svc.AddQuickEntry("2026-08-03", CreateQuickEntryInput{
		Activity: "x", StartTime: "09:00", EndTime: "10:00", Quadrant: 1,
	})

	newContent := "新内容"
	newResult := "新结果"
	err := svc.UpdateQuickEntry("2026-08-03", item.ID, UpdateQuickEntryInput{
		Content: &newContent,
		Result:  &newResult,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	log, _ := svc.repo.GetWorkLogByDate("2026-08-03")
	for _, it := range log.Items {
		if it.ID == item.ID {
			if it.Content != "新内容" {
				t.Fatalf("expected content updated, got %q", it.Content)
			}
			if it.Result != "新结果" {
				t.Fatalf("expected result updated, got %q", it.Result)
			}
		}
	}
}

func TestUpdateQuickEntry_ClearOptionalFieldWithEmptyString(t *testing.T) {
	svc := newQuickService(t)
	item, _ := svc.AddQuickEntry("2026-08-03", CreateQuickEntryInput{
		Activity: "x", StartTime: "09:00", EndTime: "10:00", Quadrant: 1,
		Content: "原内容",
	})

	emptyStr := ""
	err := svc.UpdateQuickEntry("2026-08-03", item.ID, UpdateQuickEntryInput{
		Content: &emptyStr,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	log, _ := svc.repo.GetWorkLogByDate("2026-08-03")
	for _, it := range log.Items {
		if it.ID == item.ID {
			if it.Content != "" {
				t.Fatalf("expected content cleared, got %q", it.Content)
			}
		}
	}
}
