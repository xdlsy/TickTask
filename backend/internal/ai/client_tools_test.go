package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAIClient_ChatWithTools_BuildsRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if _, ok := body["tools"]; !ok {
			t.Errorf("request missing tools field")
		}
		resp := `{"choices":[{"message":{"role":"assistant","content":"hi","tool_calls":null},"finish_reason":"stop"}]}`
		w.Write([]byte(resp))
	}))
	defer srv.Close()
	// Existing constructor order: (apiKey, baseURL, model)
	c := NewOpenAIClient("k", srv.URL, "gpt-4o-mini")
	res, err := c.ChatWithTools(context.Background(),
		[]Message{{Role: "user", Content: "hi"}},
		[]ToolSpec{{Type: "function", Function: FunctionSpec{Name: "f"}}})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.Content != "hi" {
		t.Fatalf("content = %q", res.Content)
	}
	if res.FinishReason != "stop" {
		t.Fatalf("finish = %q", res.FinishReason)
	}
}

func TestOpenAIClient_ChatWithTools_ParsesToolCalls(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := `{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"c1","type":"function","function":{"name":"list_tasks","arguments":"{\"status\":\"todo\"}"}}]},"finish_reason":"tool_calls"}]}`
		w.Write([]byte(resp))
	}))
	defer srv.Close()
	c := NewOpenAIClient("k", srv.URL, "m")
	res, err := c.ChatWithTools(context.Background(),
		[]Message{{Role: "user", Content: "go"}}, nil)
	if err != nil {
		t.Fatalf("ChatWithTools: %v", err)
	}
	if len(res.ToolCalls) != 1 {
		t.Fatalf("tool calls = %d", len(res.ToolCalls))
	}
	tc := res.ToolCalls[0]
	if tc.ID != "c1" || tc.Name != "list_tasks" {
		t.Fatalf("tc = %+v", tc)
	}
	var args map[string]string
	if err := json.Unmarshal(tc.Args, &args); err != nil {
		t.Fatalf("unmarshal args: %v", err)
	}
	if args["status"] != "todo" {
		t.Fatalf("args = %+v", args)
	}
}

func TestCLIClient_ChatWithTools_Unsupported(t *testing.T) {
	c := CLIClient{Binary: "echo"}
	_, err := c.ChatWithTools(context.Background(), nil, nil)
	if !errors.Is(err, ErrFunctionCallNotSupported) {
		t.Fatalf("err = %v, want ErrFunctionCallNotSupported", err)
	}
}

func TestOpenAIClient_NetworkError_Retries(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	c := NewOpenAIClient("k", srv.URL, "m")
	_, err := c.ChatWithTools(context.Background(),
		[]Message{{Role: "user", Content: "x"}}, nil)
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("err = %v", err)
	}
	if calls < 2 {
		t.Errorf("expected at least 2 attempts, got %d", calls)
	}
}
