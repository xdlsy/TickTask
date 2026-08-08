package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"ticktask/internal/model"
	"ticktask/internal/repository"
	"ticktask/pkg/logger"
	"time"
)

const backupApp = "ticktask"

// ErrInvalidPolicy 标识 ApplyImport 收到了非法的 policy 枚举值。handler 据此返回 400。
var ErrInvalidPolicy = errors.New("invalid policy")

type DataService interface {
	Export(includeAPIKey bool) (*model.BackupEnvelope, error)
	PreviewImport(file *model.BackupData, fileVersion int) (*model.ImportPreview, error)
	ApplyImport(req *model.ApplyImportRequest) (*model.ApplyResult, error)
	ClearAll() (*model.ClearResult, error)
}

type dataService struct {
	repo        repository.BackupRepository
	settingRepo repository.SettingRepository
}

// NewDataService wires the data service. settingRepo is used to run an
// immediate api_key migration after ApplyImport so legacy plaintext imports
// don't sit in the DB waiting for the next server restart. May be nil in
// tests that don't exercise ApplyImport.
func NewDataService(repo repository.BackupRepository, settingRepo repository.SettingRepository) DataService {
	return &dataService{repo: repo, settingRepo: settingRepo}
}

func (s *dataService) Export(includeAPIKey bool) (*model.BackupEnvelope, error) {
	data, err := s.repo.ReadAll()
	if err != nil {
		return nil, err
	}
	// Defense in depth: includeAPIKey is now a no-op back-compat parameter.
	// The export NEVER carries the api_key, regardless of what the client
	// requests (old frontend, hand-crafted URL, etc.). The encrypted blob
	// never leaks either — dataRepository.ReadAll unmarshals into
	// model.AISettings which has no api_key_encrypted field, so it's already
	// absent. We additionally blank APIKey here as belt-and-suspenders.
	if data.Settings.AI != nil {
		data.Settings.AI.APIKey = ""
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

var validPolicies = map[string]bool{
	model.PolicyAddNewOnly: true, model.PolicyMergeFile: true,
	model.PolicyMergeCurrent: true, model.PolicyReplace: true,
}

func (s *dataService) ApplyImport(req *model.ApplyImportRequest) (*model.ApplyResult, error) {
	// 1. 策略校验(把非法 policy 转为 error,handler 据此返回 400)
	for key, mod := range req.Modules {
		if !validPolicies[mod.Policy] {
			return nil, fmt.Errorf("%w: %q for module %q", ErrInvalidPolicy, mod.Policy, key)
		}
	}

	// 2. 读当前库,构造 plan
	cur, err := s.repo.ReadAll()
	if err != nil {
		return nil, err
	}
	plan := model.ApplyPlan{Settings: &req.Data.Settings}
	result := &model.ApplyResult{Applied: map[string]model.ModuleApplyResult{}}

	// 3. 每模块按 policy + override 计算 upsert/delete + 计数
	plan.Tasks, plan.DeleteTasks, result.Applied["tasks"] = resolveModule(
		cur.Tasks, req.Data.Tasks, req.Modules["tasks"], idOfTask)
	plan.Sessions, plan.DeleteSessions, result.Applied["sessions"] = resolveModule(
		cur.Sessions, req.Data.Sessions, req.Modules["sessions"], idOfSession)
	plan.Schedules, plan.DeleteSchedules, result.Applied["schedules"] = resolveModule(
		cur.Schedules, req.Data.Schedules, req.Modules["schedules"], idOfSchedule)
	plan.WorkReports, plan.DeleteWorkReports, result.Applied["work_reports"] = resolveModule(
		cur.WorkReports, req.Data.WorkReports, req.Modules["work_reports"], idOfWorkReport)
	plan.WorkLogs, plan.DeleteWorkLogs, result.Applied["work_logs"] = resolveModule(
		cur.WorkLogs, req.Data.WorkLogs, req.Modules["work_logs"], func(l model.WorkLog) string { return l.ID })

	// 4. 单事务执行
	if err := s.repo.Apply(plan); err != nil {
		return nil, err
	}

	// 5. 导入提交后立即迁移 legacy 明文 api_key — 否则导入的明文
	//    会一直驻留 DB 直到下次重启。失败仅 warn：api_key 处理不是
	//    导入的核心目的，启动期 migration 会在下次重启时重试。
	if s.settingRepo != nil {
		if err := s.settingRepo.MigrateLegacyAPIKey(); err != nil {
			logger.Logger.Warn("post-import api_key migration", "err", err)
		}
	}
	return result, nil
}

func (s *dataService) ClearAll() (*model.ClearResult, error) {
	return s.repo.ClearAll()
}

// resolveModule 按 policy + overrides 计算某表的 upsert/delete 集合 + 计数。
// 返回:toUpsert(新增 + 冲突解决为 file 的),toDelete(replace 下的 orphan),计数。
func resolveModule[T any](cur, file []T, mod model.ModuleApply, idOf func(T) string) (upsert []T, del []string, r model.ModuleApplyResult) {
	if !validPolicies[mod.Policy] {
		return nil, nil, model.ModuleApplyResult{}
	}
	curByID := map[string]T{}
	for _, x := range cur {
		curByID[idOf(x)] = x
	}
	fileByID := map[string]T{}
	for _, x := range file {
		fileByID[idOf(x)] = x
	}
	upsert = []T{}
	del = []string{}

	wantFile := func(id string) bool {
		if ch, ok := mod.Overrides[id]; ok {
			return ch == model.ChoiceFile
		}
		switch mod.Policy {
		case model.PolicyMergeFile, model.PolicyReplace:
			return true
		default: // add_new_only, merge_current
			return false
		}
	}

	for _, x := range file {
		id := idOf(x)
		ex, inCur := curByID[id]
		if !inCur {
			upsert = append(upsert, x) // 新增:所有策略都插入
			r.Inserted++
			continue
		}
		if jsonEqual(x, ex) {
			continue // identical:跳过
		}
		// 冲突
		if wantFile(id) {
			upsert = append(upsert, x)
			r.Updated++
		}
		// 否则保留当前,不动
	}
	if mod.Policy == model.PolicyReplace {
		for id := range curByID {
			if _, inFile := fileByID[id]; !inFile {
				del = append(del, id)
				r.Deleted++
			}
		}
	}
	return upsert, del, r
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
