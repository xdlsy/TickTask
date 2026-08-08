package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"ticktask/internal/agent"
	"ticktask/internal/ai"
	"ticktask/internal/model"
)

// LLMClient is the subset of ai.LLMClient methods that classify_task needs.
// The production ai.OpenAIClient / AnthropicClient / CLIClient all satisfy
// this implicitly. Tools use the single-turn ChatCompletion (not ChatWithTools)
// because classify is a one-shot prompt → JSON reply.
type LLMClient interface {
	ChatCompletion(ctx context.Context, prompt string) (string, error)
}

// ClassifyTaskTool asks the LLM to classify a task into the Eisenhower matrix.
// The task can be referenced by ID (loaded via TaskService.GetTask) or supplied
// inline as title+description. PermRead: classification does not mutate state.
type ClassifyTaskTool struct {
	Svc TaskService
	LLM LLMClient
}

func (t *ClassifyTaskTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "classify_task",
		Function: agent.FunctionSpec{
			Name:        "classify_task",
			Description: "Ask the LLM to classify a task's importance/urgency and recommend an Eisenhower-matrix quadrant.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"task_id":     map[string]any{"type": "string", "description": "Existing task ID to classify"},
					"title":       map[string]any{"type": "string", "description": "Used when no task_id: inline title to classify"},
					"description": map[string]any{"type": "string", "description": "Optional inline description"},
				},
			},
		},
		Permission: agent.PermRead,
	}
}

func (t *ClassifyTaskTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct {
		TaskID      string  `json:"task_id"`
		Title       *string `json:"title"`
		Description *string `json:"description"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}

	// Resolve the task: either by ID (preferred) or synthesized from text.
	var task *model.Task
	if in.TaskID != "" {
		loaded, err := t.Svc.GetTask(in.TaskID)
		if err != nil {
			return nil, err
		}
		task = loaded
	} else if in.Title != nil {
		task = &model.Task{
			Title:       *in.Title,
			Description: derefStr(in.Description),
		}
	} else {
		return nil, fmt.Errorf("schema: must provide either task_id or title")
	}

	if t.LLM == nil {
		return nil, fmt.Errorf("AI service not configured")
	}

	// Build the prompt. Deadline is formatted as "无" (none) when absent,
	// matching the original ClassifyPrompt template's expectation.
	deadline := "无"
	if task.Deadline != nil {
		deadline = task.Deadline.Format("2006-01-02 15:04")
	}
	prompt := fmt.Sprintf(ai.ClassifyPrompt, task.Title, task.Description, deadline)

	response, err := t.LLM.ChatCompletion(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("AI request failed: %w", err)
	}

	parsed, err := parseClassifyResponse(response)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"task_id":   task.ID,
		"title":     task.Title,
		"important": parsed.Important,
		"urgent":    parsed.Urgent,
		"quadrant":  int(quadrantOf(parsed.Important, parsed.Urgent)),
		"reason":    parsed.Reason,
	}, nil
}

func (t *ClassifyTaskTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	return t.Execute(ctx, args)
}
