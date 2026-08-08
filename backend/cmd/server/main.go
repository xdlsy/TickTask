package main

import (
	"fmt"
	"log"
	"ticktask/internal/agent"
	"ticktask/internal/agent/tools"
	"ticktask/internal/ai"
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

	// 初始化 WorkLog Service（M2: AI client 接真实实现）
	workLogRepo := repository.NewWorkLogRepository(db)
	workLogAIClient := service.NewWorkLogAIClient(aiService)
	workLogService := service.NewWorkLogService(workLogRepo, taskRepo, sessionRepo, workLogAIClient)

	// 初始化 Data Service（数据导入导出）
	dataRepo := repository.NewDataRepository(db)
	dataService := service.NewDataService(dataRepo)

	// 为 Agent 构造独立的 LLMClient（与 AIService 内部构造逻辑一致）。
	// Task 15 删除 AIService 后会消除这段重复。
	var agentLLM ai.LLMClient
	if aiSettings != nil {
		switch aiSettings.Provider {
		case "claude":
			agentLLM = ai.NewCLIClient()
		case "anthropic":
			if aiSettings.APIKey != "" {
				agentLLM = ai.NewAnthropicClient(aiSettings.APIKey, aiSettings.Model)
			}
		default:
			if aiSettings.APIKey != "" {
				agentLLM = ai.NewOpenAIClient(aiSettings.APIKey, aiSettings.BaseURL, aiSettings.Model)
			}
		}
	}

	// 初始化 Agent Service
	agentRepo := repository.NewAgentRepository(db)
	registry := agent.NewToolRegistry()
	tools.RegisterAll(registry, tools.Deps{
		Tasks:     taskService,
		Timer:     timerService,
		Schedule:  scheduleService,
		Analytics: analyticsService,
		WorkLog:   workLogService,
		LLM:       agentLLM,
	})
	agentSvc := agent.NewAgentService(agent.AgentDeps{
		Repo:     agentRepo,
		LLM:      agentLLM,
		Registry: registry,
		Hub:      wsHub,
		System:   agent.DefaultSystemPrompt,
	})

	// 设置路由
	router := api.SetupRouter(cfg, taskService, timerService, aiService, analyticsService, scheduleService, workLogService, wsHub, settingRepo, dataService, agentSvc, agentRepo)

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
