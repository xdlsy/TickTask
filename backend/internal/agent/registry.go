package agent

import (
	"errors"
)

var (
	ErrToolNotFound  = errors.New("tool not found")
	ErrDuplicateTool = errors.New("duplicate tool name")
)

type ToolRegistry interface {
	Register(t Tool)
	MustRegister(t Tool) // panics on duplicate (used at startup)
	Lookup(name string) (Tool, error)
	ToOpenAITools() []OpenAIToolSpec
	ListByPermission(p ToolPermission) []Tool
}

type OpenAIToolSpec struct {
	Type     string       `json:"type"`
	Function FunctionSpec `json:"function"`
}

type toolRegistry struct{ tools map[string]Tool }

func NewToolRegistry() ToolRegistry {
	return &toolRegistry{tools: make(map[string]Tool)}
}

func (r *toolRegistry) Register(t Tool) {
	r.tools[t.Schema().Name] = t
}

func (r *toolRegistry) MustRegister(t Tool) {
	name := t.Schema().Name
	if _, exists := r.tools[name]; exists {
		panic(ErrDuplicateTool)
	}
	r.tools[name] = t
}

func (r *toolRegistry) Lookup(name string) (Tool, error) {
	t, ok := r.tools[name]
	if !ok {
		return nil, ErrToolNotFound
	}
	return t, nil
}

func (r *toolRegistry) ToOpenAITools() []OpenAIToolSpec {
	specs := make([]OpenAIToolSpec, 0, len(r.tools))
	for _, t := range r.tools {
		s := t.Schema()
		// ToolSchema.Name is the canonical tool identifier; ensure OpenAI's
		// function.name mirrors it so the model can reference the tool.
		fn := s.Function
		if fn.Name == "" {
			fn.Name = s.Name
		}
		specs = append(specs, OpenAIToolSpec{Type: "function", Function: fn})
	}
	return specs
}

func (r *toolRegistry) ListByPermission(p ToolPermission) []Tool {
	out := []Tool{}
	for _, t := range r.tools {
		if t.Schema().Permission == p {
			out = append(out, t)
		}
	}
	return out
}
