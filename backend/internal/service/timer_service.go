package service

import (
	"ticktask/internal/model"
	"ticktask/internal/repository"
	"ticktask/internal/websocket"
	"ticktask/pkg/logger"
	"time"

	"github.com/google/uuid"
)

type TimerService struct {
	sessionRepo  repository.SessionRepository
	taskRepo     repository.TaskRepository
	analytics    repository.AnalyticsRepository
	settingRepo  repository.SettingRepository
	wsHub        *websocket.Hub
	currentTimer *runningTimer
}

type runningTimer struct {
	session      *model.PomodoroSession
	ticker       *time.Ticker
	stopChan     chan bool
	pauseChan    chan bool
	resumeChan   chan bool
	remainingSec int
}

func NewTimerService(
	sessionRepo repository.SessionRepository,
	taskRepo repository.TaskRepository,
	analytics repository.AnalyticsRepository,
	settingRepo repository.SettingRepository,
	wsHub *websocket.Hub,
) *TimerService {
	return &TimerService{
		sessionRepo: sessionRepo,
		taskRepo:    taskRepo,
		analytics:   analytics,
		settingRepo: settingRepo,
		wsHub:       wsHub,
	}
}

type CreateSessionRequest struct {
	TaskID   *string            `json:"task_id"`
	Type     model.SessionType  `json:"type"`
	Duration int                `json:"duration"` // 秒
}

type ControlSessionRequest struct {
	Action string `json:"action" binding:"required"` // pause, resume, complete, abandon
}

// StartSession 创建并启动新会话
func (s *TimerService) StartSession(req CreateSessionRequest) (*model.PomodoroSession, error) {
	// 停止当前运行的会话
	if s.currentTimer != nil && s.currentTimer.session.Status == model.SessionRunning {
		s.stopTimer()
	}

	settings, _ := s.settingRepo.GetPomodoroSettings()
	duration := req.Duration
	if duration == 0 {
		// 使用默认时长
		switch req.Type {
		case model.SessionWork:
			duration = settings.WorkDuration
		case model.SessionShortBreak:
			duration = settings.ShortBreakDuration
		case model.SessionLongBreak:
			duration = settings.LongBreakDuration
		}
	}

	session := &model.PomodoroSession{
		ID:              uuid.New().String(),
		TaskID:          req.TaskID,
		Type:            req.Type,
		Status:          model.SessionRunning,
		StartTime:       time.Now(),
		PlannedDuration: duration,
		Interruptions:   0,
		CreatedAt:       time.Now(),
	}

	if err := s.sessionRepo.Create(session); err != nil {
		return nil, err
	}

	// 启动计时器
	s.startTimer(session, duration)

	// 通知前端连接
	s.wsHub.BroadcastTimerTick(session.ID, duration, duration, 100)

	return session, nil
}

// StartTimer 启动计时器
func (s *TimerService) startTimer(session *model.PomodoroSession, duration int) {
	s.startTimerWithRemaining(session, duration)
}

// startTimerWithRemaining 使用指定的剩余时间启动计时器
func (s *TimerService) startTimerWithRemaining(session *model.PomodoroSession, remainingSec int) {
	ticker := time.NewTicker(1 * time.Second)
	stopChan := make(chan bool)
	pauseChan := make(chan bool)
	resumeChan := make(chan bool)

	totalDuration := int(session.PlannedDuration)

	s.currentTimer = &runningTimer{
		session:      session,
		ticker:       ticker,
		remainingSec: remainingSec,
		stopChan:     stopChan,
		pauseChan:    pauseChan,
		resumeChan:   resumeChan,
	}

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if s.currentTimer != nil {
					s.currentTimer.remainingSec--
					percentage := int(float64(s.currentTimer.remainingSec) / float64(totalDuration) * 100)

					// 广播进度
					s.wsHub.BroadcastTimerTick(
						session.ID,
						s.currentTimer.remainingSec,
						totalDuration,
						percentage,
					)

					if s.currentTimer.remainingSec <= 0 {
						s.completeSessionInternal(session)
						return
					}
				}

			case <-stopChan:
				return

			case <-pauseChan:
				if s.currentTimer != nil {
					s.currentTimer.session.Status = model.SessionPaused
					s.sessionRepo.Update(s.currentTimer.session)
					s.wsHub.BroadcastSessionState(session.ID, string(model.SessionPaused))
				}
				return

			case <-resumeChan:
				// 从 pause 恢复后继续循环
			}
		}
	}()
}

// PauseSession 暂停会话
func (s *TimerService) PauseSession(sessionID string) error {
	if s.currentTimer == nil || s.currentTimer.session.ID != sessionID {
		return nil
	}

	s.currentTimer.pauseChan <- true
	s.currentTimer.ticker.Stop()
	return nil
}

