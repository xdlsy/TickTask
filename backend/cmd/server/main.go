package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
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
	"ticktask/pkg/vault"
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

	// Initialize the .keyvault BEFORE opening the DB: we need the vault ready
	// to construct the encrypted settingRepo, and the migration runs against
	// a freshly-opened DB right after seed.
	vaultPath := filepath.Join(filepath.Dir(cfg.Database.Path), ".keyvault")
	v, err := vault.New(vaultPath)
	if err != nil {
		log.Fatalf("vault load: %v", err)
	}
	if err := v.Init(); err != nil {
		log.Fatalf("vault init: %v", err)
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
	settingRepo := repository.NewSettingRepository(db, v)
	analyticsRepo := repository.NewAnalyticsRepository(db)
	scheduleRepo := repository.NewScheduleRepository(db)

	// One-time legacy api_key migration. Idempotent — runs once on first
	// post-upgrade startup, becomes a no-op afterwards.
	if err := settingRepo.MigrateLegacyAPIKey(); err != nil {
		log.Fatalf("legacy api_key migration: %v", err)
	}

	// Safety net: honor TT_AI_API_KEY one last time. If env is set AND the DB
	// has no encrypted key yet (fresh install, user just upgraded), persist
	// the env key into encrypted storage and warn the user to remove the env.
	if envKey := os.Getenv("TT_AI_API_KEY"); envKey != "" {
		envAI, _ := settingRepo.GetAISettings()
		if envAI != nil && envAI.APIKey == "" {
			logger.Logger.Warn("TT_AI_API_KEY env detected; migrating to encrypted storage. Remove this env var from your shell/scripts.")
			envAI.APIKey = envKey
			if err := settingRepo.UpdateAISettings(envAI); err != nil {
				logger.Logger.Warn("env api_key migration failed", "err", err)
			}
		}
	}

	// 初始化 WebSocket Hub
	wsHub := websocket.NewHub()

	// 初始化 Service
	taskService := service.NewTaskService(taskRepo, analyticsRepo, settingRepo, sessionRepo)
	timerService := service.NewTimerService(sessionRepo, taskRepo, analyticsRepo, settingRepo, wsHub)
	analyticsService := service.NewAnalyticsService(analyticsRepo, taskRepo, sessionRepo, settingRepo)

	// 构造共享的 LLMClient：Schedule / WorkLog 两处服务只调用 ChatCompletion，
	// 启动期一次性构造即可（用户极少在运行中切换 provider）。
	// Task 4 把构造逻辑下沉到 ai.NewClientFromSettings，此处直接复用。
	aiSettings, err := settingRepo.GetAISettings()
	if err != nil {
		aiSettings = nil
	}
	llm := ai.NewClientFromSettings(aiSettings)

	// llmFactory 在 agent 每一轮调用时按当前 settings 重新构造 LLM 客户端，
	// 这样用户在 Settings 页切换 provider / 修改 API key 后无需重启即可生效
	// （Task 22 配置热重载）。当 settings 缺失或 provider 不变时构造函数本身
	// 是廉价的对象分配，无需缓存。
	llmFactory := func() ai.LLMClient {
		current, err := settingRepo.GetAISettings()
		if err != nil || current == nil {
			return llm // 回落到启动期构造的客户端
		}
		return ai.NewClientFromSettings(current)
	}

	// 初始化 Schedule Service
	scheduleService := service.NewScheduleService(scheduleRepo, taskRepo, llm, settingRepo, wsHub)

	// 初始化 WorkLog Service
	workLogRepo := repository.NewWorkLogRepository(db)
	workLogService := service.NewWorkLogService(workLogRepo, taskRepo, sessionRepo, llm)

	// 初始化 Data Service（数据导入导出）
	dataRepo := repository.NewDataRepository(db)
	dataService := service.NewDataService(dataRepo, settingRepo)

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
		Settings:  settingRepo,
	})
	agentSvc := agent.NewAgentService(agent.AgentDeps{
		Repo:         agentRepo,
		LLMFactory:   llmFactory,
		SettingsRepo: settingRepo,
		Registry:     registry,
		Hub:          wsHub,
		System:       agent.DefaultSystemPrompt,
		Tracer:       agent.SelectTracerFromEnv(os.Getenv),
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
