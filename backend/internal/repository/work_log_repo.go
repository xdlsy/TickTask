// backend/internal/repository/work_log_repo.go
package repository

import (
	"errors"
	"ticktask/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WorkLogRepository 工作日志数据访问接口
type WorkLogRepository interface {
	// WorkLog CRUD
	CreateWorkLog(log *model.WorkLog) error
	GetWorkLogByDate(date string) (*model.WorkLog, error)
	GetWorkLogsInRange(from, to string) ([]*model.WorkLog, error)
	UpsertWorkLog(log *model.WorkLog) error // PUT 语义：存在则更新 items 全量替换，否则创建

	// WorkItem
	ReplaceItems(workLogID string, items []model.WorkItem) error

	// WorkReport
	CreateWorkReport(report *model.WorkReport) error
	UpdateWorkReport(report *model.WorkReport) error
	GetWorkReportByTypeAndPeriod(t model.WorkReportType, periodKey string) (*model.WorkReport, error)
	ListWorkReports(t model.WorkReportType) ([]*model.WorkReport, error)
}

type workLogRepository struct {
	db *gorm.DB
}

func NewWorkLogRepository(db *gorm.DB) WorkLogRepository {
	return &workLogRepository{db: db}
}

func (r *workLogRepository) CreateWorkLog(log *model.WorkLog) error {
	return r.db.Create(log).Error
}

func (r *workLogRepository) GetWorkLogByDate(date string) (*model.WorkLog, error) {
	var log model.WorkLog
	err := r.db.Preload("Items").Where("date = ?", date).First(&log).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func (r *workLogRepository) GetWorkLogsInRange(from, to string) ([]*model.WorkLog, error) {
	var logs []*model.WorkLog
	err := r.db.Preload("Items").
		Where("date BETWEEN ? AND ?", from, to).
		Order("date DESC").
		Find(&logs).Error
	if err != nil {
		return nil, err
	}
	return logs, nil
}

func (r *workLogRepository) UpsertWorkLog(log *model.WorkLog) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var existing model.WorkLog
		err := tx.Where("date = ?", log.Date).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(log).Error
		}
		if err != nil {
			return err
		}
		log.ID = existing.ID
		if err := tx.Save(log).Error; err != nil {
			return err
		}
		// items 全量替换
		if err := tx.Where("work_log_id = ?", log.ID).Delete(&model.WorkItem{}).Error; err != nil {
			return err
		}
		for i := range log.Items {
			log.Items[i].ID = uuid.New().String() // 生成新 UUID（GORM 不为 string PK 自动生成）
			log.Items[i].WorkLogID = log.ID
		}
		if len(log.Items) > 0 {
			return tx.Create(&log.Items).Error
		}
		return nil
	})
}

func (r *workLogRepository) ReplaceItems(workLogID string, items []model.WorkItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("work_log_id = ?", workLogID).Delete(&model.WorkItem{}).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].WorkLogID = workLogID
		}
		if len(items) > 0 {
			return tx.Create(&items).Error
		}
		return nil
	})
}

func (r *workLogRepository) CreateWorkReport(report *model.WorkReport) error {
	return r.db.Create(report).Error
}

func (r *workLogRepository) UpdateWorkReport(report *model.WorkReport) error {
	return r.db.Save(report).Error
}

func (r *workLogRepository) GetWorkReportByTypeAndPeriod(t model.WorkReportType, periodKey string) (*model.WorkReport, error) {
	var report model.WorkReport
	err := r.db.Where("type = ? AND period_key = ?", t, periodKey).First(&report).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &report, nil
}

func (r *workLogRepository) ListWorkReports(t model.WorkReportType) ([]*model.WorkReport, error) {
	var reports []*model.WorkReport
	err := r.db.Where("type = ?", t).Order("period_key DESC").Find(&reports).Error
	if err != nil {
		return nil, err
	}
	return reports, nil
}
