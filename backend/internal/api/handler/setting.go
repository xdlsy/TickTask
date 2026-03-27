package handler

import (
	"net/http"
	"ticktask/internal/model"
	"ticktask/internal/repository"

	"github.com/gin-gonic/gin"
)

type SettingHandler struct {
	settingRepo repository.SettingRepository
}

func NewSettingHandler(settingRepo repository.SettingRepository) *SettingHandler {
	return &SettingHandler{
		settingRepo: settingRepo,
	}
}

// GetSettings 获取所有设置
func (h *SettingHandler) GetSettings(c *gin.Context) {
	pomodoroSettings, err := h.settingRepo.GetPomodoroSettings()
	if err != nil {
		pomodoroSettings = model.DefaultPomodoroSettings()
	}

	aiSettings, err := h.settingRepo.GetAISettings()
	if err != nil {
		aiSettings = model.DefaultAISettings()
	}

	// 隐藏 API Key 的部分内容
	if aiSettings.APIKey != "" {
		if len(aiSettings.APIKey) > 8 {
			aiSettings.APIKey = aiSettings.APIKey[:4] + "****" + aiSettings.APIKey[len(aiSettings.APIKey)-4:]
		} else {
			aiSettings.APIKey = "****"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"pomodoro": pomodoroSettings,
		"ai":       aiSettings,
	})
}

// UpdatePomodoroSettings 更新番茄设置
func (h *SettingHandler) UpdatePomodoroSettings(c *gin.Context) {
	var settings model.PomodoroSettings

	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.settingRepo.UpdatePomodoroSettings(&settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Settings updated"})
}

// UpdateAISettings 更新 AI 设置
func (h *SettingHandler) UpdateAISettings(c *gin.Context) {
	var settings model.AISettings

	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.settingRepo.UpdateAISettings(&settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "AI settings updated"})
}