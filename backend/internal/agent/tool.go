package agent

import (
	"context"
	"encoding/json"
)

type ToolPermission int

const (
	PermRead ToolPermission = iota
	PermWrite
	PermDangerous
)

type FunctionSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type ToolSchema struct {
	Name       string
	Function   FunctionSpec
	Permission ToolPermission
}

type Tool interface {
	Schema() ToolSchema
	Execute(ctx context.Context, args json.RawMessage) (any, error)
	Preview(ctx context.Context, args json.RawMessage) (any, error)
}
