package api

import (
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"ticktask/internal/agent"
	"ticktask/internal/api/handler"
	"ticktask/internal/api/middleware"
	"ticktask/internal/repository"
	"ticktask/internal/service"
	"ticktask/internal/websocket"
	"ticktask/pkg/config"
	"ticktask/web"

	"github.com/gin-gonic/gin"
)

// SetupRouter 设置路由
func SetupRouter(
	cfg *config.Config,
	taskService *service.TaskService,
	timerService *service.TimerService,
	analyticsService *service.AnalyticsService,
	scheduleService *service.ScheduleService,
	workLogService *service.WorkLogService,
	wsHub *websocket.Hub,
	settingRepo repository.SettingRepository,
	dataService service.DataService,
	agentSvc agent.AgentService,
	agentRepo repository.AgentRepository,
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

		// 设置
		settings := api.Group("/settings")
		{
			settingHandler := handler.NewSettingHandler(settingRepo)
			settings.GET("", settingHandler.GetSettings)
			settings.PUT("/pomodoro", settingHandler.UpdatePomodoroSettings)
			settings.PUT("/ai", settingHandler.UpdateAISettings)
		}

		// 数据导入导出
		data := api.Group("/data")
		{
			dataHandler := handler.NewDataHandler(dataService)
			data.GET("/export", dataHandler.Export)
			data.POST("/import/preview", dataHandler.PreviewImport)
			data.POST("/import/apply", dataHandler.ApplyImport)
			data.DELETE("/all", dataHandler.ClearAll)
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
			workLogs.PATCH("/:date/summary", wlHandler.UpdateSummary)
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

		// Agent
		agentGroup := api.Group("/agent")
		{
			handler.NewAgentHandler(agentSvc, agentRepo).Register(agentGroup)
		}
	}

	// 静态文件服务：磁盘 dist 优先（仓库开发布局），嵌入式前端兜底（打包 exe）
	serveFrontend(r)

	return r
}

// serveFrontend 注册前端静态资源服务。
// 顺序：磁盘 dist（dist / ../frontend/dist / ../../frontend/dist）→
// 嵌入的真实前端 → 嵌入占位页。API 前缀的未匹配路由一律 404 JSON。
func serveFrontend(r *gin.Engine) {
	if diskPath := web.FindDiskDist(); diskPath != "" {
		r.Static("/assets", filepath.Join(diskPath, "assets"))
		r.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/api") {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.File(filepath.Join(diskPath, "index.html"))
		})
		return
	}

	dist := web.DistFS()
	if !web.IsStub() {
		if assets, err := fs.Sub(dist, "assets"); err == nil {
			r.StaticFS("/assets", http.FS(assets))
		}
	}
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		// 注：传 "/" 而非 "index.html"。net/http 的 serveFile 会对后缀为 "/index.html"
		// 的请求强制 301 跳转到 "./"，导致 NoRoute 返回 301 而非 200。传 "/" 时 FileServer
		// 直接服务根目录的 index.html（Vite 构建产物与占位页均符合此布局）。
		c.FileFromFS("/", http.FS(dist))
	})
}
