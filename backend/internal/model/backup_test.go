package model

import (
	"encoding/json"
	"testing"
)

func TestBackupEnvelope_RoundTrip(t *testing.T) {
	env := BackupEnvelope{
		App:           "ticktask",
		SchemaVersion: BackupSchemaVersion,
		Data: BackupData{
			Tasks:    []Task{{ID: "t1", Title: "x"}},
			Settings: SettingsBundle{Pomodoro: DefaultPomodoroSettings(), AI: DefaultAISettings()},
			WorkLogs: []WorkLog{{ID: "wl1", Date: "2026-08-07", Items: []WorkItem{{ID: "wi1", WorkLogID: "wl1"}}}},
		},
	}
	raw, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got BackupEnvelope
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.App != "ticktask" || got.SchemaVersion != 1 {
		t.Errorf("envelope meta wrong: %+v", got)
	}
	if len(got.Data.Tasks) != 1 || got.Data.Tasks[0].ID != "t1" {
		t.Errorf("tasks lost: %+v", got.Data.Tasks)
	}
	if got.Data.Settings.Pomodoro == nil || got.Data.WorkLogs[0].Items[0].ID != "wi1" {
		t.Errorf("settings/items nesting wrong: %+v", got.Data)
	}
}

func TestApplyPlan_ZeroValue(t *testing.T) {
	var p ApplyPlan
	raw, _ := json.Marshal(p)
	if string(raw) == "" {
		t.Error("zero ApplyPlan should marshal")
	}
}
