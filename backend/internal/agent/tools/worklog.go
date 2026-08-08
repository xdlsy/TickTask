package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"ticktask/internal/agent"
	"ticktask/internal/model"
	"ticktask/internal/service"
)

// WorkLogStructureSvc is the subset of service.WorkLogService methods that
// structure_worklog needs. The production service.WorkLogService struct
// satisfies this implicitly; tests substitute a mock that implements only
// StructureBrainDump. Defining the interface here keeps the tools package
// decoupled from the concrete WorkLogService (which owns an aiClient
// dependency that the StructureBrainDump method delegates to).
//
// Divergence from the original brief: the brief asked the tool to call
// workLogStructurePrompt + LLM.ChatCompletion directly, then JSON-parse the
// output. That flow is already wrapped by WorkLogService.StructureBrainDump
// (which calls WorkLogAIClient, which itself runs the prompt + parses), so we
// delegate to the service instead of duplicating the prompt logic. The tool
// therefore does not need an LLMClient field.
type WorkLogStructureSvc interface {
	StructureBrainDump(input service.BrainDumpInput) (*service.StructuredWorkLog, error)
}

// WorkLogSaveSvc is the subset of service.WorkLogService methods that
// save_worklog needs. The return type is *model.WorkLog because that is what
// service.WorkLogService.SaveWorkLog returns — the service DTO layer for
// work-log (BrainDumpInput / SaveWorkLogInput / StructuredWorkLog) lives in
// the service package, but the persisted WorkLog entity is the model type.
type WorkLogSaveSvc interface {
	SaveWorkLog(input service.SaveWorkLogInput) (*model.WorkLog, error)
}

// =====================================================================
// structure_worklog
// =====================================================================

// StructureWorklogTool turns a free-form brain-dump into 4-dimensional work
// items (content / problem_solved / result / impact) plus a summary. PermRead:
// the underlying StructureBrainDump does not persist anything, so the agent
// loop auto-executes it. After this step the user typically reviews the
// structured output and the agent calls save_worklog.
type StructureWorklogTool struct {
	Svc WorkLogStructureSvc
}

func (t *StructureWorklogTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "structure_worklog",
		Function: agent.FunctionSpec{
			Name:        "structure_worklog",
			Description: "Structure a free-form brain-dump into 4-dimensional work items (content/problem_solved/result/impact) plus a summary. Does not persist.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"brain_dump": map[string]any{"type": "string", "description": "free-form brain-dump of the day's work"},
				},
				"required": []any{"brain_dump"},
			},
		},
		Permission: agent.PermRead,
	}
}

func (t *StructureWorklogTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct {
		BrainDump string `json:"brain_dump"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	if in.BrainDump == "" {
		return nil, fmt.Errorf("schema: missing required field brain_dump")
	}
	if t.Svc == nil {
		return nil, fmt.Errorf("structure_worklog: WorkLog service not configured")
	}
	out, err := t.Svc.StructureBrainDump(service.BrainDumpInput{BrainDump: in.BrainDump})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (t *StructureWorklogTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	// PermRead: preview mirrors execute (auto-execute semantics — no side
	// effects, since StructureBrainDump does not persist).
	return t.Execute(ctx, args)
}

// =====================================================================
// save_worklog
// =====================================================================

// SaveWorklogTool persists a structured work log entry. PermWrite: requires
// confirmation. The agent typically chains this after structure_worklog.
//
// Divergence from the original brief: the brief called `Svc.Save(date, items)`
// with []map[string]any. The actual service method is
// WorkLogService.SaveWorkLog(service.SaveWorkLogInput) — SaveWorkLogInput
// already defines Date, Summary, RawBrainDump, and a typed []SaveItemInput
// (each item carries Seq/Title/Content/ProblemSolved/Result/Impact). We map
// the structured items argument directly into []service.SaveItemInput, which
// is cleaner than carrying raw maps through the boundary and lets the service
// validate types instead of decoding JSON a second time.
type SaveWorklogTool struct {
	Svc WorkLogSaveSvc
}

func (t *SaveWorklogTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "save_worklog",
		Function: agent.FunctionSpec{
			Name:        "save_worklog",
			Description: "Save a structured work log for a single date. Items are the 4-dimensional work items produced by structure_worklog (or supplied manually).",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"date":    map[string]any{"type": "string", "description": "YYYY-MM-DD"},
					"summary": map[string]any{"type": "string", "description": "one-line day summary (optional)"},
					"items": map[string]any{
						"type":        "array",
						"description": "4D work items",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"seq":            map[string]any{"type": "integer"},
								"title":          map[string]any{"type": "string"},
								"content":        map[string]any{"type": "string"},
								"problem_solved": map[string]any{"type": "string"},
								"result":         map[string]any{"type": "string"},
								"impact":         map[string]any{"type": "string"},
							},
						},
					},
				},
				"required": []any{"date", "items"},
			},
		},
		Permission: agent.PermWrite,
	}
}

func (t *SaveWorklogTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct {
		Date    string `json:"date"`
		Summary string `json:"summary"`
		Items   []struct {
			Seq           int    `json:"seq"`
			Title         string `json:"title"`
			Content       string `json:"content"`
			ProblemSolved string `json:"problem_solved"`
			Result        string `json:"result"`
			Impact        string `json:"impact"`
		} `json:"items"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	if in.Date == "" {
		return nil, fmt.Errorf("schema: missing required field date")
	}
	if in.Items == nil {
		return nil, fmt.Errorf("schema: missing required field items")
	}
	if t.Svc == nil {
		return nil, fmt.Errorf("save_worklog: WorkLog service not configured")
	}
	items := make([]service.SaveItemInput, 0, len(in.Items))
	for _, it := range in.Items {
		items = append(items, service.SaveItemInput{
			Seq:           it.Seq,
			Title:         it.Title,
			Content:       it.Content,
			ProblemSolved: it.ProblemSolved,
			Result:        it.Result,
			Impact:        it.Impact,
		})
	}
	input := service.SaveWorkLogInput{
		Date:    in.Date,
		Summary: in.Summary,
		Items:   items,
	}
	return t.Svc.SaveWorkLog(input)
}

func (t *SaveWorklogTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	// PermWrite: preview returns a plan WITHOUT calling the service. Best-effort
	// decode — preview must tolerate partial / malformed args so the user can
	// see what the agent intends to do.
	var in struct {
		Date  string `json:"date"`
		Items []struct {
			Seq int `json:"seq"`
		} `json:"items"`
	}
	_ = json.Unmarshal(args, &in)
	plan := map[string]any{
		"action":      "save_worklog",
		"date":        in.Date,
		"items_count": len(in.Items),
	}
	return plan, nil
}
