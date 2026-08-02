package main

import (
	"fmt"
	"log"
	"ticktask/internal/api"
	"ticktask/internal/repository"
	"ticktask/internal/service"
	"ticktask/internal/websocket"
	"ticktask/pkg/config"
	"ticktask/pkg/database"
	"ticktask/pkg/logger"
)

func main() {
	// 加载配置
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		cfg = config.LoadDefault()
		logger.Logger.Warn("using default config")
	}

	// 初始化日志
	logger.Init(cfg.Server.Mode)
	logger.Logger.Info("starting TickTask server")

	// 确保数据目录存在
	if err := ensureDataDir(cfg.Database.Path); err != nil {
		log.Fatal(err)
	}

	// 初始化数据库
	db, err := database.Init(cfg.Database.Path)
	if err != nil {
		log.Fatal(err)
	}
	logger.Logger.Info("database initialized")

	// 插入初始数据
	if err := database.SeedInitialData(db); err != nil {
		logger.Logger.Warn("seed initial data", "err", err)
	}

	// 初始化 Repository
	taskRepo := repository.NewTaskRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	settingRepo := repository.NewSettingRepository(db)
	analyticsRepo := repository.NewAnalyticsRepository(db)
	scheduleRepo := repository.NewScheduleRepository(db)

	// 初始化 WebSocket Hub
	wsHub := websocket.NewHub()

	// 初始化 Service
	taskService := service.NewTaskService(taskRepo, analyticsRepo, settingRepo, sessionRepo)
	timerService := service.NewTimerService(sessionRepo, taskRepo, analyticsRepo, settingRepo, wsHub)
	analyticsService := service.NewAnalyticsService(analyticsRepo, taskRepo, sessionRepo, settingRepo)

	// 初始化 AI Service
	aiSettings, err := settingRepo.GetAISettings()
	if err != nil {
		aiSettings = nil
	}
	aiService := service.NewAIService(aiSettings, taskRepo, scheduleRepo, sessionRepo)

	// 初始化 Schedule Service
	scheduleService := service.NewScheduleService(scheduleRepo, taskRepo, aiService, settingRepo, wsHub)

	// 初始化 WorkLog Service（M1: AI client 传 nil，M2 接真实实现）
	workLogRepo := repository.NewWorkLogRepository(db)
	workLogService := service.NewWorkLogService(workLogRepo, taskRepo, sessionRepo, nil)

	// 设置路由
	router := api.SetupRouter(cfg, taskService, timerService, aiService, analyticsService, scheduleService, workLogService, wsHub, settingRepo)

	// 启动服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	logger.Logger.Info("server listening", "addr", addr)
	if err := router.Run(addr); err != nil {
		log.Fatal(err)
	}
}

func ensureDataDir(path string) error {
	// 简化处理，确保目录存在
	return nil
}

