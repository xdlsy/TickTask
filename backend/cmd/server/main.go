package main

import (
	"fmt"
	"log"
	"ticktask/internal/agent"
	"ticktask/internal/agent/tools"
	"ticktask/internal/ai"
	"ticktask/internal/api"
	"ticktask/internal/model"
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

	// 构造共享的 LLMClient：Schedule / WorkLog 两处服务只调用 ChatCompletion，
	// 启动期一次性构造即可（用户极少在运行中切换 provider）。
	// 之前这段逻辑散落在 AIService 内部 + main 里的 agentLLM 重复块（Task 14 引入），
	// Task 15 删除 AIService 后此处成为唯一构造点。
	aiSettings, err := settingRepo.GetAISettings()
	if err != nil {
		aiSettings = nil
	}
	llm := constructLLMClient(aiSettings)

	// llmFactory 在 agent 每一轮调用时按当前 settings 重新构造 LLM 客户端，
	// 这样用户在 Settings 页切换 provider / 修改 API key 后无需重启即可生效
	// （Task 22 配置热重载）。当 settings 缺失或 provider 不变时构造函数本身
	// 是廉价的对象分配，无需缓存。
	llmFactory := func() ai.LLMClient {
		current, err := settingRepo.GetAISettings()
		if err != nil || current == nil {
			return llm // 回落到启动期构造的客户端
		}
		return constructLLMClient(current)
	}

	// 初始化 Schedule Service
	scheduleService := service.NewScheduleService(scheduleRepo, taskRepo, llm, settingRepo, wsHub)

	// 初始化 WorkLog Service
	workLogRepo := repository.NewWorkLogRepository(db)
	workLogService := service.NewWorkLogService(workLogRepo, taskRepo, sessionRepo, llm)

	// 初始化 Data Service（数据导入导出）
	dataRepo := repository.NewDataRepository(db)
	dataService := service.NewDataService(dataRepo)

	// 初始化 Agent Service
	agentRepo := repository.NewAgentRepository(db)
	registry := agent.NewToolRegistry()
	tools.RegisterAll(registry, tools.Deps{
		Tasks:     taskService,
		Timer:     timerService,
		Schedule:  scheduleService,
		Analytics: analyticsService,
		WorkLog:   workLogService,
		LLM:       llm,
	})
	agentSvc := agent.NewAgentService(agent.AgentDeps{
		Repo:         agentRepo,
		LLMFactory:   llmFactory,
		SettingsRepo: settingRepo,
		Registry:     registry,
		Hub:          wsHub,
		System:       agent.DefaultSystemPrompt,
	})

	// 设置路由
	router := api.SetupRouter(cfg, taskService, timerService, analyticsService, scheduleService, workLogService, wsHub, settingRepo, dataService, agentSvc, agentRepo)

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

// constructLLMClient 根据 AISettings 构造一个 LLMClient 实例。
// 提取自启动期的内联 switch：Schedule / WorkLog 仍按启动期配置取一次，
// agent 服务则通过 llmFactory 每轮重新构造（热重载）。
// nil settings 返回 nil client，调用方需自行处理（Schedule/WorkLog 仍可
// 接受 nil 路径，但触发 AI 调用时会返回 "API key not configured" 错误）。
func constructLLMClient(settings *model.AISettings) ai.LLMClient {
	if settings == nil {
		return nil
	}
	switch settings.Provider {
	case "claude", "cli":
		return ai.NewCLIClient()
	case "anthropic":
		if settings.APIKey == "" {
			return nil
		}
		return ai.NewAnthropicClient(settings.APIKey, settings.BaseURL, settings.Model)
	default: // openai / custom / 兼容 OpenAI 协议
		if settings.APIKey == "" {
			return nil
		}
		return ai.NewOpenAIClient(settings.APIKey, settings.BaseURL, settings.Model)
	}
}
