package tools

import (
	"context"
	"encoding/json"

	"ticktask/internal/agent"
	"ticktask/internal/model"
)

// SettingsReader is the subset of repository.SettingRepository methods the
// settings tool needs. The production repository.SettingRepository satisfies
// this implicitly (it has GetPomodoroSettings); main.go wires the concrete
// settingRepo into Deps.Settings.
type SettingsReader interface {
	GetPomodoroSettings() (*model.PomodoroSettings, error)
}

// GetSettingsTool returns the user's pomodoro settings (durations, breaks,
// automation toggles). PermRead. (AI settings are deliberately NOT exposed.)
type GetSettingsTool struct{ Svc SettingsReader }

func (t *GetSettingsTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "get_settings",
		Function: agent.FunctionSpec{
			Name:        "get_settings",
			Description: "Read the user's pomodoro settings (work/break durations, long-break cadence, auto-start toggles, sound).",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		Permission: agent.PermRead,
	}
}

func (t *GetSettingsTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	return t.Svc.GetPomodoroSettings()
}

func (t *GetSettingsTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	return t.Execute(ctx, args)
}
