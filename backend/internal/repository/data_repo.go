package repository

import (
	"encoding/json"
	"ticktask/internal/model"

	"gorm.io/gorm"
)

// BackupRepository 横跨全表的整表读 + 单事务写。
type BackupRepository interface {
	ReadAll() (*model.BackupData, error)
	Apply(plan model.ApplyPlan) error
	ClearAll() (*model.ClearResult, error)
}

type dataRepository struct {
	db *gorm.DB
}

func NewDataRepository(db *gorm.DB) BackupRepository {
	return &dataRepository{db: db}
}

func (r *dataRepository) ReadAll() (*model.BackupData, error) {
	data := &model.BackupData{}
	if err := r.db.Find(&data.Tasks).Error; err != nil {
		return nil, err
	}
	if err := r.db.Find(&data.Sessions).Error; err != nil {
		return nil, err
	}
	if err := r.db.Find(&data.Schedules).Error; err != nil {
		return nil, err
	}
	if err := r.db.Preload("Items").Find(&data.WorkLogs).Error; err != nil {
		return nil, err
	}
	if err := r.db.Find(&data.WorkReports).Error; err != nil {
		return nil, err
	}

	pomodoro := model.DefaultPomodoroSettings()
	var pomoSetting model.Setting
	if err := r.db.Where("key = ?", "pomodoro.settings").First(&pomoSetting).Error; err == nil {
		_ = json.Unmarshal([]byte(pomoSetting.Value), pomodoro)
	}
	ai := model.DefaultAISettings()
	var aiSetting model.Setting
	if err := r.db.Where("key = ?", "ai.settings").First(&aiSetting).Error; err == nil {
		_ = json.Unmarshal([]byte(aiSetting.Value), ai)
	}
	data.Settings = model.SettingsBundle{Pomodoro: pomodoro, AI: ai}

	return data, nil
}

func (r *dataRepository) Apply(plan model.ApplyPlan) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := upsertSlice(tx, plan.Tasks); err != nil {
			return err
		}
		if err := upsertSlice(tx, plan.Sessions); err != nil {
			return err
		}
		if err := upsertSlice(tx, plan.Schedules); err != nil {
			return err
		}
		if err := upsertSlice(tx, plan.WorkReports); err != nil {
			return err
		}
		if err := applyWorkLogs(tx, plan.WorkLogs); err != nil {
			return err
		}

		if err := deleteByIDs(tx, &model.Task{}, plan.DeleteTasks); err != nil {
			return err
		}
		if err := deleteByIDs(tx, &model.PomodoroSession{}, plan.DeleteSessions); err != nil {
			return err
		}
		if err := deleteByIDs(tx, &model.Schedule{}, plan.DeleteSchedules); err != nil {
			return err
		}
		if err := deleteByIDs(tx, &model.WorkReport{}, plan.DeleteWorkReports); err != nil {
			return err
		}
		if err := deleteWorkLogOrphans(tx, plan.DeleteWorkLogs); err != nil {
			return err
		}

		if plan.Settings != nil {
			if err := writeSetting(tx, "pomodoro.settings", plan.Settings.Pomodoro); err != nil {
				return err
			}
			if err := writeSetting(tx, "ai.settings", plan.Settings.AI); err != nil {
				return err
			}
		}
		return nil
	})
}

// upsertSlice 用 Save 批量 upsert(按主键)。
func upsertSlice[T any](tx *gorm.DB, records []T) error {
	if len(records) == 0 {
		return nil
	}
	return tx.Save(&records).Error
}

func deleteByIDs(tx *gorm.DB, dest any, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return tx.Where("id IN ?", ids).Delete(dest).Error
}

// applyWorkLogs 每条 log:Save 标量 → 删旧 items → 建新 items(原子)。
func applyWorkLogs(tx *gorm.DB, logs []model.WorkLog) error {
	for i := range logs {
		log := logs[i]
		if err := tx.Save(&log).Error; err != nil {
			return err
		}
		if err := tx.Where("work_log_id = ?", log.ID).Delete(&model.WorkItem{}).Error; err != nil {
			return err
		}
		if len(log.Items) > 0 {
			if err := tx.Create(&log.Items).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func deleteWorkLogOrphans(tx *gorm.DB, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if err := tx.Where("work_log_id IN ?", ids).Delete(&model.WorkItem{}).Error; err != nil {
		return err
	}
	return tx.Where("id IN ?", ids).Delete(&model.WorkLog{}).Error
}

func writeSetting(tx *gorm.DB, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return tx.Save(&model.Setting{Key: key, Value: string(raw)}).Error
}

// ClearAll 单事务清空全部用户数据;Setting 表保留(配置不丢)。
func (r *dataRepository) ClearAll() (*model.ClearResult, error) {
	res := &model.ClearResult{}
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var err error
		// 子表先删:WorkItem 属于 WorkLog(不计入 ClearResult)。
		if _, err = clearTable(tx, &model.WorkItem{}); err != nil {
			return err
		}
		if res.WorkLogs, err = clearTable(tx, &model.WorkLog{}); err != nil {
			return err
		}
		if res.WorkReports, err = clearTable(tx, &model.WorkReport{}); err != nil {
			return err
		}
		if res.Tasks, err = clearTable(tx, &model.Task{}); err != nil {
			return err
		}
		if res.Sessions, err = clearTable(tx, &model.PomodoroSession{}); err != nil {
			return err
		}
		if res.Schedules, err = clearTable(tx, &model.Schedule{}); err != nil {
			return err
		}
		if res.DailyStats, err = clearTable(tx, &model.DailyStats{}); err != nil {
			return err
		}
		// Setting 故意不清。
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// clearTable 先计数再删全表,返回删除行数。
func clearTable(tx *gorm.DB, dest any) (int64, error) {
	var n int64
	if err := tx.Model(dest).Count(&n).Error; err != nil {
		return 0, err
	}
	if err := tx.Where("1 = 1").Delete(dest).Error; err != nil {
		return 0, err
	}
	return n, nil
}
