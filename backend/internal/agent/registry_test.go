package agent

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

type fakeTool struct {
	name string
	perm ToolPermission
}

func (t *fakeTool) Schema() ToolSchema {
	return ToolSchema{
		Name: t.name,
		Function: FunctionSpec{
			Description: "fake",
			Parameters:  map[string]any{"type": "object"},
		},
		Permission: t.perm,
	}
}
func (t *fakeTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	return "ok:" + t.name, nil
}
func (t *fakeTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	return "preview:" + t.name, nil
}

func TestRegistry_RegisterAndLookup(t *testing.T) {
	r := NewToolRegistry()
	r.Register(&fakeTool{name: "t1", perm: PermRead})
	got, err := r.Lookup("t1")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	res, _ := got.Execute(context.Background(), nil)
	if res != "ok:t1" {
		t.Fatalf("res = %v", res)
	}
}

func TestRegistry_LookupUnknown(t *testing.T) {
	r := NewToolRegistry()
	_, err := r.Lookup("xxx")
	if !errors.Is(err, ErrToolNotFound) {
		t.Fatalf("err = %v, want ErrToolNotFound", err)
	}
}

func TestRegistry_DuplicateRegister(t *testing.T) {
	r := NewToolRegistry()
	r.Register(&fakeTool{name: "t1", perm: PermRead})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate MustRegister")
		}
	}()
	r.MustRegister(&fakeTool{name: "t1", perm: PermRead})
}

func TestRegistry_ToOpenAITools(t *testing.T) {
	r := NewToolRegistry()
	r.Register(&fakeTool{name: "t1", perm: PermRead})
	specs := r.ToOpenAITools()
	if len(specs) != 1 {
		t.Fatalf("got %d specs", len(specs))
	}
	if specs[0].Type != "function" || specs[0].Function.Name != "t1" {
		t.Fatalf("spec malformed: %+v", specs[0])
	}
}

func TestRegistry_ListByPermission(t *testing.T) {
	r := NewToolRegistry()
	r.Register(&fakeTool{name: "r1", perm: PermRead})
	r.Register(&fakeTool{name: "w1", perm: PermWrite})
	r.Register(&fakeTool{name: "d1", perm: PermDangerous})
	if len(r.ListByPermission(PermRead)) != 1 {
		t.Fatal("read count")
	}
}
