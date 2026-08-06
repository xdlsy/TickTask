package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"ticktask/internal/model"
	"ticktask/internal/repository"
	"time"
)

const backupApp = "ticktask"

type DataService interface {
	Export() (*model.BackupEnvelope, error)
	PreviewImport(file *model.BackupData, fileVersion int) (*model.ImportPreview, error)
	ApplyImport(req *model.ApplyImportRequest) (*model.ApplyResult, error)
}

type dataService struct {
	repo repository.BackupRepository
}

func NewDataService(repo repository.BackupRepository) DataService {
	return &dataService{repo: repo}
}

func (s *dataService) Export() (*model.BackupEnvelope, error) {
	data, err := s.repo.ReadAll()
	if err != nil {
		return nil, err
	}
	return &model.BackupEnvelope{
		App:           backupApp,
		SchemaVersion: model.BackupSchemaVersion,
		ExportedAt:    time.Now().UTC(),
		Data:          *data,
	}, nil
}

func (s *dataService) PreviewImport(file *model.BackupData, fileVersion int) (*model.ImportPreview, error) {
	cur, err := s.repo.ReadAll()
	if err != nil {
		return nil, err
	}
	warning := ""
	if fileVersion != model.BackupSchemaVersion {
		warning = fmt.Sprintf("备份 schema 版本 %d 与当前 %d 不一致,导入可能不完整", fileVersion, model.BackupSchemaVersion)
	}
	return &model.ImportPreview{
		SchemaVersion: model.BackupSchemaVersion,
		SchemaWarning: warning,
		Modules: map[string]*model.ModulePreview{
			"tasks":        classify(cur.Tasks, file.Tasks, idOfTask),
			"sessions":     classify(cur.Sessions, file.Sessions, idOfSession),
			"schedules":    classify(cur.Schedules, file.Schedules, idOfSchedule),
			"work_reports": classify(cur.WorkReports, file.WorkReports, idOfWorkReport),
			"work_logs":    classifyWorkLogs(cur.WorkLogs, file.WorkLogs),
			"settings":     diffSettings(cur.Settings, file.Settings),
		},
	}, nil
}

func (s *dataService) ApplyImport(req *model.ApplyImportRequest) (*model.ApplyResult, error) {
	return nil, nil // Task 6
}

// classify 泛型分类:按 id 把 file 记录归入 new/identical/conflict,把 cur 独有的归入 orphan。
func classify[T any](cur, file []T, idOf func(T) string) *model.ModulePreview {
	curByID := map[string]T{}
	for _, r := range cur {
		curByID[idOf(r)] = r
	}
	fileByID := map[string]T{}
	for _, r := range file {
		fileByID[idOf(r)] = r
	}

	m := &model.ModulePreview{Conflicts: []model.RecordConflict{}}
	for _, r := range file {
		id := idOf(r)
		ex, inCur := curByID[id]
		if !inCur {
			m.New++
			continue
		}
		if jsonEqual(r, ex) {
			m.Identical++
			continue
		}
		m.Conflict++
		m.Conflicts = append(m.Conflicts, model.RecordConflict{ID: id, Fields: fieldDiffs(ex, r)})
	}
	for id := range curByID {
		if _, inFile := fileByID[id]; !inFile {
			m.Orphan++
		}
	}
	return m
}

func classifyWorkLogs(cur, file []model.WorkLog) *model.ModulePreview {
	// 整条 log(含 items)参与 identical 判定 → 原子。
	return classify(cur, file, func(l model.WorkLog) string { return l.ID })
}

func diffSettings(cur, file model.SettingsBundle) *model.ModulePreview {
	m := &model.ModulePreview{SettingsConflicts: []model.SettingsFieldDiff{}}
	m.SettingsConflicts = append(m.SettingsConflicts, diffSection("pomodoro", cur.Pomodoro, file.Pomodoro)...)
	m.SettingsConflicts = append(m.SettingsConflicts, diffSection("ai", cur.AI, file.AI)...)
	return m
}

func diffSection(section string, cur, file any) []model.SettingsFieldDiff {
	if cur == nil || file == nil {
		return nil
	}
	out := []model.SettingsFieldDiff{}
	cm := toMap(cur)
	fm := toMap(file)
	for k, cv := range cm {
		fv, ok := fm[k]
		if !ok {
			continue // 文件缺该字段,跳过(不视为冲突)
		}
		if !reflect.DeepEqual(cv, fv) {
			out = append(out, model.SettingsFieldDiff{Section: section, Field: k, Current: cv, Imported: fv})
		}
	}
	return out
}

// toMap 把结构 marshal 再 unmarshal 成 map[string]any,便于逐字段比对。
func toMap(v any) map[string]any {
	raw, _ := json.Marshal(v)
	m := map[string]any{}
	_ = json.Unmarshal(raw, &m)
	return m
}

func jsonEqual(a, b any) bool {
	ra, _ := json.Marshal(a)
	rb, _ := json.Marshal(b)
	return bytes.Equal(ra, rb)
}

// fieldDiffs 对比两条同类型记录,列出值不同的字段(canonical JSON)。
func fieldDiffs(cur, file any) []model.FieldDiff {
	cm := toMap(cur)
	fm := toMap(file)
	out := []model.FieldDiff{}
	for k, cv := range cm {
		fv, ok := fm[k]
		if !ok {
			continue
		}
		if !reflect.DeepEqual(cv, fv) {
			out = append(out, model.FieldDiff{Field: k, Current: cv, Imported: fv})
		}
	}
	return out
}

// id 提取器
func idOfTask(t model.Task) string               { return t.ID }
func idOfSession(s model.PomodoroSession) string { return s.ID }
func idOfSchedule(s model.Schedule) string       { return s.ID }
func idOfWorkReport(r model.WorkReport) string   { return r.ID }
