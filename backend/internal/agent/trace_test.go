package agent

import (
	"os"
	"testing"
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
