package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"ticktask/internal/ai"
)

// TraceRecorder observes a completed agent turn as a structured TurnTrace.
// Implementations must be safe for concurrent use across conversations.
// It is a nil-safe byproduct injected alongside HubBroadcaster: the noop
// default records nothing, so production and existing tests are unaffected
// unless a real tracer is wired in.
type TraceRecorder interface {
	RecordTurn(convID string, trace TurnTrace)
}

// TraceStep captures one tool-call decision plus its execution outcome. The
// ToolCall (model-requested name+args) is the decision evidence L2 asserts on;
// Status/Result/Error record what happened when it ran.
type TraceStep struct {
	ToolCall   ai.ToolCall
	Permission ToolPermission
	Status     string // started | pending_confirmation | rejected | succeeded | failed
	Result     any
	Error      string
}

// TurnTrace is the full record of one runTurn invocation: the user message
// that triggered it, every assistant text chunk concatenated, and one step per
// tool call. JSONL-serialized by jsonTracer; also the future VCR cassette shape.
type TurnTrace struct {
	ConversationID string
	UserText       string
	AssistantText  string
	Steps          []TraceStep
}

// noopTracer discards everything. NewAgentService injects it as the default so
// the field is never nil and existing call sites need no nil checks.
type noopTracer struct{}

func (noopTracer) RecordTurn(string, TurnTrace) {}

// jsonTracer appends one JSONL line per turn to <dir>/<conversation_id>.jsonl.
// Phase-1 observability sink + future record/replay cassette source. A mutex
// serializes appends to the same conversation file across concurrent turns.
type jsonTracer struct {
	dir string
	mu  sync.Mutex
}

func (j *jsonTracer) RecordTurn(convID string, trace TurnTrace) {
	trace.ConversationID = convID
	data, err := json.Marshal(trace)
	if err != nil {
		return
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	path := filepath.Join(j.dir, convID+".jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(data, '\n'))
}

// selectTracer picks a tracer by configuration: empty dir -> noop (default),
// non-empty -> jsonTracer writing to that directory. Wired in main.go from the
// TICKTASK_TRACE_DIR env var. (Phase 2 will add a langfuseTracer branch here.)
func selectTracer(dir string) TraceRecorder {
	if dir == "" {
		return noopTracer{}
	}
	return &jsonTracer{dir: dir}
}
