package api

import (
	"net/http"
	"os"
	"path/filepath"
	"ticktask/internal/api/handler"
	"ticktask/internal/api/middleware"
	"ticktask/internal/repository"
	"ticktask/internal/service"
	"ticktask/internal/websocket"
	"ticktask/pkg/config"

	"github.com/gin-gonic/gin"
)

// SetupRouter 设置路由
func SetupRouter(
	cfg *config.Config,
	taskService *service.TaskService,
	timerService *service.TimerService,
	aiService *service.AIService,
	analyticsService *service.AnalyticsService,
	scheduleService *service.ScheduleService,
	workLogService *service.WorkLogService,
	wsHub *websocket.Hub,
	settingRepo repository.SettingRepository,
) *gin.Engine {
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS(cfg))

	// WebSocket
	r.GET("/ws", wsHub.WebSocketHandler)

	// API 路由组
	api := r.Group("/api")
	{
		// 任务
		tasks := api.Group("/tasks")
		{
			tasks.GET("", handler.NewTaskHandler(taskService).GetTasks)
			tasks.GET("/quadrant", handler.NewTaskHandler(taskService).GetTasksByQuadrant)
			tasks.GET("/:id", handler.NewTaskHandler(taskService).GetTask)
			tasks.POST("", handler.NewTaskHandler(taskService).CreateTask)
			tasks.PUT("/:id", handler.NewTaskHandler(taskService).UpdateTask)
			tasks.DELETE("/:id", handler.NewTaskHandler(taskService).DeleteTask)
			tasks.PATCH("/:id/move", handler.NewTaskHandler(taskService).MoveTask)
		}

		// 计时器
		sessions := api.Group("/sessions")
		{
			sessions.GET("/active", handler.NewTimerHandler(timerService).GetActiveSession)
			sessions.GET("/recent", handler.NewTimerHandler(timerService).GetRecentSessions)
			sessions.GET("/today-stats", handler.NewTimerHandler(timerService).GetTodayTaskStats)
			sessions.POST("", handler.NewTimerHandler(timerService).CreateSession)
			sessions.PATCH("/:id/control", handler.NewTimerHandler(timerService).ControlSession)
		}

		// AI 智能功能
		ai := api.Group("/ai")
		{
			aiHandler := handler.NewAIHandler(aiService, taskService)
			ai.GET("/status", aiHandler.GetAIStatus)
			ai.POST("/classify", aiHandler.ClassifyTask)
			ai.POST("/classify/batch", aiHandler.ClassifyTasks)
			ai.POST("/classify-task-text", aiHandler.ClassifyTaskByText)
			ai.POST("/schedule", aiHandler.GenerateSchedule)
			ai.POST("/reschedule-after-interrupt", aiHandler.RescheduleAfterInterrupt)
			ai.GET("/priority", aiHandler.GetPrioritySuggestions)
			ai.GET("/daily-insights", aiHandler.GetDailyInsights)
		}

		// 设置
		settings := api.Group("/settings")
		{
			settingHandler := handler.NewSettingHandler(settingRepo)
			settings.GET("", settingHandler.GetSettings)
			settings.PUT("/pomodoro", settingHandler.UpdatePomodoroSettings)
			settings.PUT("/ai", settingHandler.UpdateAISettings)
		}

		// 数据分析
		analytics := api.Group("/analytics")
		{
			analyticsHandler := handler.NewAnalyticsHandler(analyticsService)
			analytics.GET("/summary", analyticsHandler.GetSummary)
			analytics.GET("/trend", analyticsHandler.GetTrend)
			analytics.GET("/distribution", analyticsHandler.GetDistribution)
				analytics.GET("/pomodoro-by-task", analyticsHandler.GetPomodoroByTask)
				analytics.GET("/pomodoro-trends", analyticsHandler.GetPomodoroTrends)
		}

		// 日程
		schedules := api.Group("/schedules")
		{
			scheduleHandler := handler.NewScheduleHandler(scheduleService)
			schedules.GET("", scheduleHandler.GetSchedules)
				schedules.POST("/revise", scheduleHandler.ReviseWithAI)
				schedules.POST("/revise/apply", scheduleHandler.ApplyRevision)
			schedules.GET("/:id", scheduleHandler.GetSchedule)
			schedules.POST("", scheduleHandler.CreateSchedule)
			schedules.PUT("/:id", scheduleHandler.UpdateSchedule)
			schedules.DELETE("/:id", scheduleHandler.DeleteSchedule)
			schedules.PUT("/:id/move", scheduleHandler.MoveSchedule)
			schedules.POST("/generate", scheduleHandler.GenerateWithAI)
			schedules.DELETE("", scheduleHandler.DeleteAll)
		}

		// 工作日志
		workLogs := api.Group("/work-logs")
		{
			wlHandler := handler.NewWorkLogHandler(workLogService)
			workLogs.GET("/today/context", wlHandler.GetTodayContext)
			workLogs.POST("/structure", wlHandler.Structure)
			workLogs.GET("", wlHandler.ListWorkLogs)
			workLogs.POST("", wlHandler.CreateWorkLog)
			workLogs.GET("/:date", wlHandler.GetWorkLog)
			workLogs.PUT("/:date", wlHandler.UpdateWorkLog)
			// 快捷录入（今日全景）
			workLogs.POST("/:date/items", wlHandler.AddQuickEntry)
			workLogs.PATCH("/:date/items/:itemId", wlHandler.UpdateQuickEntry)
			workLogs.DELETE("/:date/items/:itemId", wlHandler.DeleteQuickEntry)
		}

		// 工作日志报告
		workReports := api.Group("/work-reports")
		{
			wrHandler := handler.NewWorkLogHandler(workLogService)
			workReports.POST("/generate", wrHandler.GenerateReport)
			workReports.GET("", wrHandler.ListReports)
			workReports.GET("/:type/:periodKey", wrHandler.GetReport)
		}
	}

	// 静态文件服务（生产模式）
	// 检查 dist 目录是否存在（相对于可执行文件或工作目录）
	distPaths := []string{
		"dist",
		"../frontend/dist",
		"../../frontend/dist",
	}

	for _, distPath := range distPaths {
		if _, err := os.Stat(distPath); err == nil {
			// 服务静态资源
			r.Static("/assets", filepath.Join(distPath, "assets"))

			// 对于所有非 API 和非 WebSocket 的请求，返回 index.html（支持 SPA 路由）
			r.NoRoute(func(c *gin.Context) {
				// 如果是 API 请求但路由不存在，返回 404
				if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
					c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
					return
				}
				// 否则返回 index.html（SPA 路由支持）
				c.File(filepath.Join(distPath, "index.html"))
			})

			break
		}
	}

	return r
}