// ResumeSession 继续会话
func (s *TimerService) ResumeSession(sessionID string) error {
	session, err := s.sessionRepo.GetByID(sessionID)
	if err != nil {
		return err
	}

	if session.Status != model.SessionPaused {
		return nil
	}

	session.Status = model.SessionRunning
	s.sessionRepo.Update(session)

	// 计算剩余时间：从暂停时刻的剩余时间继续
	// 由于暂停时 goroutine 已退出，我们需要基于开始时间和已运行时间计算
	// 但更准确的方式是保存暂停时的剩余时间
	// 这里简化处理：使用当前 timer 的 remainingSec（如果存在）
	var remainingSec int
	if s.currentTimer != nil && s.currentTimer.session.ID == sessionID {
		remainingSec = s.currentTimer.remainingSec
	} else {
		// 如果 currentTimer 已丢失，基于开始时间估算（不准确，但总比从头开始好）
		elapsed := int(time.Since(session.StartTime).Seconds())
		remainingSec = int(session.PlannedDuration) - elapsed
		if remainingSec < 0 {
			remainingSec = 0
		}
	}

	// 启动新的计时器，使用剩余时间
	s.startTimerWithRemaining(session, remainingSec)
	s.wsHub.BroadcastSessionState(session.ID, string(model.SessionRunning))

	return nil
}

// CompleteSession 完成会话
func (s *TimerService) CompleteSession(sessionID string) error {
	s.stopTimer()

	session, err := s.sessionRepo.GetByID(sessionID)
	if err != nil {
		return err
	}

	s.completeSessionInternal(session)
	return nil
}

// AbandonSession 放弃会话
func (s *TimerService) AbandonSession(sessionID string) error {
	s.stopTimer()

	session, err := s.sessionRepo.GetByID(sessionID)
	if err != nil {
		return err
	}

	now := time.Now()
	session.Status = model.SessionAbandoned
	session.EndTime = &now
	s.sessionRepo.Update(session)

	s.wsHub.BroadcastSessionState(session.ID, string(model.SessionAbandoned))
	return nil
}

// GetActiveSession 获取当前活跃会话
func (s *TimerService) GetActiveSession() (*model.PomodoroSession, error) {
	return s.sessionRepo.GetActive()
}

// GetRecentSessions 获取最近会话
func (s *TimerService) GetRecentSessions(limit int) ([]model.PomodoroSession, error) {
	return s.sessionRepo.GetRecent(limit)
}

// GetTodayTaskStats 获取今日各任务的投入统计
func (s *TimerService) GetTodayTaskStats() ([]TaskTimeStats, error) {
	sessions, err := s.sessionRepo.GetByDate(time.Now())
	if err != nil {
		return nil, err
	}

	// 按任务分组统计
	statsMap := make(map[string]*TaskTimeStats)
	for _, session := range sessions {
		// 只统计已完成的 work 类型会话
		if session.Type != model.SessionWork || session.Status != model.SessionCompleted {
			continue
		}

		taskID := ""
		if session.TaskID != nil {
			taskID = *session.TaskID
		}

		if _, exists := statsMap[taskID]; !exists {
			statsMap[taskID] = &TaskTimeStats{
				TaskID:      taskID,
				TaskTitle:   "",
				SessionCount: 0,
				TotalTime:   0,
			}
		}

		if session.ActualDuration != nil {
			statsMap[taskID].SessionCount++
			statsMap[taskID].TotalTime += *session.ActualDuration
		}
	}

	// 获取任务标题
	for taskID, stats := range statsMap {
		if taskID != "" {
			task, err := s.taskRepo.GetByID(taskID)
			if err == nil {
				stats.TaskTitle = task.Title
			}
		}
	}

	// 转换为切片
	result := make([]TaskTimeStats, 0, len(statsMap))
	for _, stats := range statsMap {
		result = append(result, *stats)
	}

	return result, nil
}

// TaskTimeStats 任务时间统计
type TaskTimeStats struct {
	TaskID       string `json:"task_id"`
	TaskTitle    string `json:"task_title"`
	SessionCount int    `json:"session_count"`
	TotalTime    int    `json:"total_time"` // 秒
}

// ControlSession 控制会话（统一接口）
func (s *TimerService) ControlSession(sessionID string, action string) error {
	switch action {
	case "pause":
		return s.PauseSession(sessionID)
	case "resume":
		return s.ResumeSession(sessionID)
	case "complete":
		return s.CompleteSession(sessionID)
	case "abandon":
		return s.AbandonSession(sessionID)
	default:
		logger.Logger.Error("unknown action", "action", action)
		return nil
	}
}

// 内部方法：停止计时器
func (s *TimerService) stopTimer() {
	if s.currentTimer != nil {
		close(s.currentTimer.stopChan)
		if s.currentTimer.ticker != nil {
			s.currentTimer.ticker.Stop()
		}
		s.currentTimer = nil
	}
}

// 内部方法：完成会话
func (s *TimerService) completeSessionInternal(session *model.PomodoroSession) {
	now := time.Now()
	actualDuration := int(now.Sub(session.StartTime).Seconds())

	session.Status = model.SessionCompleted
	session.EndTime = &now
	session.ActualDuration = &actualDuration
	s.sessionRepo.Update(session)

	// 更新统计
	if session.Type == model.SessionWork {
		today := time.Now().Truncate(24 * time.Hour)
		s.analytics.IncrementCompletedPomodoros(today)
		s.analytics.IncrementFocusTime(today, actualDuration)
	}

	// 广播完成状态
	s.wsHub.BroadcastSessionState(session.ID, string(model.SessionCompleted))
	s.wsHub.BroadcastTimerComplete(session.ID)

	logger.Logger.Info("session completed", "id", session.ID, "type", session.Type, "duration", actualDuration)
}
