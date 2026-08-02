package database

import (
	"encoding/json"
	"ticktask/internal/model"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Init 初始化数据库连接
func Init(path string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	// 自动迁移表结构
	if err := AutoMigrate(db); err != nil {
		return nil, err
	}

	// 一次性数据迁移：旧 manual items 的 title 从 activity 回填
	if err := MigrateWorkItemsTitleBackfill(db); err != nil {
		return nil, err
	}

	return db, nil
}

// AutoMigrate 自动迁移表结构
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.Task{},
		&model.PomodoroSession{},
		&model.Setting{},
		&model.DailyStats{},
		&model.Schedule{},
		&model.WorkLog{},
		&model.WorkItem{},
		&model.WorkReport{},
	)
}

// SeedInitialData 插入初始数据
func SeedInitialData(db *gorm.DB) error {
	// 检查是否已存在设置
	var count int64
	if err := db.Model(&model.Setting{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil // 已有数据，跳过
	}

	// 插入默认番茄设置
	pomoSettings := model.DefaultPomodoroSettings()
	pomoJSON, _ := marshalSettings(pomoSettings)
	db.Create(&model.Setting{
		Key:   "pomodoro.settings",
		Value: string(pomoJSON),
	})

	// 插入默认 AI 设置
	aiSettings := model.DefaultAISettings()
	aiJSON, _ := marshalSettings(aiSettings)
	db.Create(&model.Setting{
		Key:   "ai.settings",
		Value: string(aiJSON),
	})

	// 插入今日统计
	today := time.Now().Truncate(24 * time.Hour)
	db.Create(&model.DailyStats{
		Date:                today,
		CompletedPomodoros:  0,
		TotalFocusTime:      0,
		CompletedTasks:      0,
		CreatedTasks:        0,
	})

	return nil
}

func marshalSettings(v any) ([]byte, error) {
	return json.Marshal(v)
}
