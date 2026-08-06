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
	// Task 3 实现
	return nil
}
