package handler

import (
	"net/http"
	"strings"
	"ticktask/internal/model"
	"ticktask/internal/repository"

	"github.com/gin-gonic/gin"
)

type SettingHandler struct {
	settingRepo repository.SettingRepository
}

func NewSettingHandler(settingRepo repository.SettingRepository) *SettingHandler {
	return &SettingHandler{settingRepo: settingRepo}
}

// GetSettings returns pomodoro + AI settings. For AI, the secret api_key is
// NEVER returned. Instead, api_key_set (bool) and api_key_preview (masked
// prefix/suffix) tell the frontend whether a key exists and what to display.
func (h *SettingHandler) GetSettings(c *gin.Context) {
	pomodoroSettings, err := h.settingRepo.GetPomodoroSettings()
	if err != nil {
		pomodoroSettings = model.DefaultPomodoroSettings()
	}

	aiSettings, err := h.settingRepo.GetAISettings()
	if err != nil {
		aiSettings = model.DefaultAISettings()
	}

	aiPayload := gin.H{
		"provider": aiSettings.Provider,
		"base_url": aiSettings.BaseURL,
		"model":    aiSettings.Model,
	}
	if aiSettings.APIKey != "" {
		aiPayload["api_key_set"] = true
		aiPayload["api_key_preview"] = maskKey(aiSettings.APIKey)
	} else {
		aiPayload["api_key_set"] = false
	}

	c.JSON(http.StatusOK, gin.H{
		"pomodoro": pomodoroSettings,
		"ai":       aiPayload,
	})
}

// maskKey returns "abcd****wxyz" form for keys > 8 chars, "****" otherwise.
// The full secret never leaves the server.
func maskKey(k string) string {
	if len(k) <= 8 {
		return "****"
	}
	return k[:4] + "****" + k[len(k)-4:]
}

// UpdatePomodoroSettings unchanged.
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

// UpdateAISettings accepts an ai settings body. The api_key field has two
// non-fallthrough semantics:
//   - empty / missing: preserve the previously stored key (handled in repo)
//   - contains "****" literal: reject as 400 — this is a mask-roundtrip bug
//     signal (frontend forwarded our own masked preview back to us)
//
// Non-empty, non-mask values are written through to the repo, which encrypts.
func (h *SettingHandler) UpdateAISettings(c *gin.Context) {
	var settings model.AISettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.Contains(settings.APIKey, "****") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "api_key looks like a masked preview — clear the field to keep the existing key, or enter a new value"})
		return
	}
	if err := h.settingRepo.UpdateAISettings(&settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "AI settings updated"})
}
