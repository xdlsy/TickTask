package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"ticktask/internal/ai"
)

func TestSelectTracer_DefaultsToNoop(t *testing.T) {
	got := selectTracer("")
	if _, ok := got.(noopTracer); !ok {
		t.Fatalf("selectTracer(\"\") = %T, want noopTracer", got)
	}
}

func TestSelectTracer_NonEmptyReturnsJsonTracer(t *testing.T) {
	got := selectTracer(t.TempDir())
	jt, ok := got.(*jsonTracer)
	if !ok {
		t.Fatalf("selectTracer(dir) = %T, want *jsonTracer", got)
	}
	if jt.dir == "" {
		t.Fatal("jsonTracer.dir is empty")
	}
}

// noopTracer.RecordTurn must be a no-op (no panic, no file writes).
func TestNoopTracer_DoesNotPanic(t *testing.T) {
	noopTracer{}.RecordTurn("any", TurnTrace{UserText: "x"})
	dir := t.TempDir()
	selectTracer("").(noopTracer).RecordTurn("c", TurnTrace{})
	if ents, _ := os.ReadDir(dir); len(ents) != 0 {
		t.Fatalf("noop wrote files: %v", ents)
	}
}

func TestJsonTracer_AppendsOneLinePerTurn(t *testing.T) {
	dir := t.TempDir()
	jt := selectTracer(dir).(*jsonTracer)

	jt.RecordTurn("conv-1", TurnTrace{
		UserText:       "我一会有啥安排吗？",
		AssistantText:  "你今天有 3 个安排",
		Steps:          []TraceStep{{ToolCall: ai.ToolCall{ID: "c1", Name: "list_schedule", Args: json.RawMessage(`{"from":"2026-08-09","to":"2026-08-09"}`)}, Permission: PermRead, Status: "succeeded"}},
	})
	jt.RecordTurn("conv-1", TurnTrace{UserText: "第二问"})

	data, err := os.ReadFile(filepath.Join(dir, "conv-1.jsonl"))
	if err != nil {
		t.Fatalf("read trace file: %v", err)
	}
	lines := splitLines(string(data))
	if len(lines) != 2 {
		t.Fatalf("want 2 JSONL lines, got %d: %q", len(lines), string(data))
	}

	var first TurnTrace
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("unmarshal line 1: %v", err)
	}
	if first.UserText != "我一会有啥安排吗？" || first.AssistantText != "你今天有 3 个安排" {
		t.Errorf("line 1 trace = %+v", first)
	}
	if len(first.Steps) != 1 || first.Steps[0].ToolCall.Name != "list_schedule" || first.Steps[0].Status != "succeeded" {
		t.Errorf("line 1 steps = %+v", first.Steps)
	}
	if first.ConversationID != "conv-1" {
		t.Errorf("ConversationID = %q, want conv-1", first.ConversationID)
	}
}

// separate conversations write separate files
func TestJsonTracer_PerConversationFile(t *testing.T) {
	dir := t.TempDir()
	jt := selectTracer(dir).(*jsonTracer)
	jt.RecordTurn("a", TurnTrace{UserText: "A"})
	jt.RecordTurn("b", TurnTrace{UserText: "B"})
	for _, id := range []string{"a", "b"} {
		if _, err := os.Stat(filepath.Join(dir, id+".jsonl")); err != nil {
			t.Errorf("missing %s.jsonl: %v", id, err)
		}
	}
}

func TestSelectTracerFromEnv(t *testing.T) {
	if _, ok := SelectTracerFromEnv(func(string) string { return "" }).(noopTracer); !ok {
		t.Fatal("empty env should yield noopTracer")
	}
	if _, ok := SelectTracerFromEnv(func(string) string { return t.TempDir() }).(*jsonTracer); !ok {
		t.Fatal("set env should yield *jsonTracer")
	}
}

func splitLines(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == '\n' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
