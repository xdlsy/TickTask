package model

type Setting struct {
	Key   string `gorm:"primaryKey;size:100" json:"key"`
	Value string `gorm:"type:text" json:"value"`
}

// TableName 指定表名
func (Setting) TableName() string {
	return "settings"
}

type PomodoroSettings struct {
	WorkDuration       int  `json:"work_duration"`         // 秒，默认 25*60
	ShortBreakDuration int  `json:"short_break_duration"` // 秒，默认 5*60
	LongBreakDuration  int  `json:"long_break_duration"`  // 秒，默认 15*60
	LongBreakAfter     int  `json:"long_break_after"`     // 多个番茄后长休息，默认 4
	AutoStartBreak     bool `json:"auto_start_break"`
	AutoStartWork      bool `json:"auto_start_work"`
	EnableSound        bool `json:"enable_sound"`
}

func DefaultPomodoroSettings() *PomodoroSettings {
	return &PomodoroSettings{
		WorkDuration:       25 * 60,
		ShortBreakDuration: 5 * 60,
		LongBreakDuration:  15 * 60,
		LongBreakAfter:     4,
		AutoStartBreak:     false,
		AutoStartWork:      false,
		EnableSound:        true,
	}
}

type AISettings struct {
	Provider string `json:"provider"` // openai, anthropic, custom
	APIKey   string `json:"api_key"`
	BaseURL  string `json:"base_url"`
	Model    string `json:"model"`
}

func DefaultAISettings() *AISettings {
	return &AISettings{
		Provider: "openai",
		APIKey:   "",
		BaseURL:  "https://api.openai.com/v1",
		Model:    "gpt-4o-mini",
	}
}
