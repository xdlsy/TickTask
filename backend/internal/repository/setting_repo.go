package repository

import (
	"encoding/json"
	"ticktask/internal/model"

	"gorm.io/gorm"
)

type SettingRepository interface {
	Get(key string) (*model.Setting, error)
	Set(key, value string) error
	GetPomodoroSettings() (*model.PomodoroSettings, error)
	UpdatePomodoroSettings(settings *model.PomodoroSettings) error
	GetAISettings() (*model.AISettings, error)
	UpdateAISettings(settings *model.AISettings) error
}

type settingRepository struct {
	db *gorm.DB
}

func NewSettingRepository(db *gorm.DB) SettingRepository {
	return &settingRepository{db: db}
}

func (r *settingRepository) Get(key string) (*model.Setting, error) {
	var setting model.Setting
	err := r.db.Where("key = ?", key).First(&setting).Error
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

func (r *settingRepository) Set(key, value string) error {
	return r.db.Save(&model.Setting{Key: key, Value: value}).Error
}

func (r *settingRepository) GetPomodoroSettings() (*model.PomodoroSettings, error) {
	settings := model.DefaultPomodoroSettings()

	setting, err := r.Get("pomodoro.settings")
	if err != nil {
		return settings, nil
	}

	if err := json.Unmarshal([]byte(setting.Value), settings); err != nil {
		return model.DefaultPomodoroSettings(), nil
	}
	return settings, nil
}

func (r *settingRepository) UpdatePomodoroSettings(settings *model.PomodoroSettings) error {
	data, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	return r.Set("pomodoro.settings", string(data))
}

func (r *settingRepository) GetAISettings() (*model.AISettings, error) {
	setting, err := r.Get("ai.settings")
	if err != nil {
		return model.DefaultAISettings(), nil
	}

	var settings model.AISettings
	if err := json.Unmarshal([]byte(setting.Value), &settings); err != nil {
		return model.DefaultAISettings(), nil
	}
	return &settings, nil
}

func (r *settingRepository) UpdateAISettings(settings *model.AISettings) error {
	data, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	return r.Set("ai.settings", string(data))
}